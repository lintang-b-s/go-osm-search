package main

import (
	"context"
	"flag"
	"os"
	"osm-search/pkg/geo"
	"osm-search/pkg/index"
	"osm-search/pkg/kvdb"
	"osm-search/pkg/searcher"

	bolt "go.etcd.io/bbolt"
)

var (
	mapFile   = flag.String("f", "jabodetabek_big.osm.pbf", "openstreeetmap file")
	outputDir = flag.String("o", "lintang", "output directory buat simpan inverted index, ngram, dll")
)

func main() {
	flag.Parse()

	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		os.Mkdir(*outputDir, 0755)
	}

	ways, onylySearchNodes, nodeMap, tagIDMap, err := geo.ParseOSM(*mapFile)
	if err != nil {
		panic(err)
	}

	db, err := bolt.Open("docs_store.db", 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(kvdb.BBOLTDB_BUCKET))
		return err
	})
	if err != nil {
		panic(err)
	}

	bboltKV := kvdb.NewKVDB(db)

	ngramLM := searcher.NewNGramLanguageModel(*outputDir)
	spellCorrectorBuilder := searcher.NewSpellCorrector(ngramLM)

	indexedData := index.NewIndexedData(ways, onylySearchNodes, nodeMap, tagIDMap)
	invertedIndex, _ := index.NewDynamicIndex(*outputDir, 1e7, false, spellCorrectorBuilder,
		indexedData, bboltKV)

	// indexing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buildSpellCorrector := func() error {
		var errChan = make(chan error, 1)

		go func() {
			errChan <- invertedIndex.BuildSpellCorrectorAndNgram()
		}()

		select {
		case <-ctx.Done():
			<-errChan
			return nil
		case err = <-errChan:
			return err
		}
	}

	c := make(chan error, 1)
	go func() {
		c <- buildSpellCorrector()
	}()

	err = invertedIndex.SpimiBatchIndex()
	if err != nil {
		panic(err)
	}

	cleanup := func() {
		err = invertedIndex.Close()
		if err != nil {
			panic(err)
		}
	}

	if err = <-c; err != nil {
		cleanup()
		panic(err)
	}

	cleanup()

}
