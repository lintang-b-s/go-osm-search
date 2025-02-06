package usecases

import (
	"github.com/lintang-b-s/osm-search/pkg/datastructure"

	"go.uber.org/zap"
)

type SearcherService struct {
	log      *zap.Logger
	searcher Searcher
}

func New(log *zap.Logger, searcher Searcher) *SearcherService {
	return &SearcherService{
		log:      log,
		searcher: searcher,
	}
}

func (s *SearcherService) Search(query string, k, offset int) ([]datastructure.Node, error) {
	return s.searcher.FreeFormQuery(query, k, offset)
}

func (s *SearcherService) Autocomplete(query string, k, offset int) ([]datastructure.Node, error) {
	return s.searcher.Autocomplete(query, k, offset)
}

func (s *SearcherService) ReverseGeocoding(lat, lon float64) (datastructure.Node, error) {
	return s.searcher.ReverseGeocoding(lat, lon)
}

func (s *SearcherService) NearestNeighboursRadiusWithFeatureFilter(k, offset int, lat, lon, radius float64, 
	featureType string) ([]datastructure.Node, error) {
	return s.searcher.NearestNeighboursRadiusWithFeatureFilter(k,offset, lat, lon, radius, featureType)
}
