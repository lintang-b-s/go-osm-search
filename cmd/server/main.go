package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"osm-search/pkg"
	"sync"

	bolt "go.etcd.io/bbolt"
)

var (
	outputDir = flag.String("o", "lintang", "output directory buat simpan inverted index, ngram, dll")
)

const (
	BUFFER_POOL_SIZE = 64
)

var (
	bufferPool BufferPool
)

type BufferPool interface {
	GetBuffer() *bytes.Buffer
	PutBuffer(*bytes.Buffer)
}

type syncPoolBufPool struct {
	pool       *sync.Pool
	makeBuffer func() interface{}
}

func NewSyncPool(buf_size int) BufferPool {
	var newPool syncPoolBufPool

	newPool.makeBuffer = func() interface{} {
		var b bytes.Buffer
		b.Grow(buf_size)
		return &b
	}
	newPool.pool = &sync.Pool{}
	newPool.pool.New = newPool.makeBuffer

	return &newPool
}

func (bp *syncPoolBufPool) GetBuffer() (b *bytes.Buffer) {
	pool_object := bp.pool.Get()

	b, ok := pool_object.(*bytes.Buffer)
	if !ok {
		b = bp.makeBuffer().(*bytes.Buffer)
	}
	return
}

func (bp *syncPoolBufPool) PutBuffer(b *bytes.Buffer) {
	bp.pool.Put(b)
}

func main() {

	ngramLM := pkg.NewNGramLanguageModel("lintang")
	spellCorrector := pkg.NewSpellCorrector(ngramLM)

	db, err := bolt.Open("docs_store.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	bboltKV := pkg.NewKVDB(db)

	invertedIndex, err := pkg.NewDynamicIndex("lintang", 1e7, true, spellCorrector, pkg.IndexedData{},
		bboltKV)
	if err != nil {
		log.Fatal(err)
	}

	err = spellCorrector.InitializeSpellCorrector(invertedIndex.TermIDMap.GetSortedTerms(), invertedIndex.GetTermIDMap())
	if err != nil {
		log.Fatal(err)
	}

	searcher := pkg.NewSearcher(invertedIndex, bboltKV, spellCorrector)
	err = searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()
	var nodes = make([]pkg.Node, 0, 100)
	nodes1, err := searcher.FreeFormQuery("Kebun BiNItsng RaHuban ", 15) // Kebun binatang ragunan
	if err != nil {
		log.Fatal(err)
	}
	nodes2, err := searcher.FreeFormQuery("Duniq Fsntssi", 15)
	if err != nil {
		log.Fatal(err)
	}

	nodes3, err := searcher.FreeFormQuery("Taman", 15)
	if err != nil {
		log.Fatal(err)
	}

	nodes4, err := searcher.FreeFormQuery("Stasiyn Gambur", 15)
	if err != nil {
		log.Fatal(err)
	}

	nodes = append(nodes, nodes1...)
	nodes = append(nodes, nodes2...)
	nodes = append(nodes, nodes3...)
	nodes = append(nodes, nodes4...)

	for _, node := range nodes {
		fmt.Println(string(node.Address[:]))
		fmt.Println(node.Lat, node.Lon)
		fmt.Println(string(node.Name[:]))
		fmt.Println(string(node.Tipe[:]))
	}
}
