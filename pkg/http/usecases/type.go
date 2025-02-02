package usecases

import (
	"osm-search/pkg/datastructure"
)

type Searcher interface {
	FreeFormQuery(query string, k int) ([]datastructure.Node, error)
	Autocomplete(query string) ([]datastructure.Node, error)
	ReverseGeocoding(lat, lon float64) (datastructure.Node, error)
}
