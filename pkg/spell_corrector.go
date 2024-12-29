package pkg

import (
	"bytes"
	"errors"

	"github.com/blevesearch/vellum"
	"github.com/blevesearch/vellum/levenshtein"
)

const (
	COUNT_THRESOLD_NGRAM = 2
	EDIT_DISTANCE        = 2
)

type NgramLM interface {
	EstimateWordCandidatesProbabilities(nextWordCandidates []int, prevNgrams []int, n int) map[int]float64
	EstimateWordCandidatesProbabilitiesWithStupidBackoff(nextWordCandidates []int, prevNgrams []int, n int) map[int]float64
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

func (sc *SpellCorrector) GetCorrectSpellingSuggestion(mispelledWord string, prevWords []string) (string, error) {
	lv, err := levenshtein.NewLevenshteinAutomatonBuilder(EDIT_DISTANCE, false) // harus false
	if err != nil {
		return "", err
	}
	dfa, err := lv.BuildDfa(mispelledWord, EDIT_DISTANCE)
	if err != nil {
		return "", err
	}

	fstIt, err := sc.CorpusTermsFST.Search(dfa, nil, nil)

	correctWordCandidates := []int{}
	for err == nil {
		if err != nil {
			if errors.Is(err, vellum.ErrIteratorDone) {
				break
			}
			return "", err
		}
		key, _ := fstIt.Current()
		correctWordCandidates = append(correctWordCandidates, sc.TermIDMap.GetID(string(key)))

		err = fstIt.Next()
	}

	prevTokens := []int{}
	for _, prevWord := range prevWords {
		prevTokens = append(prevTokens, sc.TermIDMap.GetID(prevWord))
	}

	n := 4
	if len(prevTokens) < 3 {
		// trigram = [prev-1,prev] [current]
		n = len(prevTokens) + 1
	}

	// correctWordProbabilities := sc.NGram.EstimateWordCandidatesProbabilities(correctWordCandidates, prevTokens, n)
	correctWordProbabilities := sc.NGram.EstimateWordCandidatesProbabilitiesWithStupidBackoff(correctWordCandidates, prevTokens, n)

	maxProb := -9999.0
	var correctWord string

	for key, value := range correctWordProbabilities {
		if value > maxProb {
			maxProb = value
			correctWord = sc.TermIDMap.GetStr(key)
		}
	}
	return correctWord, nil
}
