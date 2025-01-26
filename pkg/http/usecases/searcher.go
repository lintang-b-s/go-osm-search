package usecases

import (
	"osm-search/pkg/datastructure"

	"go.uber.org/zap"
)

type SearcherService struct {
	log *zap.Logger
	searcher Searcher
}

func New(log *zap.Logger, searcher Searcher) *SearcherService {
	return &SearcherService{
		log: log,
		searcher: searcher,
	}
}



func (s *SearcherService) Search(query string, k int) ([]datastructure.Node, error) {
	return s.searcher.FreeFormQuery(query, k)
}


func (s *SearcherService) Autocomplete(query string) ([]datastructure.Node, error) {
	return s.searcher.Autocomplete(query)
}





