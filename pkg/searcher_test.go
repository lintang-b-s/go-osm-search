package pkg

import (
	"log"
	"os"
	"testing"
	"time"

	"math/rand"
)

const (
	outputDir = "lintang"
)

func LoadIndex() (*Searcher, *os.File, *DocumentStore) {

	ngramLM := NewNGramLanguageModel("lintang")
	spellCorrector := NewSpellCorrector(ngramLM)

	docsBuffer := make([]byte, 0, 16*1024)
	file, err := os.OpenFile(outputDir+"/"+"docs_store.fdx", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}

	documentStoreIO := NewDiskWriterReader(docsBuffer, file)
	documentStore := NewDocumentStore(documentStoreIO, outputDir)
	documentStore.LoadMeta()

	invertedIndex, err := NewDynamicIndex("lintang", 1e7, true, spellCorrector, IndexedData{}, documentStore)
	if err != nil {
		log.Fatal(err)
	}

	err = spellCorrector.InitializeSpellCorrector(invertedIndex.TermIDMap.GetSortedTerms(), invertedIndex.GetTermIDMap())
	if err != nil {
		log.Fatal(err)
	}

	searcher := NewSearcher(invertedIndex, documentStore, spellCorrector)
	return searcher, file, documentStore
}

var searchQuery = []string{
	"Taman Anggrek",
	"Universitas Indonesia",
	"Dunia Fantasi",
	"Stasiun",
	"Kebun BiNItsng", // coba spell corrector
	// "Kebun Binatang",
	"Monumen Nasional",
	"Halim Perdana",
	"Bandar Udara",
	"Taman",
	"Buaya Lubang",
	"Mall",
	"TPU Tanah",
}

// go test -bench=./...
func BenchmarkFullTextQuery(b *testing.B) {

	searcher, f, docStore := LoadIndex()
	defer f.Close()
	defer docStore.Close()
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
	// 5054429 ns/op -> ganti badgerdb pake self made document store jadi 1739005 ns/op
}

func BenchmarkFullTextQueryWithoutSpellCorrector(b *testing.B) {
	var searchQuery = []string{
		"Taman Anggrek",
		"Universitas Indonesia",
		"Dunia Fantasi",
		"Stasiun",
		"Kebun Binatang",
		"Monumen Nasional",
		"Halim Perdana",
		"Bandar Udara",
		"Taman",
		"Buaya Lubang",
		"Mall",
		"TPU Tanah",
	}

	searcher, f, docStore := LoadIndex()
	defer f.Close()
	defer docStore.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()

	rand.Seed(time.Now().UnixNano())

	for n := 0; n < b.N; n++ {
		randomIndex := rand.Intn(len(searchQuery))
		_, err := searcher.FreeFormQueryWithoutSpellCorrection(searchQuery[randomIndex], 15)
		if err != nil {
			b.Fatal(err)
		}
	}
	// 1847172 ns/op
}
