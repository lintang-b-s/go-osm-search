package searcher

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/geo"
	"github.com/lintang-b-s/osm-search/pkg/index"

	"github.com/RadhiFadlillah/go-sastrawi"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

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

type docWithScore struct {
	DocID int
	Score float64
}

func newDocWithScore(docID int, score float64) docWithScore {
	return docWithScore{
		DocID: docID,
		Score: score,
	}
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
	allPossibleQueryTerms := make([][]datastructure.WordCandidate, len(queryTerms))

	queryWordCount := make(map[int]int, len(queryTerms))

	for i, tokenizedTerm := range queryTerms {
		isInVocab := se.TermIDMap.IsInVocabulary(tokenizedTerm)

		if !isInVocab {

			correctionOne, correctionOneString, err := se.SpellCorrector.GetWordCandidates(tokenizedTerm, 1)
			if err != nil {
				return []datastructure.Node{}, err
			}
			correctionTwo, correctionTwoString, err := se.SpellCorrector.GetWordCandidates(tokenizedTerm, 2)
			if err != nil {
				return []datastructure.Node{}, err
			}

			wordCandidates := make([]datastructure.WordCandidate, 0, len(correctionOne))
			for i, correction := range correctionOne {
				wordCandidates = append(wordCandidates, datastructure.NewWordCandidate(correction, tokenizedTerm, correctionOneString[i]))
			}

			wordCandidatesTwo := make([]datastructure.WordCandidate, 0, len(correctionTwo))
			for i, correction := range correctionTwo {
				wordCandidatesTwo = append(wordCandidatesTwo, datastructure.NewWordCandidate(correction, tokenizedTerm, correctionTwoString[i]))
			}

			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], wordCandidates...)
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], wordCandidatesTwo...)

		} else {
			termID := se.TermIDMap.GetID(tokenizedTerm)
			allPossibleQueryTerms[i] = []datastructure.WordCandidate{datastructure.NewWordCandidate(termID, tokenizedTerm, tokenizedTerm)}
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

func (se *Searcher) Autocomplete(query string, k, offset int) ([]datastructure.Node, error) {
	if query == "" {
		return []datastructure.Node{}, errors.New("query is empty")
	}

	if k == 0 {
		k = 10
	}

	queryTerms := sastrawi.Tokenize(query)

	// {{term1,term1OneEdit}, {term2, term2Edit}, ...}
	allPossibleQueryTerms := make([][]datastructure.WordCandidate, len(queryTerms))
	originalQueryTerms := make([]int, 0, len(queryTerms))

	for i, tokenizedTerm := range queryTerms {

		originalQueryTerms = append(originalQueryTerms, se.TermIDMap.GetID(tokenizedTerm))
		isInVocab := se.TermIDMap.IsInVocabulary(tokenizedTerm)

		if i == len(queryTerms)-1 || !isInVocab {
			var (
				wg                  sync.WaitGroup
				matchedWord         []int
				correctionOne       []int
				correctionTwo       []int
				correctionOneString []string
				correctionTwoString []string
				errChan             chan error
			)
			wg.Add(3)
			errChan = make(chan error, 3)

			// regex prefix search
			go func() {
				defer wg.Done()
				var err error
				matchedWord, err = se.SpellCorrector.GetMatchedWordBasedOnPrefix(tokenizedTerm)
				if err != nil {
					errChan <- err
				}
			}()

			// spell corrector

			go func() {
				defer wg.Done()
				var err error
				correctionOne, correctionOneString, err = se.SpellCorrector.GetWordCandidates(tokenizedTerm, 1)
				if err != nil {
					errChan <- err
				}

			}()

			go func() {
				defer wg.Done()
				var err error
				correctionTwo, correctionTwoString, err = se.SpellCorrector.GetWordCandidates(tokenizedTerm, 2)
				if err != nil {
					errChan <- err
				}
			}()

			go func() {
				wg.Wait()
				close(errChan)
			}()

			for err := range errChan {
				if err != nil {
					return []datastructure.Node{}, err
				}
			}

			// collect matched word

			// prefix
			matchedWordCandidates := make([]datastructure.WordCandidate, len(matchedWord))
			for j, autocompleteWordID := range matchedWord {
				matchedWordCandidates[j] = datastructure.NewWordCandidate(autocompleteWordID, tokenizedTerm, tokenizedTerm)
			}

			// correction one
			wordCandidates := make([]datastructure.WordCandidate, 0, len(correctionOne))
			for i, correction := range correctionOne {
				wordCandidates = append(wordCandidates, datastructure.NewWordCandidate(correction, tokenizedTerm, correctionOneString[i]))
			}

			// correction two
			wordCandidatesTwo := make([]datastructure.WordCandidate, 0, len(correctionTwo))
			for i, correction := range correctionTwo {
				wordCandidatesTwo = append(wordCandidatesTwo, datastructure.NewWordCandidate(correction, tokenizedTerm, correctionTwoString[i]))
			}

			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], matchedWordCandidates...)
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], wordCandidates...)
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], wordCandidatesTwo...)

		} else {
			termID := se.TermIDMap.GetID(tokenizedTerm)
			allPossibleQueryTerms[i] = []datastructure.WordCandidate{datastructure.NewWordCandidate(termID, tokenizedTerm, tokenizedTerm)}
		}
	}

	allCorrectQueryCandidates := se.SpellCorrector.GetCorrectQueryCandidates(allPossibleQueryTerms)
	matchedQueries, err := se.SpellCorrector.GetMatchedWordsAutocomplete(allCorrectQueryCandidates, originalQueryTerms)

	if err != nil {
		return []datastructure.Node{}, err
	}

	var (
		wg                sync.WaitGroup
		errChan           chan error
		docWithScoresChan = make(chan []docWithScore, len(matchedQueries))
	)
	wg.Add(len(matchedQueries))

	relDocIDs := []docWithScore{}
	for _, queryTerms := range matchedQueries {
		go func(queryTerms []int) {
			defer wg.Done()

			allPostingsNameField := make(map[int][]int, len(queryTerms))
			allPostingsAddressField := make(map[int][]int, len(queryTerms))
			queryWordCount := make(map[int]int, len(queryTerms))

			for _, termID := range queryTerms {
				postings, err := se.MainIndexNameField.GetPostingList(termID)
				if err != nil {
					errChan <- err
					return
				}
				postingsAddress, err := se.MainIndexAddressField.GetPostingList(termID)
				if err != nil {
					errChan <- err
					return
				}
				allPostingsNameField[termID] = postings
				allPostingsAddressField[termID] = postingsAddress
				queryWordCount[termID] += 1
			}

			docWithScoresChan <- se.scoreBM25FieldWithScores(allPostingsNameField, allPostingsAddressField, queryTerms)

		}(queryTerms)
	}
	go func() {
		wg.Wait()
		close(docWithScoresChan)
	}()

	for docs := range docWithScoresChan {
		relDocIDs = append(relDocIDs, docs...)

	}

	sort.Slice(relDocIDs, func(i, j int) bool {
		return relDocIDs[i].Score > relDocIDs[j].Score
	})

	relevantDocs := make([]datastructure.Node, 0, len(relDocIDs))

	for i := offset; i < len(relDocIDs); i++ {
		if i >= k+offset {
			break
		}

		doc, err := se.DocStore.GetDoc(relDocIDs[i].DocID)
		if err != nil {
			return []datastructure.Node{}, err
		}
		relevantDocs = append(relevantDocs, doc)
	}

	return relevantDocs, nil
}

func (se *Searcher) ReverseGeocoding(lat, lon float64) (datastructure.Node, error) {
	upRightLat, upRightLon := geo.GetDestinationPoint(lat, lon, 45, 0.4)
	downLeftLat, downLeftLon := geo.GetDestinationPoint(lat, lon, 225, 0.4)
	boundingBox := datastructure.NewRtreeBoundingBox(2, []float64{downLeftLat, downLeftLon}, []float64{upRightLat, upRightLon})
	nearbyOsmObjects := se.osmRtree.Search(boundingBox)

	nearestOsmObject := -1
	minDist := math.MaxFloat64
	for _, osmObject := range nearbyOsmObjects {
		distance := pointDistanceToOsmWay(osmObject.Leaf.BoundaryLatLons, lat, lon,
			osmObject.Leaf.Lat, osmObject.Leaf.Lon)
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

func pointDistanceToOsmWay(wayBoundary [][]float64, pointLat, pointLon float64,
	wayCenterLat, wayCenterLon float64) float64 {
	if len(wayBoundary) == 0 {
		dist := datastructure.HaversineDistance(pointLat, pointLon, wayCenterLat, wayCenterLon)
		return dist
	}
	minDist := math.MaxFloat64

	for i := 0; i < len(wayBoundary); i++ {
		j := (i + 1) % len(wayBoundary)
		projection := geo.ProjectPointToLineCoord(geo.NewCoordinate(wayBoundary[i][0], wayBoundary[i][1]),
			geo.NewCoordinate(wayBoundary[j][0], wayBoundary[j][1]), geo.NewCoordinate(pointLat, pointLon))
		distance := datastructure.HaversineDistance(pointLat, pointLon, projection.Lat, projection.Lon)
		if distance < minDist {
			minDist = distance
		}
	}
	return minDist
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
