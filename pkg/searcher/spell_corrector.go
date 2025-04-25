package searcher

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"

	rege "regexp"

	"github.com/blevesearch/vellum"
	"github.com/blevesearch/vellum/levenshtein"
	"github.com/blevesearch/vellum/regexp"
)

type SpellCorrector struct {
	NGram             NgramLM
	CorpusTermsFST    *vellum.FST
	Data              [][]int
	TermIDMap         *pkg.IDMap
	NoisyChannelModel *NoisyChannelModel
	outputDir         string
}

type NoisyChannelModel struct {
	UnigramCount map[rune]int
	BigramCount  map[[2]rune]int
	EditCount    map[EditConst]map[[2]int]int // map for (editType, confusion matrix). confustion matrix: del[x_{i-1}, w_i], ins[x_{i-1}, w_i], sub[x_i,w_i], trans[w_i, w_{i+1}]
	AlphabetSize int
}

func NewSpellCorrector(ngram NgramLM, outputDir string) *SpellCorrector {
	return &SpellCorrector{
		NGram:             ngram,
		NoisyChannelModel: NewNoisyChannelModel(),
		outputDir:         outputDir,
	}
}

func NewNoisyChannelModel() *NoisyChannelModel {
	return &NoisyChannelModel{
		UnigramCount: make(map[rune]int),
		BigramCount:  make(map[[2]rune]int),
		EditCount:    make(map[EditConst]map[[2]int]int),
	}
}

// https://web.stanford.edu/~jurafsky/slp3/B.pdf (noisy channel model)
func (sc SpellCorrector) BuildEditProb(filename string) error {
	/// read from spell-errors.txt

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error when opening file %s: %w", filename, err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		correctWord := strings.TrimSpace(parts[0])
		spellErrors := strings.TrimSpace(parts[1])

		// update unigram count
		for c := range correctWord {
			sc.NoisyChannelModel.UnigramCount[charToRune(correctWord[c])]++
		}
		sc.NoisyChannelModel.UnigramCount[charToRune(START_CHAR)]++

		// update bigram count
		for i := 0; i < len(correctWord)-1; i++ {
			c1 := charToRune(correctWord[i])
			c2 := charToRune(correctWord[i+1])
			sc.NoisyChannelModel.BigramCount[[2]rune{c1, c2}]++
		}

		for _, spellError := range strings.Split(spellErrors, ",") {
			edit, c1, c2 := getEdit(spellError, correctWord)
			if _, ok := sc.NoisyChannelModel.EditCount[edit]; !ok {
				sc.NoisyChannelModel.EditCount[edit] = make(map[[2]int]int)
			}
			sc.NoisyChannelModel.EditCount[edit][[2]int{int(c1), int(c2)}]++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error when reading file %s: %w", filename, err)
	}
	sc.NoisyChannelModel.AlphabetSize = len(sc.NoisyChannelModel.UnigramCount)

	return nil
}

func getEdit(edited, original string) (EditConst, rune, rune) {
	if edited == original {
		return -1, charToRune(edited[0]), charToRune(original[0])
	}
	if len(edited) == len(original) {
		// substitution or transposition
		eCounter := make(map[rune]int)
		for i := 0; i < len(edited); i++ {
			eCounter[rune(edited[i])]++
		}
		oCounter := make(map[rune]int)
		for i := 0; i < len(original); i++ {
			oCounter[rune(original[i])]++
		}

		isCounterSame := true
		for k, v := range eCounter {
			if oCounter[k] != v {
				isCounterSame = false
				break
			}
		}

		for i := 0; i < len(edited); i++ {
			c1 := edited[i]
			c2 := original[i]
			if c1 != c2 {
				if isCounterSame {

					// transposition
					// example: "abcd" -> "abdc"
					return Transposition, charToRune(c1), charToRune(c2)
				} else {
					// substitution
					// example: "abcd" -> "abcf"
					return Substitution, charToRune(c1), charToRune(c2)
				}
			}
		}
	}

	// insertion or deletion
	for i := range minInt(len(edited), len(original)) {
		e, o := rune(edited[i]), rune(original[i])
		if e != o {
			// Insertion

			if len(edited) > len(original) {
				if i > 0 {
					// example: "abcd" -> "abfcd"
					return Insertion, e, rune(original[i-1])
				} else {
					// example: "abc" -> "fabc"
					return Insertion, e, START_CHAR
				}
			} else {
				// deletion
				if i > 0 {
					// 	example: "abcde" -> "abce"
					return Deletion, e, rune(original[i-1])
				} else {
					// example: "abc" -> "bc"
					return Deletion, e, START_CHAR
				}
			}
		}
	}
	if len(edited) > len(original) {
		// insertion
		// example: "stanford" -> "stanfords"
		return Insertion, rune(edited[len(edited)-1]), rune(original[len(original)-1])
	} else {
		// deletion
		if len(original) > 1 {
			// deletion
			// example: "stanford" -> "stanfor"
			return Deletion, rune(edited[len(edited)-1]), rune(original[len(original)-2])
		} else {
			// deletion
			// example: "stanford" -> "tanford"
			return Deletion, rune(edited[len(edited)-1]), START_CHAR
		}
	}

}

func (sc SpellCorrector) getEditLogProb(edited, original string) float64 {
	var (
		denumerator float64
	)
	edit, c1, c2 := getEdit(edited, original)
	if edit == -1 {
		return math.Log(ALPHA_NO_EDIT_PROB)
	}

	numerator := float64(sc.NoisyChannelModel.EditCount[edit][[2]int{int(c1), int(c2)}])

	if edit == Insertion || edit == Substitution {
		denumerator = float64(sc.NoisyChannelModel.UnigramCount[c1]) + float64(sc.NoisyChannelModel.AlphabetSize)
	} else {
		denumerator = float64(sc.NoisyChannelModel.BigramCount[[2]rune{c1, c2}]) + float64(sc.NoisyChannelModel.AlphabetSize*sc.NoisyChannelModel.AlphabetSize)
	}

	return math.Log(numerator+1) - math.Log(denumerator)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func charToRune(c byte) rune {
	return rune(c)
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

func (sc *SpellCorrector) InitializeSpellCorrector(sortedTerms []string, termIDMap *pkg.IDMap) error {
	sc.TermIDMap = termIDMap
	sc.BuildFiniteStateTransducerSortedTerms(sortedTerms)
	err := sc.NGram.LoadNGramData()
	sc.NGram.SetTermIDMap(termIDMap)

	return err
}

// Preprocessdata. memproses data tokenized docs untuk membuat countmatrix n-gram.  Di panggil saat indexing dijalankan.
func (sc *SpellCorrector) Preprocessdata(tokenizedDocs [][]string) {
	sc.Data = sc.NGram.PreProcessData(tokenizedDocs, COUNT_THRESOLD_NGRAM)
	sc.NGram.MakeCountMatrix(sc.Data)

	sc.NGram.SaveNGramData()

}

// https://docs.google.com/presentation/d/1Z7OYvKc5dHAXiVdMpk69uulpIT6A7FGfohjHx8fmHBU/edit#slide=id.p
func (sc SpellCorrector) GetWordCandidates(mispelledWord string, editDistance int) ([]int, []string, error) {
	lv, err := levenshtein.NewLevenshteinAutomatonBuilder(uint8(editDistance), false) // harus false
	if err != nil {
		return []int{}, []string{}, err
	}
	dfa, err := lv.BuildDfa(mispelledWord, uint8(editDistance))
	if err != nil {
		return []int{}, []string{}, err
	}

	fstIt, err := sc.CorpusTermsFST.Search(dfa, nil, nil)

	correctWordCandidates := []int{}
	correctWordCandidatesString := []string{}
	for err == nil {

		key, _ := fstIt.Current()
		correctWordCandidates = append(correctWordCandidates, sc.TermIDMap.GetID(string(key)))
		correctWordCandidatesString = append(correctWordCandidatesString, string(key))

		err = fstIt.Next()
		if err != nil {
			if errors.Is(err, vellum.ErrIteratorDone) {
				break
			}
			return []int{}, []string{}, err
		}
	}
	return correctWordCandidates, correctWordCandidatesString, nil
}

// GetCorrectQueryCandidates. all query candidates that generated by combination of candidate term 1,  candidate term 2, ....
func (sc SpellCorrector) GetCorrectQueryCandidates(allPossibleQueryTerms [][]datastructure.WordCandidate) [][]datastructure.WordCandidate {
	temp := [][]datastructure.WordCandidate{{}}

	for i := 0; i < len(allPossibleQueryTerms); i++ {
		newTemp := [][]datastructure.WordCandidate{}
		for _, product := range temp {
			for _, term := range allPossibleQueryTerms[i] {
				productCopy := make([]datastructure.WordCandidate, len(product))
				copy(productCopy, product)
				productCopy = append(productCopy, term)
				newTemp = append(newTemp, productCopy)
			}
		}
		temp = newTemp
	}
	return temp
}

func (sc *SpellCorrector) GetCorrectSpellingSuggestion(allCorrectQueryCandidates [][]datastructure.WordCandidate) ([]int, error) {
	allCorrectQueryCandidateIDs := make([][]int, 0, len(allCorrectQueryCandidates))
	for _, queryCandidateIDs := range allCorrectQueryCandidates {
		currQueryTermIDs := make([]int, 0, len(queryCandidateIDs))

		for _, term := range queryCandidateIDs {

			currQueryTermIDs = append(currQueryTermIDs, term.CandiateWordID)
		}
		allCorrectQueryCandidateIDs = append(allCorrectQueryCandidateIDs, currQueryTermIDs)
	}
	correctQueriesLMProbabilities := sc.NGram.GetQueryNgramProbability(allCorrectQueryCandidateIDs, 4)

	maxProb := math.Inf(-1)
	var correctQuery = []int{}
	correctQueryIDX := -1

	for key, curQueryProb := range correctQueriesLMProbabilities {
		currCandidateQuery := allCorrectQueryCandidates[key]
		editProb := 0.0
		for _, term := range currCandidateQuery {

			editProb += sc.getEditLogProb(term.TypoWord, term.CorrectedWord)
		}
		curQueryProb += editProb

		if curQueryProb > maxProb {
			maxProb = curQueryProb
			correctQueryIDX = key
		}
	}
	correctQuery = append(correctQuery, allCorrectQueryCandidateIDs[correctQueryIDX]...)
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
		if errors.Is(err, vellum.ErrIteratorDone) {
			return []int{}, nil
		}
		return []int{}, fmt.Errorf("error when executing regex automaton: %w", err)
	}

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

const (
	kAutoComplete = 3
)

func (sc *SpellCorrector) GetMatchedWordsAutocomplete(allQueryCandidates [][]datastructure.WordCandidate, originalQueryTerms []int) ([][]int, error) {
	allCorrectQueryCandidateIDs := make([][]int, 0, len(allQueryCandidates))
	for _, queryCandidateIDs := range allQueryCandidates {
		currQueryTermIDs := make([]int, 0, len(queryCandidateIDs))

		for _, term := range queryCandidateIDs {
			currQueryTermIDs = append(currQueryTermIDs, term.CandiateWordID)
		}
		allCorrectQueryCandidateIDs = append(allCorrectQueryCandidateIDs, currQueryTermIDs)
	}

	queryCandidatesLMProbabilities := sc.NGram.GetQueryNgramProbability(allCorrectQueryCandidateIDs, 4)

	queryCandidates := make([]QueryCandidatesWithProb, 0, len(queryCandidatesLMProbabilities))

	for idx, prob := range queryCandidatesLMProbabilities {
		currCandidateQuery := allQueryCandidates[idx]
		editProb := 0.0
		for _, term := range currCandidateQuery {
			editProb += sc.getEditLogProb(term.TypoWord, term.CorrectedWord)
		}
		prob += editProb

		queryCandidates = append(queryCandidates, NewQueryCandidatesWithProb(idx, prob))
	}

	sort.Slice(queryCandidates, func(i, j int) bool {
		return queryCandidates[i].Prob > queryCandidates[j].Prob
	})

	matchedQuery := [][]int{}

	for _, qcan := range queryCandidates {
		matchedQuery = append(matchedQuery, allCorrectQueryCandidateIDs[qcan.IDx])
	}

	if len(matchedQuery) >= kAutoComplete {
		return matchedQuery[:kAutoComplete], nil
	}

	return matchedQuery, nil
}

func (sc *SpellCorrector) SaveNoisyChannelModelData() error {

	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(sc.NoisyChannelModel)
	if err != nil {
		return fmt.Errorf("error when marshalling metadata: %w", err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var noisyChannelFile *os.File

	if pwd != "/" {
		noisyChannelFile, err = os.OpenFile(pwd+"/"+sc.outputDir+"/"+"noisy_model.index", os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	} else {
		noisyChannelFile, err = os.OpenFile(sc.outputDir+"/"+"noisy_model.index", os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	}
	defer noisyChannelFile.Close()

	err = noisyChannelFile.Truncate(0)
	if err != nil {
		return err
	}

	_, err = noisyChannelFile.Write(buf.Bytes())

	return err
}

func (sc *SpellCorrector) LoadNoisyChannelData() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	var noisyChannelFile *os.File
	if pwd != "/" {
		noisyChannelFile, err = os.OpenFile(pwd+"/"+sc.outputDir+"/"+"noisy_model.index", os.O_RDONLY|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	} else {
		noisyChannelFile, err = os.OpenFile(sc.outputDir+"/"+"noisy_model.index", os.O_RDONLY|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	}
	defer noisyChannelFile.Close()

	stat, err := os.Stat(noisyChannelFile.Name())
	if err != nil {
		return fmt.Errorf("error when getting file stat: %w", err)
	}
	buf := make([]byte, stat.Size()*2)
	noisyChannelFile.Read(buf)
	dec := gob.NewDecoder(bytes.NewReader(buf))
	err = dec.Decode(&sc.NoisyChannelModel)
	if err != nil {
		return fmt.Errorf("error when unmarshalling metadata ngram: %w", err)
	}

	return nil
}
