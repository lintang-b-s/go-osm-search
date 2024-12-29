package pkg

import (
	"log"
	"testing"
	"time"

	"math/rand"

	"github.com/dgraph-io/badger/v4"
)

func LoadIndex() (*Searcher, *badger.DB) {
	db, err := badger.Open(badger.DefaultOptions("osm-searchdb"))
	if err != nil {
		log.Fatal(err)
	}

	kvDB := NewKVDB(db)

	ngramLM := NewNGramLanguageModel("lintang")
	spellCorrector := NewSpellCorrector(ngramLM)

	invertedIndex, err := NewDynamicIndex("lintang", 1e7, kvDB, true, spellCorrector, IndexedData{})
	if err != nil {
		log.Fatal(err)
	}

	err = spellCorrector.InitializeSpellCorrector(invertedIndex.TermIDMap.GetSortedTerms(), invertedIndex.GetTermIDMap())
	if err != nil {
		log.Fatal(err)
	}

	searcher := NewSearcher(invertedIndex, kvDB, spellCorrector)
	return searcher, db
}

// go test -bench=./...
func BenchmarkFullTextQuery(b *testing.B) {
	searchQuery := []string{
		"Taman Anggrek",
		"Universitas Indonesia",
		"Dunia Fantasi",
		"Stasiun",
		"Jalan Senopati",
		"Kebun Banatung", // coba spell corrector 
		"Monumen Nasional",
		"Halim Perdana",
		"Bandar Udara",
		"Taman",
		"Buaya Lubang",
		"Mall",
		"TPU Tanah",
	}
	searcher, db := LoadIndex()
	defer db.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()

	rand.Seed(time.Now().UnixNano())

	for n := 0; n < b.N; n++ {
		randomIndex := rand.Intn(len(searchQuery))
		_, err := searcher.FreeFormQuery(searchQuery[randomIndex], 15)
		if err != nil {
			b.Fatal(err)
		}
	}
	// 5054429 ns/op
}
