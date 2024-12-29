package main

import (
	"fmt"
	"log"
	"osm-search/pkg"

	"github.com/dgraph-io/badger/v4"
)

func main() {
	db, err := badger.Open(badger.DefaultOptions("osm-searchdb"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	kvDB := pkg.NewKVDB(db)

	ngramLM := pkg.NewNGramLanguageModel("lintang")
	spellCorrector := pkg.NewSpellCorrector(ngramLM)

	invertedIndex, err := pkg.NewDynamicIndex("lintang", 1e7, kvDB, true, spellCorrector, pkg.IndexedData{})
	if err != nil {
		log.Fatal(err)
	}

	err = spellCorrector.InitializeSpellCorrector(invertedIndex.TermIDMap.GetSortedTerms(), invertedIndex.GetTermIDMap())
	if err != nil {
		log.Fatal(err)
	}

	searcher := pkg.NewSearcher(invertedIndex, kvDB, spellCorrector)
	err = searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()
	nodes, err := searcher.FreeFormQuery("Kebun BInuTung RaGunin ", 15) // Kebun binatang ragunan
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
