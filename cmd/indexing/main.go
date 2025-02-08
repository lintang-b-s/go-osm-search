package main

import (
	"context"
	"flag"
	"log"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/lintang-b-s/osm-search/pkg/geo"
	"github.com/lintang-b-s/osm-search/pkg/index"
	"github.com/lintang-b-s/osm-search/pkg/kvdb"
	"github.com/lintang-b-s/osm-search/pkg/searcher"

	bolt "go.etcd.io/bbolt"
)

var (
	mapFile    = flag.String("f", "jabodetabek.osm.pbf", "openstreeetmap file")
	outputDir  = flag.String("o", "lintang", "output directory buat simpan inverted index, ngram, dll")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile = flag.String("memprofile", "", "write memory profile to this file")
)

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		// https://go.dev/blog/pprof
		// ./bin/osm-search-indexer -f "jabodetabek_big.osm.pbf" -cpuprofile=osmsearchcpu.prof -memprofile=osmsearchmem.mprof
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

	}

	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		os.Mkdir(*outputDir, 0700)
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

	indexedData := index.NewIndexedData(ways, onylySearchNodes, nodeMap, tagIDMap, spatialIndex, osmRelations)
	invertedIndex, _ := index.NewDynamicIndex(*outputDir, 1e7, false, spellCorrectorBuilder,
		indexedData, bboltKV)

	// indexing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	allSearchNodes, err := invertedIndex.SpimiBatchIndex(ctx)
	if err != nil {
		panic(err)
	}

	if *memprofile != "" {
		*memprofile = strings.Replace(*memprofile, ".mprof", "_indexing.mprof", -1)
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()

	}

	cleanup := func() {
		err = invertedIndex.Close()
		if err != nil {
			panic(err)
		}
	}

	defer cleanup()

	ngramLM.SetTermIDMap(invertedIndex.GetTermIDMap())
	err = invertedIndex.BuildSpellCorrectorAndNgram(ctx, allSearchNodes, spatialIndex, osmRelations)
	if err != nil {
		panic(err)
	}

	if *memprofile != "" {
		*memprofile = strings.Replace(*memprofile, ".mprof", "_spell.mprof", -1)

		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()

	}
}
