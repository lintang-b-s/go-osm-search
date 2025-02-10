package usecases

import (
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/geofence"

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
	return s.searcher.NearestNeighboursRadiusWithFeatureFilter(k, offset, lat, lon, radius, featureType)
}

type GeofenceService struct {
	geofenceIndex GeofenceIndex
}

func NewGeofenceService(geofenceIndex GeofenceIndex) *GeofenceService {
	return &GeofenceService{

		geofenceIndex: geofenceIndex,
	}
}

func (s *GeofenceService) AddFence(name string) error{
return 	s.geofenceIndex.AddFence(name)
}

func (s *GeofenceService) DeleteFence(name string) {
	s.geofenceIndex.DeleteFence(name)
}

func (s *GeofenceService) Search(name string, lat, lon float64, fencePointID string) ([]geofence.FenceStatusObj, error) {
	return s.geofenceIndex.Search(name, lat, lon, fencePointID)
}

func (s *GeofenceService) UpdateFencePoint(name string, lat, lon float64, fencePointID string) error {
	return s.geofenceIndex.UpdateFencePoint(name, lat, lon, fencePointID)
}

func (s *GeofenceService) AddFencePoint(name, fencePointName string, lat, lon, radius float64) error {
	return s.geofenceIndex.AddFencePoint(name, fencePointName, lat, lon, radius)
}
