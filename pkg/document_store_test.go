package pkg

import (
	"log"
	"math/rand"
	"testing"
	"time"
)

// go test -bench=./...
func BenchmarkDiskSeek(b *testing.B) {

	searcher, f, docStore := LoadIndex()

	defer f.Close()
	defer docStore.Close()
	err := searcher.LoadMainIndex()
	if err != nil {
		log.Fatal(err)
	}
	defer searcher.Close()

	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 0; i < 15; i++ {
			docID := rand.Intn(200000)
			_, err = searcher.DocStore.GetDoc(docID)
			if err != nil {
				b.Fatal(err)
			}
		}

	}

}
