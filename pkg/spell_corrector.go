package pkg

import (
	"bytes"
	"errors"
	"math"

	"github.com/blevesearch/vellum"
	"github.com/blevesearch/vellum/levenshtein"
)

const (
	COUNT_THRESOLD_NGRAM = 2
	EDIT_DISTANCE        = 2
)

type NgramLM interface {
	EstimateQueriesProbabilities(queries [][]int, n int, originalQueryTerms []int) []float64
	PreProcessData(tokenizedDocs [][]string, countThresold int) [][]int
	MakeCountMatrix(data [][]int)
	SaveNGramData() error
	LoadNGramData() error
}

type SpellCorrector struct {
	NGram          NgramLM
	CorpusTermsFST *vellum.FST
	Data           [][]int
	TermIDMap      IDMap
}

func NewSpellCorrector(ngram NgramLM) *SpellCorrector {
	return &SpellCorrector{
		NGram: ngram,
	}
}

// BuildFiniteStateTransducerSortedTerms. membuat finite state transducer dari sorted terms vocabulary. Di panggil saat server dijalankan.
func (sc *SpellCorrector) BuildFiniteStateTransducerSortedTerms(sortedTerms []string) error {

	var buf bytes.Buffer
	fstBuilder, err := vellum.New(&buf, nil)
	if err != nil {
		return err
	}

	for _, term := range sortedTerms {
		if err := fstBuilder.Insert([]byte(term), 0); err != nil {
			return err
		}
	}

	if err := fstBuilder.Close(); err != nil {
		return err
	}

	fst, err := vellum.Load(buf.Bytes())
	if err != nil {
		return err
	}
	sc.CorpusTermsFST = fst

	return nil
}

func (sc *SpellCorrector) InitializeSpellCorrector(sortedTerms []string, termIDMap IDMap) error {
	sc.TermIDMap = termIDMap
	sc.BuildFiniteStateTransducerSortedTerms(sortedTerms)
	err := sc.NGram.LoadNGramData()

	return err
}

// Preprocessdata. memproses data tokenized docs untuk membuat countmatrix n-gram.  Di panggil saat indexing dijalankan.
func (sc *SpellCorrector) Preprocessdata(tokenizedDocs [][]string) {
	sc.Data = sc.NGram.PreProcessData(tokenizedDocs, COUNT_THRESOLD_NGRAM)
	sc.NGram.MakeCountMatrix(sc.Data)

	sc.NGram.SaveNGramData()

}

func (sc SpellCorrector) GetWordCandidates(mispelledWord string, editDistance int) ([]int, error) {
	lv, err := levenshtein.NewLevenshteinAutomatonBuilder(uint8(editDistance), false) // harus false
	if err != nil {
		return []int{}, err
	}
	dfa, err := lv.BuildDfa(mispelledWord, uint8(editDistance))
	if err != nil {
		return []int{}, err
	}

	fstIt, err := sc.CorpusTermsFST.Search(dfa, nil, nil)

	correctWordCandidates := []int{}
	for err == nil {
		if err != nil {
			if errors.Is(err, vellum.ErrIteratorDone) {
				break
			}
			return []int{}, err
		}
		key, _ := fstIt.Current()
		correctWordCandidates = append(correctWordCandidates, sc.TermIDMap.GetID(string(key)))

		err = fstIt.Next()
	}
	return correctWordCandidates, nil
}

func (sc SpellCorrector) GetCorrectQueryCandidates(allPossibleQueryTerms [][]int) [][]int {
	temp := [][]int{{}}

	for i := 0; i < len(allPossibleQueryTerms); i++ {
		newTemp := [][]int{}
		for _, product := range temp {
			for _, term := range allPossibleQueryTerms[i] {
				tempCopy := product
				tempCopy = append(tempCopy, term)
				newTemp = append(newTemp, tempCopy)
			}
		}
		temp = newTemp
	}
	return temp
}

func (sc *SpellCorrector) GetCorrectSpellingSuggestion(allCorrectQueryCandidates [][]int, originalQueryTerms []int) ([]int, error) {

	correctQueriesProbabilities := sc.NGram.EstimateQueriesProbabilities(allCorrectQueryCandidates, 4, originalQueryTerms)

	maxProb := math.Inf(-1)
	var correctQuery = []int{}
	correctQueryIDX := -1

	for key, value := range correctQueriesProbabilities {
		if value > maxProb {
			maxProb = value
			correctQueryIDX = key
		}
	}
	correctQuery = append(correctQuery, allCorrectQueryCandidates[correctQueryIDX]...)
	return correctQuery, nil
}
