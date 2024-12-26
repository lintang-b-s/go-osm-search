package main

import (
	"flag"
	"log"
	"osm-search/pkg"

	"github.com/dgraph-io/badger/v4"
)

var (
	listenAddr = flag.String("listenaddr", ":5000", "server listen address")
	mapFile    = flag.String("f", "surakarta.osm.pbf", "openstreeetmap file buat road network graphnya")
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

	invertedIndex, _ := pkg.NewDynamicIndex("lintang", 1e7, kvDB, false)
	err = invertedIndex.SipmiBatchIndex(ways, onylySearchNodes, nodeMap, tagIDMap)
	if err != nil {
		log.Fatal(err)
	}
	err = invertedIndex.Close()
	if err != nil {
		log.Fatal(err)
	}
}
