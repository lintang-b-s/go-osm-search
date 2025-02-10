package usecases

import (
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/geofence"
)

type Searcher interface {
	FreeFormQuery(query string, k, offset int) ([]datastructure.Node, error)
	Autocomplete(query string, k, offset int) ([]datastructure.Node, error)
	ReverseGeocoding(lat, lon float64) (datastructure.Node, error)
	NearestNeighboursRadiusWithFeatureFilter(k, offset int, lat, lon, radius float64, featureType string) ([]datastructure.Node, error)
}

type GeofenceIndex interface {
	AddFence(name string)
	DeleteFence(name string)
	Search(name string, lat, lon float64, fencePointID string) ([]geofence.FenceStatusObj, error)
	UpdateFencePoint(name string, lat, lon float64, fencePointID string) error
	AddFencePoint(name, fencePointName string, lat, lon, radius float64) error
}
