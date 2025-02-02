package searcher

import (
	"errors"
	"log"
	"osm-search/pkg/index"
	"osm-search/pkg/kvdb"
	"strings"
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

	bboltKV := kvdb.NewKVDB(db)

	invertedIndex, err := index.NewDynamicIndex("lintang", 1e7, true, spellCorrector, index.IndexedData{}, bboltKV)
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

// copy dulu directory "lintang" & "docs_store.db" ke directory ini
func TestFullTextSearch(t *testing.T) {
	searcher, db := LoadIndex()
	defer db.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()
	t.Run("Test full text query without spell correction", func(t *testing.T) {
		relevantDocs, err := searcher.FreeFormQuery("Dunia Fantasi", 15)
		if err != nil {
			t.Error(err)
		}

		mostRelDoc := relevantDocs[0].Name + " " + relevantDocs[0].Address + " " +
			" " + relevantDocs[0].Tipe
		assert.Contains(t, mostRelDoc, "Dunia Fantasi")
	})

	t.Run("Test full text query with spell correction", func(t *testing.T) {
		relevantDocs, err := searcher.FreeFormQuery("Duniu Fsntaso", 15)
		if err != nil {
			t.Error(err)
		}

		mostRelDoc := relevantDocs[0].Name + " " + relevantDocs[0].Address + " " +
			" " + relevantDocs[0].Tipe
		assert.Contains(t, mostRelDoc, "Dunia Fantasi")
	})

	tests := []struct {
		name  string
		query string

		wantRes string
		wantErr error
	}{
		{
			name:  "success 1",
			query: "Kebun Bibatqng Raginan",

			wantRes: "Kebun Binatang Ragunan",
			wantErr: nil,
		},
		{
			name:  "error",
			query: "",

			wantRes: "",
			wantErr: errors.New("query is empty"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relevantDocs, err := searcher.FreeFormQuery(tt.query, 15)
			if err != nil {
				assert.Equal(t, tt.wantErr, err)
				return
			}
			mostRelDoc := relevantDocs[0].Name + " " + relevantDocs[0].Address + " " +
				" " + relevantDocs[0].Tipe
			assert.Contains(t, mostRelDoc, tt.wantRes)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

// copy dulu directory "lintang" & "docs_store.db" ke directory ini
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

	tests := []struct {
		name  string
		query string

		wantRes string
		wantErr error
	}{
		{
			name:  "success 1",
			query: "Kebun Binatang Ra",

			wantRes: "kebun binatang ragunan",
			wantErr: nil,
		},
		{
			name:  "success 2",
			query: "Taman Min",

			wantRes: "taman mini indonesia",
			wantErr: nil,
		},
		{
			name:  "error",
			query: "",

			wantRes: "",
			wantErr: errors.New("query is empty"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relevantDocs, err := searcher.Autocomplete(tt.query)
			if err != nil {
				assert.Equal(t, tt.wantErr, err)
				return
			}

			isContain := false
			for _, doc := range relevantDocs {
				relDocName := doc.Name + " " + doc.Address + " " +
					doc.Tipe

				if strings.Contains(strings.ToLower(relDocName), tt.wantRes) {
					isContain = true
					break

				}
			}
			assert.Equal(t, true, isContain)
			assert.Equal(t, tt.wantErr, err)
		})
	}
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
	b.StopTimer()

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

// BenchmarkAutocomplete-12    	    4250	    237504 ns/op	  266636 B/op	     685 allocs/op
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
	b.StopTimer()

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
