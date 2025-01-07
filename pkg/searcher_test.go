package pkg

import (
	"log"
	"regexp"
	"testing"
	"time"

	"math/rand"

	"github.com/stretchr/testify/assert"
	bolt "go.etcd.io/bbolt"
)

const (
	outputDir = "lintang"
)

func LoadIndex() (*Searcher, *bolt.DB) {

	ngramLM := NewNGramLanguageModel("lintang")
	spellCorrector := NewSpellCorrector(ngramLM)

	db, err := bolt.Open("docs_store.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	bboltKV := NewKVDB(db)

	invertedIndex, err := NewDynamicIndex("lintang", 1e7, true, spellCorrector, IndexedData{}, bboltKV)
	if err != nil {
		log.Fatal(err)
	}

	err = spellCorrector.InitializeSpellCorrector(invertedIndex.TermIDMap.GetSortedTerms(), invertedIndex.GetTermIDMap())
	if err != nil {
		log.Fatal(err)
	}

	searcher := NewSearcher(invertedIndex, bboltKV, spellCorrector)
	return searcher, db
}

func TestFullTextSearch(t *testing.T) {
	searcher, db := LoadIndex()
	defer db.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()
	reg := regexp.MustCompile(`[^\w\s]+`)
	t.Run("Test full text query without spell correction", func(t *testing.T) {
		relevantDocs, err := searcher.FreeFormQuery("Duniq Fsntssi", 15)
		if err != nil {
			t.Error(err)
		}

		mostRelDoc := reg.ReplaceAllString(string(relevantDocs[0].Name[:]), "") + " " + reg.ReplaceAllString(string(relevantDocs[0].Address[:]), "") + " " +
			reg.ReplaceAllString(string(relevantDocs[0].City[:]), "") + " " + reg.ReplaceAllString(string(relevantDocs[0].Tipe[:]), "")
		assert.Contains(t, mostRelDoc, "Dunia Fantasi")
	})

	t.Run("Test full text query with spell correction", func(t *testing.T) {
		relevantDocs, err := searcher.FreeFormQuery("Duniu Fsntaso", 15)
		if err != nil {
			t.Error(err)
		}

		mostRelDoc := reg.ReplaceAllString(string(relevantDocs[0].Name[:]), "") + " " + reg.ReplaceAllString(string(relevantDocs[0].Address[:]), "") + " " +
			reg.ReplaceAllString(string(relevantDocs[0].City[:]), "") + " " + reg.ReplaceAllString(string(relevantDocs[0].Tipe[:]), "")
		assert.Contains(t, mostRelDoc, "Dunia Fantasi")
	})

}

func TestAutocomplete(t *testing.T) {
	searcher, db := LoadIndex()
	defer db.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()

	rand.Seed(time.Now().UnixNano())

	relevantDocs, err := searcher.Autocomplete("Monumen Nasi")
	if err != nil {
		t.Error(err)
	}
	mostRelDoc := string(relevantDocs[0].Name[:])
	assert.Contains(t, mostRelDoc, "Monumen Nasional")
}

var searchQuery = []string{
	"Taman Anggrek",
	"Universitas Indonesia",
	"Dunia Fantasi",
	"Stasiun",
	"Kebun BiNItsng", // coba spell corrector
	"Monumen Nasional",
	"Halim Perdana",
	"Bandar Udara",
	"Taman",
	"Buaya Lubang",
	"Mall",
	"TPU Tanah",
}

// go test -bench=./...
// BenchmarkFullTextSearchQuery-12    	    2930	    360077 ns/op	  413571 B/op	    1516 allocs/op
func BenchmarkFullTextSearchQuery(b *testing.B) {

	searcher, db := LoadIndex()
	defer db.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()

	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		randomIndex := rand.Intn(len(searchQuery))
		_, err := searcher.FreeFormQuery(searchQuery[randomIndex], 15)
		if err != nil {
			b.Fatal(err)
		}
	}

}

var autoCompleteQuery = []string{
	"Taman An",
	"Universitas In",
	"Dunia Fan",
	"Stasi",
	"Kebun Bin",
	"Monumen Nasio",
	"Halim Perd",
	"Bandar Uda",
	"Tam",
	"Buaya Lub",
	"Mall",
	"TPU Tan",
}

// BenchmarkAutocomplete-12    	    3816	    288859 ns/op	  246140 B/op	     819 allocs/op
func BenchmarkAutocomplete(b *testing.B) {

	searcher, db := LoadIndex()
	defer db.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()

	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		randomIndex := rand.Intn(len(autoCompleteQuery))
		_, err := searcher.Autocomplete(autoCompleteQuery[randomIndex])
		if err != nil {
			b.Fatal(err)
		}
	}

}

func BenchmarkFullTextSearchQueryWithoutDocs(b *testing.B) {

	searcher, db := LoadIndex()
	defer db.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()

	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		randomIndex := rand.Intn(len(searchQuery))
		_, err := searcher.FreeFormQueryWithoutDocs(searchQuery[randomIndex], 15)
		if err != nil {
			b.Fatal(err)
		}
	}
	//BenchmarkFullTextSearchQueryWithoutDocs-12    	    4718	    260868 ns/op	  323354 B/op	     232 allocs/op
}

func BenchmarkGetPostingList(b *testing.B) {
	searcher, db := LoadIndex()
	defer db.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()

	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 0; i < 5; i++ {
			termID := rand.Intn(10000)
			_, err = searcher.MainIndex.GetPostingList(termID)
			if err != nil {
				b.Fatal(err)
			}
		}

	}
}
