package main

import (
	"flag"
	"log"
	"os"
	"osm-search/pkg"
)

var (
	listenAddr = flag.String("listenaddr", ":5000", "server listen address")
	mapFile    = flag.String("f", "jabodetabek_big.osm.pbf", "openstreeetmap file")
	outputDir  = flag.String("o", "lintang", "output directory buat simpan inverted index, ngram, dll")
)

func main() {
	flag.Parse()
	ways, onylySearchNodes, nodeMap, tagIDMap, err := pkg.ParseOSM(*mapFile)
	if err != nil {
		log.Fatal(err)
	}

	docsBuffer := make([]byte, 0, 16*1024)
	file, err := os.OpenFile(*outputDir+"/"+"docs_store.fdx", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	documentStoreIO := pkg.NewDiskWriterReader(docsBuffer, file)
	documentStore := pkg.NewDocumentStore(documentStoreIO, *outputDir)

	ngramLM := pkg.NewNGramLanguageModel(*outputDir)
	spellCorrectorBuilder := pkg.NewSpellCorrector(ngramLM)

	indexedData := pkg.NewIndexedData(ways, onylySearchNodes, nodeMap, tagIDMap)
	invertedIndex, _ := pkg.NewDynamicIndex(*outputDir, 1e7, false, spellCorrectorBuilder,
		indexedData, documentStore)

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

	if err = <-errChan; err != nil {
		log.Fatal(err)
	}

	err = invertedIndex.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = documentStore.Close()
	if err != nil {
		log.Fatal(err)
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
