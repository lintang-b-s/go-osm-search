package pkg

import (
	"container/heap"
	"math"

	"github.com/RadhiFadlillah/go-sastrawi"
)

const (
	NUM_WORKER_FANINFANOUT = 3
)

type DynamicIndexer interface {
	GetOutputDir() string
	GetDocWordCount() map[int]int
	GetDocsCount() int
	GetTermIDMap() IDMap
	BuildVocabulary()
}

type SearcherKVDB interface {
	GetNode(id int) (Node, error)
}

type Searcher struct {
	Idx            DynamicIndexer
	KV             SearcherKVDB
	MainIndex      *InvertedIndex
	SpellCorrector SpellCorrectorI
	TermIDMap      IDMap
}

func NewSearcher(idx DynamicIndexer, kv SearcherKVDB, spell SpellCorrectorI) *Searcher {

	return &Searcher{Idx: idx, KV: kv, SpellCorrector: spell}
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

// https://nlp.stanford.edu/IR-book/pdf/06vect.pdf (figure 6.14 bagian function COSINESCORE(q))
func (se *Searcher) FreeFormQuery(query string, k int) ([]Node, error) {
	if k == 0 {
		k = 10
	}
	documentScore := make(map[int]float64) // menyimpan skor cosine tf-idf docs \dot tf-idf query
	allPostings := make(map[int][]int)
	docsPQ := NewMaxPriorityQueue[int, float64]()
	heap.Init(docsPQ)

	queryWordCount := make(map[int]int)

	queryTermsID := []int{}

	queryTerms := sastrawi.Tokenize(query)

	// {{term1,term1OneEdit}, {term2, term2Edit}, ...}
	allPossibleQueryTerms := make([][]int, len(queryTerms))
	originalQueryTerms := make([]int, len(queryTerms))
	

	for i, term := range queryTerms {
		tokenizedTerm := stemmer.Stem(term)
		isInVocab := se.TermIDMap.IsInVocabulary(tokenizedTerm)

		originalQueryTerms[i] = se.TermIDMap.GetID(tokenizedTerm)

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
	

	fanInFanOut := NewFanInFanOut[int, PostingsResult](len(queryTermsID))
	fanInFanOut.GeneratePipeline(queryTermsID)

	outs := []<-chan PostingsResult{}
	for i := 0; i < NUM_WORKER_FANINFANOUT; i++ {
		outs1 := fanInFanOut.FanOut(se.GetPostingListCon)
		outs = append(outs, outs1)
	}

	results := fanInFanOut.FanIn(outs...)
	for postingsRes := range results {
		err := postingsRes.GetError()
		if err != nil {
			return []Node{}, err
		}
		allPostings[postingsRes.GetTermID()] = postingsRes.GetPostings()
		queryWordCount[postingsRes.GetTermID()] += 1
	}

	docWordCount := se.Idx.GetDocWordCount()

	docNorm := make(map[int]float64)
	queryNorm := 0.0
	for qTermID, postings := range allPostings {
		// iterate semua term di query, hitung tf-idf query dan tf-idf document, accumulate skor cosine di docScore
		tfTermQuery := float64(queryWordCount[qTermID]) / float64(len(queryWordCount))
		termOccurences := len(postings)
		idfTermQuery := math.Log10(float64(se.Idx.GetDocsCount())) - math.Log10(float64(termOccurences))
		tfIDFTermQuery := tfTermQuery * idfTermQuery
		for _, docID := range postings {
			// compute tf-idf query dan document & compute cosine nya

			tf := 1.0 / float64(docWordCount[docID])
			termOccurences := len(postings)
			idf := math.Log10(float64(se.Idx.GetDocsCount())) - math.Log10(float64(termOccurences))
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

	// normalize dengan cara dibagi dengan norm vector query & document
	for docID, score := range documentScore {
		documentScore[docID] = score / (queryNorm * docNorm[docID])
		pqItem := NewPriorityQueueNode[int, float64](documentScore[docID], docID)
		heap.Push(docsPQ, pqItem)

	}

	relevantDocs := []Node{}
	for i := 0; i < k; i++ {
		if docsPQ.Len() == 0 {
			break
		}

		heapItem := heap.Pop(docsPQ).(*priorityQueueNode[int, float64])
		currRelDocID := heapItem.item
		doc, err := se.KV.GetNode(currRelDocID)
		if err != nil {
			return []Node{}, err
		}
		relevantDocs = append(relevantDocs, doc)
	}

	return relevantDocs, nil
}
