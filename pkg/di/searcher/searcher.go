package searcher_di

import (
	"context"
	"osm-search/pkg/http/usecases"
	"osm-search/pkg/index"
	"osm-search/pkg/kvdb"
	"osm-search/pkg/searcher"
)

func New(ctx context.Context, db *kvdb.KVDB) (usecases.Searcher, error) {
	ngramLM := searcher.NewNGramLanguageModel("lintang")
	spellCorrector := searcher.NewSpellCorrector(ngramLM)
	invertedIndex, err := index.NewDynamicIndex("lintang", 1e7, true, spellCorrector, index.IndexedData{},
		db)
	if err != nil {
		return nil, err
	}

	err = spellCorrector.InitializeSpellCorrector(invertedIndex.TermIDMap.GetSortedTerms(), invertedIndex.GetTermIDMap())
	if err != nil {
		return nil, err
	}

	osmSearcher := searcher.NewSearcher(invertedIndex, db, spellCorrector)
	err = osmSearcher.LoadMainIndex()
	if err != nil {
		return nil, err
	}

	cleanup := func() {
		osmSearcher.Close()
	}

	go func() {
		<-ctx.Done()
		cleanup()
	}()

	return osmSearcher, nil
}
