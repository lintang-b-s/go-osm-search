package pkg

import (
	"bytes"

	"github.com/blevesearch/vellum"
	"github.com/blevesearch/vellum/levenshtein"
)

const (
	COUNT_THRESOLD_NGRAM = 1
	EDIT_DISTANCE        = 2
)

type NgramLM interface {
	EstimateWordCandidatesProbabilities(nextWordCandidates []int, prevNgrams []int, n int) map[int]float64
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
// BuildFiniteStateTransducerSortedTerms. membuat finite state transducer dari sorted terms. Di panggil saat server dijalankan.
func (sc *SpellCorrector) BuildFiniteStateTransducerSortedTerms(sortedTerms []string) error {

	var buf bytes.Buffer
	fstBuilder, err := vellum.New(&buf, nil)
	if err != nil {
		return err
	}

	for _, term := range sortedTerms {
		fstBuilder.Insert([]byte(term), 0)
	}

	fst, err := vellum.Load(buf.Bytes())
	if err != nil {
		return err
	}
	sc.CorpusTermsFST = fst

	return nil
}

func (sc *SpellCorrector) InitializeSpellCorrector(sortedTerms []string) error {
	sc.BuildFiniteStateTransducerSortedTerms(sortedTerms)
	err := sc.NGram.LoadNGramData()
	return err 
}

// Preprocessdata. memproses data tokenized docs untuk membuat countmatrix n-gram.  Di panggil saat indexing dijalankan.
func (sc *SpellCorrector) Preprocessdata(tokenizedDocs [][]string)  {
	sc.Data = sc.NGram.PreProcessData(tokenizedDocs, COUNT_THRESOLD_NGRAM)
	sc.NGram.MakeCountMatrix(sc.Data)
	// sortedTerms := sc.TermIDMap.GetSortedTerms()
	// err := sc.BuildFiniteStateTransducerSortedTerms(sortedTerms)
	sc.NGram.SaveNGramData()
	return
}

func (sc *SpellCorrector) GetCorrectSpellingSuggestion(mispelledWord string, prevWords []string) (string, error) {
	lv, err := levenshtein.NewLevenshteinAutomatonBuilder(EDIT_DISTANCE, true)
	if err != nil {
		return "", err
	}
	dfa, err := lv.BuildDfa(mispelledWord, 2)
	if err != nil {
		return "", err
	}

	fstIt, err := sc.CorpusTermsFST.Search(dfa, []byte{}, []byte{})

	correctWordCandidates := []int{}
	for err != nil {
		key, _ := fstIt.Current()
		err = fstIt.Next()
		correctWordCandidates = append(correctWordCandidates, sc.TermIDMap.GetID(string(key)))
	}
	if err != nil {
		return "", err
	}

	prevTokens := []int{}
	for _, prevWord := range prevWords {
		prevTokens = append(prevTokens, sc.TermIDMap.GetID(prevWord))
	}

	correctWordProbabilities := sc.NGram.EstimateWordCandidatesProbabilities(correctWordCandidates, prevTokens, 1)

	maxProb := -1.0
	var correctWord string

	for key, value := range correctWordProbabilities {
		if value > maxProb {
			maxProb = value
			correctWord = sc.TermIDMap.GetStr(key)
		}
	}
	return correctWord, nil
}
