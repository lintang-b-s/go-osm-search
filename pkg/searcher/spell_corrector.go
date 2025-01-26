package searcher

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"osm-search/pkg"
	"sort"

	rege "regexp"

	"github.com/blevesearch/vellum"
	"github.com/blevesearch/vellum/levenshtein"
	"github.com/blevesearch/vellum/regexp"
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
	TermIDMap      pkg.IDMap
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

func (sc *SpellCorrector) InitializeSpellCorrector(sortedTerms []string, termIDMap pkg.IDMap) error {
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

// https://docs.google.com/presentation/d/1Z7OYvKc5dHAXiVdMpk69uulpIT6A7FGfohjHx8fmHBU/edit#slide=id.p
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

		key, _ := fstIt.Current()
		correctWordCandidates = append(correctWordCandidates, sc.TermIDMap.GetID(string(key)))

		err = fstIt.Next()
		if err != nil {
			if errors.Is(err, vellum.ErrIteratorDone) {
				break
			}
			return []int{}, err
		}
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

// section autocomplete
// https://www.elastic.co/blog/you-complete-me
// GetMatchedWordBasedOnPrefix. return all matched word di term dictionary yang match dengan prefixWord.
func (sc SpellCorrector) GetMatchedWordBasedOnPrefix(prefixWord string) ([]int, error) {

	prefixReg := fmt.Sprintf(`%s.*`, rege.QuoteMeta(prefixWord))
	regAutomaton, err := regexp.New(prefixReg)
	if err != nil {
		return []int{}, fmt.Errorf("error when initializing regex automaton: %w", err)
	}
	fstIt, err := sc.CorpusTermsFST.Search(regAutomaton, nil, nil)
	if err != nil {
		return []int{}, fmt.Errorf("error when executing regex automaton: %w", err)
	}
	// searcher, err :=sc.CorpusTermsFST.Iterator()
	matchedWordCandidates := []int{}
	for err == nil {

		key, _ := fstIt.Current()
		matchedWordCandidates = append(matchedWordCandidates, sc.TermIDMap.GetID(string(key)))

		err = fstIt.Next()
		if err != nil {
			if errors.Is(err, vellum.ErrIteratorDone) {
				break
			}
			return []int{}, err
		}
	}
	return matchedWordCandidates, nil
}

type QueryCandidatesWithProb struct {
	IDx  int
	Prob float64
}

func NewQueryCandidatesWithProb(idx int, prob float64) QueryCandidatesWithProb {
	return QueryCandidatesWithProb{
		IDx:  idx,
		Prob: prob,
	}
}

func (sc *SpellCorrector) GetMatchedWordsAutocomplete(allQueryCandidates [][]int, originalQueryTerms []int) ([][]int, error) {

	queryCandidatesProbabilities := sc.NGram.EstimateQueriesProbabilities(allQueryCandidates, 4, originalQueryTerms)

	queryCandidates := make([]QueryCandidatesWithProb, 0, len(queryCandidatesProbabilities))

	for idx, prob := range queryCandidatesProbabilities {
		queryCandidates = append(queryCandidates, NewQueryCandidatesWithProb(idx, prob))
	}

	sort.Slice(queryCandidates, func(i, j int) bool {
		return queryCandidates[i].Prob > queryCandidates[j].Prob
	})

	matchedQuery := [][]int{}

	for _, qcan := range queryCandidates {
		matchedQuery = append(matchedQuery, allQueryCandidates[qcan.IDx])
	}

	if len(matchedQuery) >= 5 {
		return matchedQuery[:5], nil
	}

	return matchedQuery, nil
}
