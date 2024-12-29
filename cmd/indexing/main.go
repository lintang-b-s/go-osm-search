package main

import (
	"flag"
	"log"
	"osm-search/pkg"

	"github.com/dgraph-io/badger/v4"
)

var (
	listenAddr = flag.String("listenaddr", ":5000", "server listen address")
	mapFile    = flag.String("f", "jabodetabek_big.osm.pbf", "openstreeetmap file")
	outputDir = flag.String("o", "lintang", "output directory buat simpan inverted index, ngram, dll")
)

func main() {
	flag.Parse()
	ways, onylySearchNodes, nodeMap, tagIDMap, err := pkg.ParseOSM(*mapFile)
	if err != nil {
		log.Fatal(err)
	}

	db, err := badger.Open(badger.DefaultOptions("osm-searchdb"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	kvDB := pkg.NewKVDB(db)

	ngramLM := pkg.NewNGramLanguageModel(*outputDir)
	spellCorrectorBuilder := pkg.NewSpellCorrector(ngramLM)

	indexedData := pkg.NewIndexedData(ways, onylySearchNodes, nodeMap, tagIDMap)
	invertedIndex, _ := pkg.NewDynamicIndex(*outputDir, 1e7, kvDB, false, spellCorrectorBuilder, indexedData)

	// indexing
	var errChan = make(chan error, 1)
	go func() {
		errChan <- invertedIndex.BuildSpellCorrectorAndNgram()
		close(errChan)
	}()

	err = invertedIndex.SpimiBatchIndex()
	if err != nil {
		log.Fatal(err)
	}
	err = invertedIndex.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = <-errChan
	if err != nil {
		log.Fatal(err)
	}
}
