package searcher_di

import (
	"context"

	"github.com/lintang-b-s/osm-search/pkg/http/usecases"
	"github.com/lintang-b-s/osm-search/pkg/index"
	"github.com/lintang-b-s/osm-search/pkg/kvdb"
	"github.com/lintang-b-s/osm-search/pkg/searcher"
)

func New(ctx context.Context, db *kvdb.KVDB, scoring searcher.SimiliarityScoring) (usecases.Searcher, error) {
	ngramLM := searcher.NewNGramLanguageModel("lintang")
	spellCorrector := searcher.NewSpellCorrector(ngramLM, "lintang")
	invertedIndex, err := index.NewDynamicIndex("lintang", 1e7, true, spellCorrector, index.IndexedData{},
		db)
	if err != nil {
		return nil, err
	}

	err = spellCorrector.InitializeSpellCorrector(invertedIndex.TermIDMap.GetSortedTerms(), invertedIndex.GetTermIDMap())
	if err != nil {
		return nil, err
	}

	osmSearcher := searcher.NewSearcher(invertedIndex, db, spellCorrector, scoring)
	err = osmSearcher.LoadMainIndex()
	if err != nil {
		return nil, err
	}
	err = spellCorrector.LoadNoisyChannelData()
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
