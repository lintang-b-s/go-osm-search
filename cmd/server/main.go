package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"osm-search/pkg"
)

var (
	outputDir = flag.String("o", "lintang", "output directory buat simpan inverted index, ngram, dll")
)

func main() {

	ngramLM := pkg.NewNGramLanguageModel("lintang")
	spellCorrector := pkg.NewSpellCorrector(ngramLM)

	docsBuffer := make([]byte, 0, 16*1024)
	file, err := os.OpenFile(*outputDir+"/"+"docs_store.fdx", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	documentStoreIO := pkg.NewDiskWriterReader(docsBuffer, file)
	err = documentStoreIO.PreloadFile()
	if err != nil {
		log.Fatal(err)
	}
	documentStore := pkg.NewDocumentStore(documentStoreIO, *outputDir)
	defer documentStore.Close()
	err = documentStore.LoadMeta()
	if err != nil {
		log.Fatal(err)
	}

	invertedIndex, err := pkg.NewDynamicIndex("lintang", 1e7, true, spellCorrector, pkg.IndexedData{},
		documentStore)
	if err != nil {
		log.Fatal(err)
	}

	err = spellCorrector.InitializeSpellCorrector(invertedIndex.TermIDMap.GetSortedTerms(), invertedIndex.GetTermIDMap())
	if err != nil {
		log.Fatal(err)
	}

	searcher := pkg.NewSearcher(invertedIndex, documentStore, spellCorrector)
	err = searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()
	nodes, err := searcher.FreeFormQuery("Kebun BiNItsng RaHuban ", 15) // Kebun binatang ragunan
	if err != nil {
		log.Fatal(err)
	}
	for _, node := range nodes {
		fmt.Println(string(node.Address[:]))
		fmt.Println(node.Lat, node.Lon)
		fmt.Println(string(node.Name[:]))
		fmt.Println(string(node.Tipe[:]))
	}
}

/*

db, err := badger.Open(badger.DefaultOptions("osm-searchdb"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	kvDB := pkg.NewKVDB(db)
*/
