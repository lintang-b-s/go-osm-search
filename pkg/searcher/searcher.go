package searcher

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/geo"
	"github.com/lintang-b-s/osm-search/pkg/index"

	"github.com/RadhiFadlillah/go-sastrawi"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type SimiliarityScoring int

const (
	TF_IDF_COSINE SimiliarityScoring = iota
	BM25_PLUS
	BM25_FIELD
)

// BM25+ parameter
const (
	DELTA = 1.0
	K1    = 1.2
	B     = 0.98
	// param BM25F
	K1_BM25F       = 10
	NAME_WEIGHT    = 20
	ADDRESS_WEIGHT = 1
	NAME_B         = 0.95
	ADDRESS_B      = 0.3
)

type DynamicIndexer interface {
	GetOutputDir() string
	GetWorkingDir() string
	GetDocWordCount() map[int]int
	GetDocsCount() int
	GetTermIDMap() *pkg.IDMap
	GetAverageDocLength() float64
	BuildVocabulary()
	GetOSMFeatureMap() *pkg.IDMap
}

type SearcherDocStore interface {
	GetDoc(docID int) (datastructure.Node, error)
}

type InvertedIndexI interface {
	Close() error
	GetPostingList(termID int) ([]int, error)
	GetLenFieldInDoc() map[int]int
	GetAverageFieldLength() float64
}

type RtreeI interface {
	ImprovedNearestNeighbor(p datastructure.Point) datastructure.OSMObject
	Search(bound  datastructure.RtreeBoundingBox) []datastructure.RtreeNode 
	NearestNeighboursRadiusFilterOSM(k int, offfset int, p datastructure.Point, maxRadius float64, osmFeature int) []datastructure.OSMObject
}

type Searcher struct {
	Idx                   DynamicIndexer
	MainIndexNameField    InvertedIndexI
	MainIndexAddressField InvertedIndexI
	SpellCorrector        index.SpellCorrectorI
	TermIDMap             *pkg.IDMap
	DocStore              SearcherDocStore
	osmRtree              RtreeI
	similiarityScoring    SimiliarityScoring
}

func NewSearcher(idx DynamicIndexer, docStore SearcherDocStore, spell index.SpellCorrectorI,
	scoring SimiliarityScoring) *Searcher {

	return &Searcher{Idx: idx, DocStore: docStore, SpellCorrector: spell, similiarityScoring: scoring}
}

func (se *Searcher) LoadMainIndex() error {
	bar := progressbar.NewOptions(5,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][1/3]Loading Inverted & R-tree Index..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	fmt.Println("")

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	mainIndexNameField := index.NewInvertedIndex("merged_name_index", se.Idx.GetOutputDir(), pwd)
	err = mainIndexNameField.OpenReader()
	if err != nil {
		return err
	}
	se.MainIndexNameField = mainIndexNameField

	mainIndexAddressField := index.NewInvertedIndex("merged_address_index", se.Idx.GetOutputDir(), pwd)
	err = mainIndexAddressField.OpenReader()
	if err != nil {
		return err
	}
	se.MainIndexAddressField = mainIndexAddressField
	bar.Add(1)

	// build vocabulary
	se.Idx.BuildVocabulary()
	se.TermIDMap = se.Idx.GetTermIDMap()
	bar.Add(1)

	// load r*-tree
	rt := datastructure.NewRtree(25, 50, 2)
	err = rt.Deserialize(se.Idx.GetWorkingDir(), se.Idx.GetOutputDir())
	if err != nil {
		return fmt.Errorf("error when deserialize rtree: %w", err)
	}
	se.osmRtree = rt
	bar.Add(1)
	return nil
}

func (se *Searcher) Close() error {
	err := se.MainIndexNameField.Close()
	if err != nil {
		return err
	}

	err = se.MainIndexAddressField.Close()
	return err
}

type DocWithScore struct {
	DocID int
	Score float64
}

func (se *Searcher) FreeFormQuery(query string, k, offset int) ([]datastructure.Node, error) {
	if query == "" {
		return []datastructure.Node{}, errors.New("query is empty")
	}
	if k == 0 {
		k = 10
	}

	queryTerms := sastrawi.Tokenize(query)

	queryTermsID := make([]int, 0, len(queryTerms))

	// {{term1,term1OneEdit}, {term2, term2Edit}, ...}
	allPossibleQueryTerms := make([][]int, len(queryTerms))

	queryWordCount := make(map[int]int, len(queryTerms))

	for i, tokenizedTerm := range queryTerms {
		isInVocab := se.TermIDMap.IsInVocabulary(tokenizedTerm)

		if !isInVocab {

			correctionOne, err := se.SpellCorrector.GetWordCandidates(tokenizedTerm, 1)
			if err != nil {
				return []datastructure.Node{}, err
			}
			correctionTwo, err := se.SpellCorrector.GetWordCandidates(tokenizedTerm, 2)
			if err != nil {
				return []datastructure.Node{}, err
			}
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], correctionOne...)
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], correctionTwo...)
		} else {
			termID := se.TermIDMap.GetID(tokenizedTerm)
			allPossibleQueryTerms[i] = []int{termID}
		}
	}

	allCorrectQueryCandidates := se.SpellCorrector.GetCorrectQueryCandidates(allPossibleQueryTerms)
	correctQuery, err := se.SpellCorrector.GetCorrectSpellingSuggestion(allCorrectQueryCandidates)

	if err != nil {
		return []datastructure.Node{}, err
	}

	queryTermsID = append(queryTermsID, correctQuery...)

	allPostingsNameField := make(map[int][]int, len(queryTerms))
	allPostingsAddressField := make(map[int][]int, len(queryTerms))

	for _, termID := range queryTermsID {
		postings, err := se.MainIndexNameField.GetPostingList(termID)
		if err != nil {
			return []datastructure.Node{}, err
		}
		postingsAddress, err := se.MainIndexAddressField.GetPostingList(termID)
		if err != nil {
			return []datastructure.Node{}, err
		}
		allPostingsNameField[termID] = postings
		allPostingsAddressField[termID] = postingsAddress
		queryWordCount[termID] += 1
	}

	docWithScores := []int{}
	switch se.similiarityScoring {
	case TF_IDF_COSINE:
		for termID, postings := range allPostingsAddressField {
			allPostingsNameField[termID] = append(allPostingsNameField[termID], postings...)
		}
		docWithScores = se.scoreTFIDFCosine(allPostingsNameField, queryWordCount)
	case BM25_PLUS:
		for termID, postings := range allPostingsAddressField {
			allPostingsNameField[termID] = append(allPostingsNameField[termID], postings...)
		}
		docWithScores = se.scoreBM25Plus(allPostingsNameField)
	case BM25_FIELD:
		docWithScores = se.scoreBM25Field(allPostingsNameField, allPostingsAddressField, queryTermsID)
	}

	relevantDocs := make([]datastructure.Node, 0, k)

	for i := offset; i < len(docWithScores); i++ {

		if i >= k+offset {
			break
		}

		doc, err := se.DocStore.GetDoc(docWithScores[i])
		if err != nil {
			return []datastructure.Node{}, err
		}
		relevantDocs = append(relevantDocs, doc)
	}

	return relevantDocs, nil
}

// https://trec.nist.gov/pubs/trec13/papers/microsoft-cambridge.web.hard.pdf
func (se *Searcher) scoreBM25Field(allPostingsNameField map[int][]int,
	allPostingsAddressField map[int][]int, allQueryTermIDs []int) []int {

	documentScore := make(map[int]float64)

	docCount := float64(se.Idx.GetDocsCount())

	nameLenDF := se.MainIndexNameField.GetLenFieldInDoc()
	addressLenDF := se.MainIndexAddressField.GetLenFieldInDoc()
	averageNameLenDF := se.MainIndexNameField.GetAverageFieldLength()
	averageAddressLenDF := se.MainIndexAddressField.GetAverageFieldLength()

	for _, qTermID := range allQueryTermIDs {

		namePostingsList, ok := allPostingsNameField[qTermID]
		addressPostingsList, ok := allPostingsAddressField[qTermID]

		uniqueDocContainingTerm := make(map[int]struct{}, len(namePostingsList)+len(addressPostingsList))

		// name field
		tfTermDocNameField := make(map[int]float64, len(namePostingsList))

		if ok {
			for _, docID := range namePostingsList {
				tfTermDocNameField[docID]++ // conunt(t,d)
				uniqueDocContainingTerm[docID] = struct{}{}
			}
		}

		// address field

		tfTermDocAddressField := make(map[int]float64, len(addressPostingsList))

		if ok {
			for _, docID := range addressPostingsList {
				tfTermDocAddressField[docID]++ // conunt(t,d)
				uniqueDocContainingTerm[docID] = struct{}{}
			}
		}

		// score untuk doc yang include term di name field

		idf := math.Log10(docCount-float64(len(uniqueDocContainingTerm))+0.5) - math.Log10(float64(len(uniqueDocContainingTerm))+0.5) // log(N-df_t+0.5/df_t+0.5)

		for docID, tftd := range tfTermDocNameField {
			weightTD := NAME_WEIGHT * (tftd / (1 + NAME_B*((float64(nameLenDF[docID])/averageNameLenDF)-1)))
			documentScore[docID] += (weightTD / (K1_BM25F + weightTD)) * idf
		}

		for docID, tftd := range tfTermDocAddressField {
			weightTD := ADDRESS_WEIGHT * (tftd / (1 + NAME_B*((float64(addressLenDF[docID])/averageAddressLenDF)-1)))
			documentScore[docID] += (weightTD / (K1_BM25F + weightTD)) * idf
		}

	}

	documentIDs := make([]int, 0, len(documentScore))
	for k := range documentScore {
		documentIDs = append(documentIDs, k)
	}

	sort.SliceStable(documentIDs, func(i, j int) bool {
		return documentScore[documentIDs[i]] > documentScore[documentIDs[j]]
	})

	return documentIDs
}

func (se *Searcher) scoreBM25Plus(allPostingsField map[int][]int) []int {
	// param bm25+

	documentScore := make(map[int]float64)

	docsCount := float64(se.Idx.GetDocsCount())
	docWordCount := se.Idx.GetDocWordCount()

	avgDocLength := se.Idx.GetAverageDocLength()

	for _, postings := range allPostingsField {

		tfTermDoc := make(map[int]float64)
		for _, docID := range postings {
			tfTermDoc[docID]++ // conunt(t,d)
		}

		idf := math.Log10(docsCount+1) - math.Log10(float64(len(tfTermDoc))) // log(N/df_t)

		for docID, tftd := range tfTermDoc {
			// https://www.cs.otago.ac.nz/homepages/andrew/papers/2014-2.pdf

			documentScore[docID] += idf * (DELTA +
				((K1+1)+tftd)/(K1*(1-B+B*float64(docWordCount[docID])/avgDocLength)+tftd))
		}
	}

	documentIDs := make([]int, 0, len(documentScore))
	for k := range documentScore {
		documentIDs = append(documentIDs, k)
	}

	sort.SliceStable(documentIDs, func(i, j int) bool {
		return documentScore[documentIDs[i]] > documentScore[documentIDs[j]]
	})

	return documentIDs
}

func (se *Searcher) scoreTFIDFCosine(allPostings map[int][]int,
	queryWordCount map[int]int) []int {
	documentScore := make(map[int]float64) // menyimpan skor cosine tf-idf docs \dot tf-idf query

	docsCount := float64(se.Idx.GetDocsCount())
	docNorm := make(map[int]float64)
	queryNorm := 0.0
	for qTermID, postings := range allPostings {
		// iterate semua term di query, hitung tf-idf query dan tf-idf document, accumulate skor cosine di docScore

		termCountInDoc := make(map[int]int)
		for _, docID := range postings {
			termCountInDoc[docID]++ // conunt(t,d)
		}

		tfTermQuery := 1 + math.Log10(float64(queryWordCount[qTermID]))                  //  1 + log(count(t,q))
		idfTermQuery := math.Log10(docsCount) - math.Log10(float64(len(termCountInDoc))) // log(N/df_t)
		tfIDFTermQuery := tfTermQuery * idfTermQuery

		for docID, termCount := range termCountInDoc {
			tf := 1 + math.Log10(float64(termCount)) //  //  1 + log(count(t,d))

			tfIDFTermDoc := tf * idfTermQuery //tfidf docID

			documentScore[docID] += tfIDFTermDoc * tfIDFTermQuery // summation tfidfDoc*tfIDfquery over query terms

			docNorm[docID] += tfIDFTermDoc * tfIDFTermDoc // document Norm
		}

		queryNorm += tfIDFTermQuery * tfIDFTermQuery
	}

	queryNorm = math.Sqrt(queryNorm)

	documentIDs := make([]int, 0, len(documentScore))
	for k := range documentScore {
		documentIDs = append(documentIDs, k)
	}

	sort.SliceStable(documentIDs, func(i, j int) bool {
		return documentScore[documentIDs[i]] > documentScore[documentIDs[j]]
	})

	return documentIDs
}

func (se *Searcher) Autocomplete(query string, k, offset int) ([]datastructure.Node, error) {
	if query == "" {
		return []datastructure.Node{}, errors.New("query is empty")
	}

	if k == 0 {
		k = 10
	}

	queryTerms := sastrawi.Tokenize(query)

	// {{term1,term1OneEdit}, {term2, term2Edit}, ...}
	allPossibleQueryTerms := make([][]int, len(queryTerms))
	originalQueryTerms := make([]int, 0, len(queryTerms))

	for i, tokenizedTerm := range queryTerms {

		originalQueryTerms = append(originalQueryTerms, se.TermIDMap.GetID(tokenizedTerm))

		if i == len(queryTerms)-1 {

			matchedWord, err := se.SpellCorrector.GetMatchedWordBasedOnPrefix(tokenizedTerm)
			if err != nil {
				return []datastructure.Node{}, err
			}

			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], matchedWord...)

		} else {
			termID := se.TermIDMap.GetID(tokenizedTerm)
			allPossibleQueryTerms[i] = []int{termID}
		}
	}

	allCorrectQueryCandidates := se.SpellCorrector.GetCorrectQueryCandidates(allPossibleQueryTerms)
	matchedQueries, err := se.SpellCorrector.GetMatchedWordsAutocomplete(allCorrectQueryCandidates, originalQueryTerms)

	if err != nil {
		return []datastructure.Node{}, err
	}

	relDocIDs := []int{}
	for _, queryTerms := range matchedQueries {

		allPostings := make(map[int][]int, len(queryTerms))

		tokens := make([]int, 0, len(queryTerms)-1)
		for j, termID := range queryTerms {
			tokens = append(tokens, termID)
			if j != len(queryTerms)-1 {
				tokens = append(tokens, -1) // AND
			}

			postings, err := se.MainIndexNameField.GetPostingList(termID)
			if err != nil {
				return []datastructure.Node{}, err
			}

			allPostings[termID] = postings
		}

		// shunting Yard
		rpnDeque := NewDeque(shuntingYardRPN(tokens))
		docIDsRes, err := se.processQuery(rpnDeque)
		if err != nil {
			return []datastructure.Node{}, err
		}

		scoredDocs := se.scoreBM25FAutocomplete(allPostings, queryTerms, docIDsRes)

		relDocIDs = append(relDocIDs, scoredDocs...)

	}

	relevantDocs := make([]datastructure.Node, 0, len(relDocIDs))

	for i := offset; i < len(relDocIDs); i++ {
		if i >= k+offset {
			break
		}

		doc, err := se.DocStore.GetDoc(relDocIDs[i])
		if err != nil {
			return []datastructure.Node{}, err
		}
		relevantDocs = append(relevantDocs, doc)
	}

	return relevantDocs, nil
}

func (se *Searcher) scoreBM25FAutocomplete(allPostings map[int][]int, queryTermIDs []int,
	intersectedDocIDs []int) []int {

	documentScore := make(map[int]float64)

	docsCount := float64(se.Idx.GetDocsCount())

	nameLenDF := se.MainIndexNameField.GetLenFieldInDoc()
	averageNameLenDF := se.MainIndexNameField.GetAverageFieldLength()

	for _, postings := range allPostings {

		tfTermDoc := make(map[int]float64)
		for _, docID := range postings {
			tfTermDoc[docID]++ // conunt(t,d)
		}

		idf := math.Log10(docsCount-float64(len(tfTermDoc))+0.5) - math.Log10(float64(len(tfTermDoc))+0.5) // log(N-df_t+0.5/df_t+0.5)

		for i := 0; i < len(intersectedDocIDs); i++ {
			docID := intersectedDocIDs[i]
			tftd := tfTermDoc[docID]

			weightTD := NAME_WEIGHT * (tftd / (1 + NAME_B*((float64(nameLenDF[docID])/averageNameLenDF)-1)))
			documentScore[docID] += (weightTD / (K1_BM25F + weightTD)) * idf

		}
	}

	documentIDs := make([]int, 0, len(documentScore))
	for k := range documentScore {
		documentIDs = append(documentIDs, k)
	}

	sort.SliceStable(documentIDs, func(i, j int) bool {
		return documentScore[documentIDs[i]] > documentScore[documentIDs[j]]
	})

	return documentIDs
}

type Deque struct {
	items []int
}

func NewDeque(items []int) Deque {
	return Deque{items}
}

func (d *Deque) GetSize() int {
	return len(d.items)
}

func (d *Deque) PushFront(item int) {
	d.items = append([]int{item}, d.items...)
}

func (d *Deque) PushBack(item int) {
	d.items = append(d.items, item)
}

func (d *Deque) PopFront() (int, bool) {
	if len(d.items) == 0 {
		return 0, false
	}
	frontElement := d.items[0]
	d.items = d.items[1:]
	return frontElement, true
}

func (d *Deque) PopBack() (int, bool) {
	if len(d.items) == 0 {
		return 0, false
	}
	rearElement := d.items[len(d.items)-1]
	d.items = d.items[:len(d.items)-1]
	return rearElement, true
}

func shuntingYardRPN(tokens []int) []int {
	precedence := make(map[int]int)
	precedence[-1] = 2 // AND
	precedence[-2] = 0 // (
	precedence[-3] = 0 // )
	precedence[-4] = 1 // OR
	precedence[-5] = 3 // NOT

	output := make([]int, 0, len(tokens))
	stack := []int{}

	for _, token := range tokens {
		if token == -2 {
			stack = append(stack, -2)
		} else if token == -3 {
			// pop
			n := len(stack) - 1
			operator := stack[n]
			stack = stack[:n]

			for operator != -2 {
				output = append(output, operator)
				// pop
				n = len(stack) - 1
				operator = stack[n]
				stack = stack[:n]
			}
		} else if _, ok := precedence[token]; ok {
			if len(stack) != 0 {
				n := len(stack) - 1
				operator := stack[n]

				for len(stack) != 0 && precedence[token] < precedence[operator] {
					output = append(output, operator)

					n = len(stack) - 1
					stack = stack[:n]
					if len(stack) != 0 {
						n = len(stack) - 1
						operator = stack[n]
					}
				}
			}

			stack = append(stack, token)
		} else {
			// term
			output = append(output, token)
		}
	}

	for len(stack) != 0 {
		n := len(stack) - 1
		token := stack[n]
		stack = stack[:n]
		output = append(output, token)
	}
	return output
}

// processQuery. process query -> return hasil boolean query (AND/OR/NOT) berupa posting lists (docIDs)
func (se *Searcher) processQuery(rpnDeque Deque) ([]int, error) {
	operator := map[int]struct{}{
		-1: struct{}{},
		-5: struct{}{},
		-4: struct{}{},
	}
	postingListStack := [][]int{}
	for rpnDeque.GetSize() != 0 {
		token, valid := rpnDeque.PopFront()
		if !valid {
			return []int{}, fmt.Errorf("rpn deque size is 0")
		}

		if _, ok := operator[token]; !ok {
			postingList, err := se.MainIndexNameField.GetPostingList(token)
			if err != nil {
				return []int{}, fmt.Errorf("error when get posting list skip list: %w", err)
			}
			postingListStack = append(postingListStack, postingList)
		} else {

			if token == -1 {
				// AND
				right := postingListStack[len(postingListStack)-1]
				postingListStack = postingListStack[:len(postingListStack)-1]
				left := postingListStack[len(postingListStack)-1]
				postingListStack = postingListStack[:len(postingListStack)-1]

				postingListIntersection := PostingListIntersection2(left, right)

				postingListStack = append(postingListStack, postingListIntersection)
			} else if token == -4 {
				// OR
				// NOT IMPLEMENTED YET
			} else {
				// NOT
				// NOT IMPLEMENTED YET
			}
		}
	}

	docIDsResult := postingListStack[len(postingListStack)-1]

	return docIDsResult, nil
}

func (se *Searcher) ReverseGeocoding(lat, lon float64) (datastructure.Node, error) {
	upRightLat, upRightLon := geo.GetDestinationPoint(lat, lon, 45, 0.4)
	downLeftLat, downLeftLon := geo.GetDestinationPoint(lat, lon, 225, 0.4)
	boundingBox := datastructure.NewRtreeBoundingBox(2,[]float64{downLeftLat, downLeftLon}, []float64{upRightLat, upRightLon})
	nearbyOsmObjects := se.osmRtree.Search(boundingBox)

	nearestOsmObject := -1
	minDist := math.MaxFloat64
	for _, osmObject := range nearbyOsmObjects {
		distance := datastructure.HaversineDistance(lat, lon, osmObject.Leaf.Lat, osmObject.Leaf.Lon)
		if distance < minDist {
			minDist = distance
			nearestOsmObject = osmObject.Leaf.ID	
		}
	}


	doc, err := se.DocStore.GetDoc(nearestOsmObject)
	if err != nil {
		return datastructure.Node{}, fmt.Errorf("error when get doc: %w", err)
	}
	return doc, nil
}

func (se *Searcher) NearestNeighboursRadiusWithFeatureFilter(k, offset int, lat, lon, radius float64, featureType string) ([]datastructure.Node, error) {
	osmFeatureMap := se.Idx.GetOSMFeatureMap()
	result := se.osmRtree.NearestNeighboursRadiusFilterOSM(k, offset, datastructure.NewPoint(lat, lon), radius, osmFeatureMap.GetID(featureType))
	docs := []datastructure.Node{}
	for _, r := range result {
		doc, err := se.DocStore.GetDoc(r.ID)
		if err != nil {
			return []datastructure.Node{}, fmt.Errorf("error when get doc: %w", err)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func PostingListIntersection2(a, b []int) []int {

	idx1, idx2 := 0, 0
	result := []int{}

	for idx1 < len(a) && idx2 < len(b) {
		if a[idx1] < b[idx2] {
			idx1++
		} else if b[idx2] < a[idx1] {
			idx2++
		} else {
			result = append(result, a[idx1])
			idx1++
			idx2++
		}
	}
	return result
}
