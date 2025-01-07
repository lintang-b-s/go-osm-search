package main

import (
	"flag"
	"log"
	"osm-search/pkg"

	bolt "go.etcd.io/bbolt"
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

	db, err := bolt.Open("docs_store.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(pkg.BBOLTDB_BUCKET))
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	bboltKV := pkg.NewKVDB(db)

	ngramLM := pkg.NewNGramLanguageModel(*outputDir)
	spellCorrectorBuilder := pkg.NewSpellCorrector(ngramLM)

	indexedData := pkg.NewIndexedData(ways, onylySearchNodes, nodeMap, tagIDMap)
	invertedIndex, _ := pkg.NewDynamicIndex(*outputDir, 1e7, false, spellCorrectorBuilder,
		indexedData, bboltKV)

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

}
