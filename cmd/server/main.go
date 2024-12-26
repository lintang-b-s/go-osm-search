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

	invertedIndex, err := pkg.NewDynamicIndex("lintang", 1e7, kvDB, true)
	if err != nil {
		log.Fatal(err)
}

	searcher := pkg.NewSearcher(invertedIndex, kvDB)
	nodes, err := searcher.FreeFormQuery("Taman Anggrek Ragunan", 15)
	if err != nil {
		log.Fatal(err)
	}
	for _, node := range nodes {
		fmt.Println(string(node.Address[:]))
		fmt.Println(node.Lat, node.Lon)
		fmt.Println(string(node.Name[:]))
	}
}
