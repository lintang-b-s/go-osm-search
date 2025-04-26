package searcher

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/lintang-b-s/osm-search/pkg"
)

const (
	UNKNOWN_TOKEN = "<UNK>"
	START_TOKEN   = "<s>"
	END_TOKEN     = "</s>"
)

type NGramLanguageModel struct {
	vocabulary []int
	wordCounts map[int]int
	termIDMap  *pkg.IDMap
	Data       NGramData
	outputDir  string
}

type NGramData struct {
	UniGramCount  map[int]int
	BiGramCount   map[[2]int]int
	TriGramCount  map[[3]int]int
	QuadGramCount map[[4]int]int
	TotalWordFreq int
}

func NewNGramLanguageModel(outDir string) *NGramLanguageModel {
	return &NGramLanguageModel{
		vocabulary: make([]int, 0),
		wordCounts: make(map[int]int),
		outputDir:  outDir,
	}
}

func (lm *NGramLanguageModel) SetTermIDMap(termIDMap *pkg.IDMap) {
	lm.termIDMap = termIDMap
}

func (lm *NGramLanguageModel) addWord(word int) {
	lm.wordCounts[word]++
	if _, ok := lm.wordCounts[word]; !ok {
		lm.vocabulary = append(lm.vocabulary, word)
	}
}

// countWords. menghitung frekuensi setiap kata dalam corpus
func (lm *NGramLanguageModel) countWords(tokenizedDocs [][]string) {
	for _, doc := range tokenizedDocs {

		for _, word := range doc {
			wordID := lm.termIDMap.GetID(word)
			lm.addWord(wordID)
		}
	}
}

/*
getWordsWithNPlusFreq. return kata-kata yang memiliki frekuensi lebih dari countThresold. kata yang kurang dari thresold jadi <UNK>
*/
func (lm *NGramLanguageModel) getWordsWithNPlusFreq(tokenizedDocs [][]string, countThresold int) []int {
	lm.countWords(tokenizedDocs)
	closedWords := make([]int, 0)
	for word, count := range lm.wordCounts {
		if count >= countThresold {
			closedWords = append(closedWords, word)
		}
	}
	return closedWords
}

// replaceOOVWordsWithUNK. mengganti kata-kata yang frequensinya < 2 dengan <UNK>
func (lm *NGramLanguageModel) replaceOOVWordsWithUNK(tokenizedDocs [][]string, vocabulary []int) [][]int {
	replacedTokenizedDocs := [][]int{}

	unknownTokenID := lm.termIDMap.GetID(UNKNOWN_TOKEN)
	vocabSet := make(map[int]bool)
	for _, word := range vocabulary {
		vocabSet[word] = true
	}

	for _, doc := range tokenizedDocs {
		replacedDoc := []int{}
		for _, token := range doc {
			tokenID := lm.termIDMap.GetID(token)
			if _, ok := vocabSet[tokenID]; ok {
				replacedDoc = append(replacedDoc, tokenID)
			} else {
				replacedDoc = append(replacedDoc, unknownTokenID)
			}
		}
		replacedTokenizedDocs = append(replacedTokenizedDocs, replacedDoc)
	}
	return replacedTokenizedDocs
}

func (lm *NGramLanguageModel) PreProcessData(tokenizedDocs [][]string, countThresold int) [][]int {
	lm.countWords(tokenizedDocs)
	vocabulary := lm.getWordsWithNPlusFreq(tokenizedDocs, countThresold)
	replacedTokenizedDocs := lm.replaceOOVWordsWithUNK(tokenizedDocs, vocabulary)
	return replacedTokenizedDocs
}

func (lm *NGramLanguageModel) countUnigram(data [][]int) {

	var nGrams = make(map[int]int)

	for _, doc := range data {

		doc = lm.addStartEndToken(doc, 1)

		m := len(doc)
		for i := 0; i < m; i++ {
			nGram := doc[i]

			if _, ok := nGrams[nGram]; !ok {
				nGrams[nGram] = 1
			} else {
				nGrams[nGram]++
			}

			lm.Data.TotalWordFreq++
		}
	}

	lm.Data.UniGramCount = nGrams
}

func (lm *NGramLanguageModel) countBigram(data [][]int) {

	var nGrams = make(map[[2]int]int)

	for _, doc := range data {

		doc = lm.addStartEndToken(doc, 2)

		m := len(doc) - 2 + 1
		for i := 0; i < m; i++ {
			var nGram [2]int

			copy(nGram[:], doc[i:i+2])

			if _, ok := nGrams[nGram]; !ok {
				nGrams[nGram] = 1
			} else {
				nGrams[nGram]++
			}
		}
	}

	lm.Data.BiGramCount = nGrams
}

func (lm *NGramLanguageModel) countTrigram(data [][]int) {

	var nGrams = make(map[[3]int]int)

	for _, doc := range data {

		doc = lm.addStartEndToken(doc, 3)

		m := len(doc) - 3 + 1
		for i := 0; i < m; i++ {
			var nGram [3]int

			copy(nGram[:], doc[i:i+3])

			if _, ok := nGrams[nGram]; !ok {
				nGrams[nGram] = 1
			} else {
				nGrams[nGram]++
			}
		}
	}

	lm.Data.TriGramCount = nGrams
}

func (lm *NGramLanguageModel) countQuadgram(data [][]int) {

	var nGrams = make(map[[4]int]int)

	for _, doc := range data {

		doc = lm.addStartEndToken(doc, 4)

		m := len(doc) - 4 + 1
		for i := 0; i < m; i++ {
			var nGram [4]int

			copy(nGram[:], doc[i:i+4])

			if _, ok := nGrams[nGram]; !ok {
				nGrams[nGram] = 1
			} else {
				nGrams[nGram]++
			}
		}
	}

	lm.Data.QuadGramCount = nGrams
}

// estimateProbability. menghitung probabilitas nextWord berdasarkan previous tokens.
func (lm *NGramLanguageModel) estimateProbability(nextWord int, previousNGram []int, n int) float64 {
	switch n {
	case 1:
		var ngramCount int
		if count, ok := lm.Data.UniGramCount[nextWord]; ok {
			ngramCount = count
		} else {
			ngramCount = 0
		}

		denominator := lm.Data.TotalWordFreq
		numerator := ngramCount
		probability := float64(numerator) / float64(denominator)
		return probability

	case 2:
		var prevNgramCount int
		if count, ok := lm.Data.UniGramCount[previousNGram[0]]; ok {
			prevNgramCount = count
		} else {
			return 0
		}
		denominator := prevNgramCount

		nGram := [2]int{previousNGram[0], nextWord}

		var nGramCount int
		if count, ok := lm.Data.BiGramCount[nGram]; ok {
			nGramCount = count
		} else {
			nGramCount = 0
		}

		numerator := nGramCount

		probability := float64(numerator) / float64(denominator)
		return probability
	case 3:
		prevNGram := [2]int{previousNGram[0], previousNGram[1]}
		var prevNgramCount int
		if count, ok := lm.Data.BiGramCount[prevNGram]; ok {
			prevNgramCount = count
		} else {
			return 0
		}
		denominator := prevNgramCount

		nGram := [3]int{prevNGram[0], prevNGram[1], nextWord}

		var nGramCount int
		if count, ok := lm.Data.TriGramCount[nGram]; ok {
			nGramCount = count
		} else {
			nGramCount = 0
		}

		numerator := nGramCount

		probability := float64(numerator) / float64(denominator)
		return probability
	case 4:
		prevNGram := [3]int{previousNGram[0], previousNGram[1], previousNGram[2]}
		var prevNgramCount int
		if count, ok := lm.Data.TriGramCount[prevNGram]; ok {
			prevNgramCount = count
		} else {
			return 0
		}
		denominator := prevNgramCount

		nGram := [4]int{prevNGram[0], prevNGram[1], prevNGram[2], nextWord}

		var nGramCount int
		if count, ok := lm.Data.QuadGramCount[nGram]; ok {
			nGramCount = count
		} else {
			nGramCount = 0
		}

		numerator := nGramCount

		probability := float64(numerator) / float64(denominator)
		return probability
	}
	return 0
}

// estimate probability of a sequence of words in query.
func (lm *NGramLanguageModel) estimateQueryProbability(query []int) float64 {
	probability := 0.0

	for i := 4; i < len(query); i++ {
		fourgram := lm.stupidBackoff(query[i], query[i-3:i], 4)
		probability += math.Log(fourgram)
	}

	return probability
}

func (lm *NGramLanguageModel) GetQueryNgramProbability(queries [][]int, n int) []float64 {

	var sentencesProbabilities = make([]float64, 0, len(queries))
	for _, sentence := range queries {
		sentence = lm.addStartEndToken(sentence, n)
		probability := lm.estimateQueryProbability(sentence)
		sentencesProbabilities = append(sentencesProbabilities, probability)
	}
	return sentencesProbabilities
}

func (lm *NGramLanguageModel) stupidBackoff(nextWord int, prevNgrams []int, n int) float64 {
	newProb := 0.0
	lambda := 1.0
	for ; n > 0; n-- {

		newProb = lambda * lm.estimateProbability(nextWord, prevNgrams, n)

		if newProb != 0 {
			break
		}
		if len(prevNgrams) == 1 {
			prevNgrams = []int{}
			lambda = lambda * 0.4
			continue
		}
		prevNgrams = prevNgrams[1:]
		lambda = lambda * 0.4
	}
	return newProb
}

// MakeCountMatrix. menghitung frekuensi n-gram dari data
func (lm *NGramLanguageModel) MakeCountMatrix(data [][]int) {

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		lm.countUnigram(data)
	}()

	go func() {
		defer wg.Done()
		lm.countBigram(data)
	}()

	go func() {
		defer wg.Done()
		lm.countTrigram(data)
	}()

	go func() {
		defer wg.Done()
		lm.countQuadgram(data)
	}()
	wg.Wait()

}

// addStartEndToken. menambahkan token <s> sebanyak n dan </s> pada awal dan akhir dokumen
func (lm *NGramLanguageModel) addStartEndToken(doc []int, n int) []int {
	startToken := []int{}
	startTokenID := lm.termIDMap.GetID(START_TOKEN)
	endTokenID := lm.termIDMap.GetID(END_TOKEN)

	for i := 0; i < n; i++ {
		startToken = append(startToken, startTokenID)
	}
	doc = append(startToken, doc...)
	doc = append(doc, endTokenID)
	return doc
}

func (lm *NGramLanguageModel) SaveNGramData() error {

	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(lm.Data)
	if err != nil {
		return fmt.Errorf("error when marshalling metadata: %w", err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var ngramFile *os.File

	if pwd != "/" {
		ngramFile, err = os.OpenFile(pwd+"/"+lm.outputDir+"/"+"ngram.index", os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	} else {
		ngramFile, err = os.OpenFile(lm.outputDir+"/"+"ngram.index", os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	}
	defer ngramFile.Close()

	err = ngramFile.Truncate(0)
	if err != nil {
		return err
	}

	_, err = ngramFile.Write(buf.Bytes())

	return err
}

func (lm *NGramLanguageModel) LoadNGramData() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	var ngramFile *os.File
	if pwd != "/" {
		ngramFile, err = os.OpenFile(pwd+"/"+lm.outputDir+"/"+"ngram.index", os.O_RDONLY|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	} else {
		ngramFile, err = os.OpenFile(lm.outputDir+"/"+"ngram.index", os.O_RDONLY|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	}
	defer ngramFile.Close()

	stat, err := os.Stat(ngramFile.Name())
	if err != nil {
		return fmt.Errorf("error when getting file stat: %w", err)
	}
	buf := make([]byte, stat.Size()*2)
	ngramFile.Read(buf)
	dec := gob.NewDecoder(bytes.NewReader(buf))
	err = dec.Decode(&lm.Data)
	if err != nil {
		return fmt.Errorf("error when unmarshalling metadata ngram: %w", err)
	}

	return nil
}
