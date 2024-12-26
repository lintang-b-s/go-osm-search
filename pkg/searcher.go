package pkg

import (
	"math"

	"github.com/RadhiFadlillah/go-sastrawi"
)

type DynamicIndexer interface {
	GetOutputDir() string
	GetDocWordCount() map[int]int
	GetDocsCount() int
	GetTermIDMap() IDMap
}

type SearcherKVDB interface {
	GetNode(id int) (Node, error)
}

type Searcher struct {
	Idx DynamicIndexer
	KV  SearcherKVDB
}

func NewSearcher(idx DynamicIndexer, kv SearcherKVDB) *Searcher {
	return &Searcher{Idx: idx, KV: kv}
}

// https://nlp.stanford.edu/IR-book/pdf/06vect.pdf (figure 6.14 bagian function COSINESCORE(q))
func (se *Searcher) FreeFormQuery(query string, k int) ([]Node, error) {
	if k == 0 {
		k = 10
	}
	documentScore := make(map[int]float64) // menyimpan skor cosine tf-idf docs \dot tf-idf query
	allPostings := make(map[int][]int)
	docsPQ := NewPriorityQueue[int, float64]()
	termMapper := se.Idx.GetTermIDMap()

	mainIndex := NewInvertedIndex("index_0", se.Idx.GetOutputDir())
	err := mainIndex.OpenReader()
	if err != nil {
		return []Node{}, err
	}
	queryWordCount := make(map[int]int)
	for _, term := range sastrawi.Tokenize(query) {
		tokenizedTerm := stemmer.Stem(term)
		termID := termMapper.GetID(tokenizedTerm)
		postings, err := mainIndex.GetPostingList(termID) // sorted postings untuk termID
		if err != nil {
			return []Node{}, err
		}
		allPostings[termID] = postings
		queryWordCount[termID] += 1
	}

	docNorm := make(map[int]float64)
	queryNorm := 0.0
	for qTermID, postings := range allPostings {
		// iterate semua term di query, hitung tf-idf query dan tf-idf document, accumulate skor cosine di docScore
		//
		tfTermQuery := float64(queryWordCount[qTermID]) / float64(len(queryWordCount))
		termOccurences := len(postings)
		idfTermQuery := math.Log10(float64(se.Idx.GetDocsCount())) - math.Log10(float64(termOccurences))
		tfIDFTermQuery := tfTermQuery * idfTermQuery
		for _, docID := range postings {
			// compute tf-idf query dan document & compute cosine nya
			tfIDFTermDoc := se.computeDocTFIDFPerTerm(docID, qTermID, postings)

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
		pqItem := NewPriorityQueueNode[int](documentScore[docID], docID)
		docsPQ.Push(pqItem)
	}

	relevantDocs := []Node{}
	for i := 0; i < k; i++ {
		if docsPQ.Len() == 0 {
			break
		}
		currRelDocID := docsPQ.Pop().(*priorityQueueNode[int, float64]).item
		doc, err := se.KV.GetNode(currRelDocID)
		if err != nil {
			return []Node{}, err
		}
		relevantDocs = append(relevantDocs, doc)
	}
	return relevantDocs, nil
}

func (se *Searcher) computeDocTFIDFPerTerm(docID int, termID int, postingList []int) float64 {
	tf := 0.0
	docWordCount := se.Idx.GetDocWordCount()
	for _, docIDPosting := range postingList {
		// kalo postingListnya pake skip list lebih cepet
		if docIDPosting == docID {
			tf += 1.0 / float64(docWordCount[docID])
		}
	}
	termOccurences := len(postingList)
	idf := math.Log10(float64(se.Idx.GetDocsCount())) - math.Log10(float64(termOccurences))
	return tf * idf
}
