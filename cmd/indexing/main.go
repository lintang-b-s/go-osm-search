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
	mapFile   = flag.String("f", "jabodetabek.osm.pbf", "openstreeetmap file")
	outputDir = flag.String("o", "lintang", "output directory buat simpan inverted index, ngram, dll")
)

func main() {
	flag.Parse()

	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		os.Mkdir(*outputDir, 0755)
	}

	ways, onylySearchNodes, nodeMap, tagIDMap, spatialIndex, osmRelations, err := geo.ParseOSM(*mapFile)
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

	allSearchNodes, err := invertedIndex.SpimiBatchIndex(ctx, spatialIndex, osmRelations)
	if err != nil {
		panic(err)
	}

	cleanup := func() {
		err = invertedIndex.Close()
		if err != nil {
			panic(err)
		}
	}

	defer cleanup()

	ngramLM.SetTermIDMap(invertedIndex.GetTermIDMap())
	err = invertedIndex.BuildSpellCorrectorAndNgram(ctx,allSearchNodes, spatialIndex, osmRelations)
	if err != nil {
		panic(err)
	}
}
