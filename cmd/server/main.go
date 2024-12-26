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

	invertedIndex := pkg.NewDynamicIndex("lintang", 1e7, kvDB)

	searcher := pkg.NewSearcher(invertedIndex, kvDB)
	nodes, err := searcher.FreeFormQuery("Jalan", 10)
	if err != nil {
		log.Fatal(err)
	}
	for _, node := range nodes {
		fmt.Println(node)
	}
}
