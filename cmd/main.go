package main

import (
	"flag"
	"log"
	"osm-search/pkg"
)

var (
	listenAddr = flag.String("listenaddr", ":5000", "server listen address")
	mapFile    = flag.String("f", "jabodetabek.osm.pbf", "openstreeetmap file buat road network graphnya")
)

func main() {
	flag.Parse()
	ways, onylySearchNodes, nodeMap, err := pkg.ParseOSM(*mapFile)
	if err != nil {
		log.Fatal(err)
	}
	invertedIndex := pkg.NewDynamicIndex("lintang", 1e7)
	err = invertedIndex.SipmiBatchIndex(ways, onylySearchNodes, nodeMap)
	if err != nil {
		log.Fatal(err)
	}
	err = invertedIndex.Close()
	if err != nil {
		log.Fatal(err)
	}
}
