package searcher

import (
	"errors"
	"io/fs"
	"math"
	"os"
	"testing"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/stretchr/testify/assert"
)

func prepare(t *testing.T) {
	_, err := os.Stat("test")

	if errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir("test", 0700)
		if err != nil {
			t.Error(err)
		}
	}

	pwd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	indexFilePath := pwd + "/" + "test" + "/" + "ngram" + ".index"
	metadataFilePath := pwd + "/" + "test" + "/" + "ngram" + ".metadata"

	_, err = os.Stat(indexFilePath)

	if err == nil {
		err = os.Remove(indexFilePath)
		if err != nil {
			t.Error(err)
		}
		err = os.Remove(metadataFilePath)
		if err != nil {
			t.Error(err)
		}
	}

}

func TestCountUnigram(t *testing.T) {
	prepare(t)
	t.Run("success count unigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		unigramCount := lm.Data.UniGramCount

		expected := map[int]int{
			0:  3,
			1:  3,
			3:  1,
			4:  1,
			5:  5,
			6:  4,
			11: 1,
			12: 1,
		}

		assert.Equal(t, expected, unigramCount)

		assert.Equal(t, 19, lm.Data.TotalWordFreq)
	})
}

func TestCountBigram(t *testing.T) {
	prepare(t)
	t.Run("success count bigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countBigram(data)

		bigramCount := lm.Data.BiGramCount

		expected := map[[2]int]int{
			[2]int{0, 0}:   3,
			[2]int{0, 3}:   1,
			[2]int{0, 6}:   1,
			[2]int{0, 11}:  1,
			[2]int{3, 4}:   1,
			[2]int{4, 5}:   1,
			[2]int{6, 5}:   1,
			[2]int{5, 5}:   3,
			[2]int{11, 12}: 1,
			[2]int{12, 6}:  1,
			[2]int{6, 6}:   2,
			[2]int{5, 1}:   2,
			[2]int{6, 1}:   1,
		}

		assert.Equal(t, expected, bigramCount)
	})

}

func TestCountTrigram(t *testing.T) {
	prepare(t)
	t.Run("success count trigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}
		// prefix : 000
		// suffix : 1

		lm.termIDMap = pkg.NewIDMap()

		lm.countTrigram(data)

		trigramCount := lm.Data.TriGramCount

		expected := map[[3]int]int{
			[3]int{0, 0, 0}:   3,
			[3]int{0, 0, 3}:   1,
			[3]int{0, 0, 6}:   1,
			[3]int{0, 0, 11}:  1,
			[3]int{3, 4, 5}:   1,
			[3]int{4, 5, 1}:   1,
			[3]int{6, 5, 5}:   1,
			[3]int{5, 5, 5}:   2,
			[3]int{5, 5, 1}:   1,
			[3]int{11, 12, 6}: 1,
			[3]int{12, 6, 6}:  1,
			[3]int{6, 6, 6}:   1,
			[3]int{6, 6, 1}:   1,
			[3]int{0, 3, 4}:   1,
			[3]int{0, 6, 5}:   1,
			[3]int{0, 11, 12}: 1,
		}

		assert.Equal(t, expected, trigramCount)
	})
}

func TestCountQuadgram(t *testing.T) {
	prepare(t)
	t.Run("success count trigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}
		// prefix : 000
		// suffix : 1

		lm.termIDMap = pkg.NewIDMap()

		lm.countQuadgram(data)

		quadGramCount := lm.Data.QuadGramCount

		expected := map[[4]int]int{
			[4]int{0, 0, 0, 0}:   3,
			[4]int{0, 0, 0, 3}:   1,
			[4]int{0, 0, 0, 6}:   1,
			[4]int{0, 0, 3, 4}:   1,
			[4]int{0, 3, 4, 5}:   1,
			[4]int{3, 4, 5, 1}:   1,
			[4]int{0, 0, 6, 5}:   1,
			[4]int{0, 6, 5, 5}:   1,
			[4]int{6, 5, 5, 5}:   1,
			[4]int{5, 5, 5, 1}:   1,
			[4]int{5, 5, 5, 5}:   1,
			[4]int{0, 0, 0, 11}:  1,
			[4]int{0, 0, 11, 12}: 1,
			[4]int{0, 11, 12, 6}: 1,
			[4]int{11, 12, 6, 6}: 1,
			[4]int{12, 6, 6, 6}:  1,
			[4]int{6, 6, 6, 1}:   1,
		}

		assert.Equal(t, expected, quadGramCount)
	})
}

func TestEstimateProb(t *testing.T) {
	t.Run("success estimate prob unigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		prob := lm.estimateProbability(6, []int{}, 1)
		assert.Equal(t, float64(4)/float64(19), prob)
	})

	t.Run("zero prob estimate prob unigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		prob := lm.estimateProbability(999, []int{}, 1)
		assert.Equal(t, 0.0, prob)
	})

	t.Run("success estimate prob bigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		prob := lm.estimateProbability(6, []int{6}, 2)
		assert.Equal(t, float64(2)/float64(4), prob)
	})

	t.Run("zero prob estimate prob bigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		prob := lm.estimateProbability(30, []int{6}, 2)
		assert.Equal(t, 0.0, prob)
	})

	t.Run("success estimate prob trigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		lm.countTrigram(data)

		prob := lm.estimateProbability(6, []int{6, 6}, 3)
		assert.Equal(t, float64(1)/float64(2), prob)
	})

	t.Run("zero prob estimate prob trigram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		lm.countTrigram(data)

		prob := lm.estimateProbability(99, []int{6, 6}, 3)
		assert.Equal(t, float64(0)/float64(2), prob)
	})

	t.Run("success estimate prob quadgram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		lm.countTrigram(data)

		lm.countQuadgram(data)

		prob := lm.estimateProbability(5, []int{5, 5, 5}, 4)
		assert.Equal(t, float64(1)/float64(2), prob)
	})

	t.Run("zero prob estimate prob quadgram", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		lm.countTrigram(data)

		lm.countQuadgram(data)

		prob := lm.estimateProbability(99, []int{12, 6, 6}, 4)
		assert.Equal(t, float64(0)/float64(1), prob)
	})
}

func TestStupidBackoff(t *testing.T) {
	t.Run("success stupid backoff", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		lm.countTrigram(data)

		lm.countQuadgram(data)

		prob := lm.stupidBackoff(12, []int{9, 10, 11}, 4)
		// backoff ke bigram * 0.4*0.4
		assert.Equal(t, float64(1)*0.4*0.4/(float64(1)), prob)
	})

}

func TestEstimateQueryProbability(t *testing.T) {
	t.Run("success estimate query probability", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		lm.countTrigram(data)

		lm.countQuadgram(data)

		query := []int{5, 5, 12, 11}
		query = lm.addStartEndToken(query, 4)

		// 0 0 0 0 5 5 12 15 1

		// 0 0 0 5
		// 0 0 5 5
		// 0 5 5 12
		// 5 5 12 11
		// 5 12 11 1

		prob := lm.estimateQueryProbability(query)

		expectedProb := 0.0 + math.Log(0.4*0.4*0.4*float64(5)/float64(19)) +
			math.Log(0.4*0.4*float64(3)/float64(5)) + math.Log(0.4*0.4*0.4*float64(1)/float64(19)) +
			math.Log(0.4*0.4*0.4*float64(1)/float64(19)) + math.Log(0.4*0.4*0.4*float64(3)/float64(19))

		assert.Equal(t, expectedProb, prob)
	})
}

func TestEstimateQueryProbabilities(t *testing.T) {

	t.Run("success estimate query probability", func(t *testing.T) {
		lm := NewNGramLanguageModel("test")

		data := [][]int{
			{3, 4, 5},
			{6, 5, 5, 5, 5},
			{11, 12, 6, 6, 6},
		}

		lm.termIDMap = pkg.NewIDMap()

		lm.countUnigram(data)

		lm.countBigram(data)

		lm.countTrigram(data)

		lm.countQuadgram(data)

		query := []int{5, 5, 12, 11}

		// 0 0 0 0 5 5 12 15 1

		// 0 0 0 5
		// 0 0 5 5
		// 0 5 5 12
		// 5 5 12 11
		// 5 12 11 1

		queryTwo := []int{11, 12, 6, 6, 5}

		// 0 0 0 0 11 12 6 6 6 1

		// 0 0 0 11
		// 0 0 11 12
		// 0 11 12 6
		// 11 12 6 6
		// 12 6 6 5
		// 6 6 5 1

		expectedProbOne := 0.0 + math.Log(0.4*0.4*0.4*float64(5)/float64(19)) +
			math.Log(0.4*0.4*float64(3)/float64(5)) + math.Log(0.4*0.4*0.4*float64(1)/float64(19)) +
			math.Log(0.4*0.4*0.4*float64(1)/float64(19)) + math.Log(0.4*0.4*0.4*float64(3)/float64(19))

		expectedProbTwo := 0.0 + math.Log(float64(1)/float64(3)) +
			math.Log(float64(1)/float64(1)) + math.Log(float64(1)/float64(1)) +
			math.Log(float64(1)/float64(1)) + math.Log(0.4*0.4*float64(1)/float64(4)) +
			math.Log(0.4*0.4*float64(2)/float64(5))

		probs := lm.GetQueryNgramProbability([][]int{query, queryTwo}, 4)

		assert.InDelta(t, expectedProbOne, probs[0], 0.1)

		assert.InDelta(t, expectedProbTwo, probs[1], 0.1)
	})
}
