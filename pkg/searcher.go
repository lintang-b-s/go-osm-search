package pkg

import (
	"container/heap"
	"math"
	"sort"

	"github.com/RadhiFadlillah/go-sastrawi"
)

type DynamicIndexer interface {
	GetOutputDir() string
	GetDocWordCount() map[int]int
	GetDocsCount() int
	GetTermIDMap() IDMap
	BuildVocabulary()
}

type SearcherDocStore interface {
	GetDoc(int) (Node, error)
}

type Searcher struct {
	Idx DynamicIndexer
	// KV             SearcherKVDB
	MainIndex      *InvertedIndex
	SpellCorrector SpellCorrectorI
	TermIDMap      IDMap
	DocStore       SearcherDocStore
}

func NewSearcher(idx DynamicIndexer, docStore SearcherDocStore, spell SpellCorrectorI) *Searcher {

	return &Searcher{Idx: idx, DocStore: docStore, SpellCorrector: spell}
}

func (se *Searcher) LoadMainIndex() error {
	mainIndex := NewInvertedIndex("merged_index", se.Idx.GetOutputDir())
	err := mainIndex.OpenReader()
	if err != nil {
		return err
	}
	se.MainIndex = mainIndex
	se.Idx.BuildVocabulary()
	se.TermIDMap = se.Idx.GetTermIDMap()
	return nil
}

func (se *Searcher) Close() {
	se.MainIndex.Close()
}

type PostingsResult struct {
	Postings []int
	Err      error
	TermID   int
}

func NewPostingsResult(postings []int, err error, termID int) PostingsResult {
	return PostingsResult{
		Postings: postings,
		Err:      err,
		TermID:   termID,
	}
}

func (p *PostingsResult) GetError() error {
	return p.Err
}

func (p *PostingsResult) GetTermID() int {
	return p.TermID
}

func (p *PostingsResult) GetPostings() []int {
	return p.Postings
}

func (se *Searcher) GetPostingListCon(termID int) PostingsResult {
	postings, err := se.MainIndex.GetPostingList(termID)
	if err != nil {
		return NewPostingsResult([]int{}, err, termID)
	}
	return NewPostingsResult(postings, nil, termID)
}

type DocWithScore struct {
	DocID int
	Score float64
}

// https://nlp.stanford.edu/IR-book/pdf/06vect.pdf (figure 6.14 bagian function COSINESCORE(q))
func (se *Searcher) FreeFormQuery(query string, k int) ([]Node, error) {
	if k == 0 {
		k = 10
	}
	documentScore := make(map[int]float64) // menyimpan skor cosine tf-idf docs \dot tf-idf query

	queryTerms := sastrawi.Tokenize(query)

	queryWordCount := make(map[int]int, len(queryTerms))

	queryTermsID := make([]int, 0, len(queryTerms))

	allPostings := make(map[int][]int, len(queryTerms))

	// {{term1,term1OneEdit}, {term2, term2Edit}, ...}
	allPossibleQueryTerms := make([][]int, len(queryTerms))
	originalQueryTerms := make([]int, 0, len(queryTerms))

	for i, term := range queryTerms {
		tokenizedTerm := stemmer.Stem(term)
		isInVocab := se.TermIDMap.IsInVocabulary(tokenizedTerm)

		originalQueryTerms = append(originalQueryTerms, se.TermIDMap.GetID(tokenizedTerm))

		if !isInVocab {

			correctionOne, err := se.SpellCorrector.GetWordCandidates(tokenizedTerm, 1)
			if err != nil {
				return []Node{}, err
			}
			correctionTwo, err := se.SpellCorrector.GetWordCandidates(tokenizedTerm, 2)
			if err != nil {
				return []Node{}, err
			}
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], correctionOne...)
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], correctionTwo...)
		} else {
			termID := se.TermIDMap.GetID(tokenizedTerm)
			allPossibleQueryTerms[i] = []int{termID}
		}
	}

	allCorrectQueryCandidates := se.SpellCorrector.GetCorrectQueryCandidates(allPossibleQueryTerms)
	correctQuery, err := se.SpellCorrector.GetCorrectSpellingSuggestion(allCorrectQueryCandidates, originalQueryTerms)

	if err != nil {
		return []Node{}, err
	}

	queryTermsID = append(queryTermsID, correctQuery...)

	for _, termID := range queryTermsID {
		postings, err := se.MainIndex.GetPostingList(termID)
		if err != nil {
			return []Node{}, err
		}
		allPostings[termID] = postings
		queryWordCount[termID] += 1
	}

	docWordCount := se.Idx.GetDocWordCount() // docs ada 200k
	docsCount := float64(se.Idx.GetDocsCount())
	docNorm := make(map[int]float64)
	queryNorm := 0.0
	for qTermID, postings := range allPostings {
		// iterate semua term di query, hitung tf-idf query dan tf-idf document, accumulate skor cosine di docScore
		tfTermQuery := float64(queryWordCount[qTermID]) / float64(len(queryWordCount))
		idfTermQuery := math.Log10(docsCount) - math.Log10(float64(len(postings)))
		tfIDFTermQuery := tfTermQuery * idfTermQuery
		for _, docID := range postings {
			// compute tf-idf query dan document & compute cosine nya

			tf := 1.0 / float64(docWordCount[docID])

			tfIDFTermDoc := tf * idfTermQuery

			documentScore[docID] += tfIDFTermDoc * tfIDFTermQuery

			docNorm[docID] += tfIDFTermDoc * tfIDFTermDoc
		}
		queryNorm += tfIDFTermQuery * tfIDFTermQuery
	}

	queryNorm = math.Sqrt(queryNorm)
	for docID, norm := range docNorm {
		docNorm[docID] = math.Sqrt(norm)
	}

	docWithScores := make([]DocWithScore, 0, len(documentScore))
	// normalize dengan cara dibagi dengan norm vector query & document
	for docID, score := range documentScore {
		documentScore[docID] = score / (queryNorm * docNorm[docID])
		docWithScores = append(docWithScores, DocWithScore{DocID: docID, Score: documentScore[docID]})
	}

	sort.Slice(docWithScores, func(i, j int) bool {
		return docWithScores[i].Score > docWithScores[j].Score
	})

	relevantDocs := make([]Node, 0, k)
	for i := 0; i < k; i++ {
		if i >= len(docWithScores) {
			break
		}

		doc, err := se.DocStore.GetDoc(docWithScores[i].DocID)
		if err != nil {
			return []Node{}, err
		}
		relevantDocs = append(relevantDocs, doc)
	}

	return relevantDocs, nil
}

// pakai faninfanout malah tambah lemot

func (se *Searcher) FreeFormQueryWithoutDocs(query string, k int) ([]int, error) {
	if k == 0 {
		k = 10
	}
	documentScore := make(map[int]float64) // menyimpan skor cosine tf-idf docs \dot tf-idf query

	docsPQ := NewMaxPriorityQueue[int, float64]()
	heap.Init(docsPQ)

	queryTerms := sastrawi.Tokenize(query)

	queryWordCount := make(map[int]int, len(queryTerms))

	queryTermsID := make([]int, 0, len(queryTerms))

	allPostings := make(map[int][]int, len(queryTerms))

	// {{term1,term1OneEdit}, {term2, term2Edit}, ...}
	allPossibleQueryTerms := make([][]int, len(queryTerms))
	originalQueryTerms := make([]int, 0, len(queryTerms))

	for i, term := range queryTerms {
		tokenizedTerm := stemmer.Stem(term)
		isInVocab := se.TermIDMap.IsInVocabulary(tokenizedTerm)

		originalQueryTerms = append(originalQueryTerms, se.TermIDMap.GetID(tokenizedTerm))

		if !isInVocab {

			correctionOne, err := se.SpellCorrector.GetWordCandidates(tokenizedTerm, 1)
			if err != nil {
				return []int{}, err
			}
			correctionTwo, err := se.SpellCorrector.GetWordCandidates(tokenizedTerm, 2)
			if err != nil {
				return []int{}, err
			}
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], correctionOne...)
			allPossibleQueryTerms[i] = append(allPossibleQueryTerms[i], correctionTwo...)
		} else {
			termID := se.TermIDMap.GetID(tokenizedTerm)
			allPossibleQueryTerms[i] = []int{termID}
		}
	}

	allCorrectQueryCandidates := se.SpellCorrector.GetCorrectQueryCandidates(allPossibleQueryTerms)
	correctQuery, err := se.SpellCorrector.GetCorrectSpellingSuggestion(allCorrectQueryCandidates, originalQueryTerms)

	if err != nil {
		return []int{}, err
	}

	queryTermsID = append(queryTermsID, correctQuery...)

	for _, termID := range queryTermsID {
		postings, err := se.MainIndex.GetPostingList(termID)
		if err != nil {
			return []int{}, err
		}
		allPostings[termID] = postings
		queryWordCount[termID] += 1
	}

	docWordCount := se.Idx.GetDocWordCount()
	docsCount := float64(se.Idx.GetDocsCount())
	docNorm := make(map[int]float64)
	queryNorm := 0.0
	for qTermID, postings := range allPostings {
		// iterate semua term di query, hitung tf-idf query dan tf-idf document, accumulate skor cosine di docScore
		tfTermQuery := float64(queryWordCount[qTermID]) / float64(len(queryWordCount))
		termOccurences := len(postings)
		idfTermQuery := math.Log10(docsCount) - math.Log10(float64(termOccurences))
		tfIDFTermQuery := tfTermQuery * idfTermQuery
		for _, docID := range postings {
			// compute tf-idf query dan document & compute cosine nya

			tf := 1.0 / float64(docWordCount[docID])

			idf := math.Log10(docsCount) - math.Log10(float64(termOccurences))
			tfIDFTermDoc := tf * idf

			documentScore[docID] += tfIDFTermDoc * tfIDFTermQuery

			docNorm[docID] += tfIDFTermDoc * tfIDFTermDoc
		}
		queryNorm += tfIDFTermQuery * tfIDFTermQuery
	}

	queryNorm = math.Sqrt(queryNorm)
	for docID, norm := range docNorm {
		docNorm[docID] = math.Sqrt(norm)
	}

	docWithScores := make([]DocWithScore, 0, len(documentScore))
	// normalize dengan cara dibagi dengan norm vector query & document
	for docID, score := range documentScore {
		documentScore[docID] = score / (queryNorm * docNorm[docID])
		docWithScores = append(docWithScores, DocWithScore{DocID: docID, Score: documentScore[docID]})
	}

	sort.Slice(docWithScores, func(i, j int) bool {
		return docWithScores[i].Score > docWithScores[j].Score
	})

	relevantDocs := make([]int, 0, k)
	for i := 0; i < k; i++ {
		if i >= len(docWithScores) {
			break
		}

		currRelDocID := docWithScores[i].DocID

		relevantDocs = append(relevantDocs, currRelDocID)
	}

	return relevantDocs, nil
}

// pake pq sendiri tambah lemot
