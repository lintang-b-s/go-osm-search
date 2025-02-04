package controllers

import "github.com/lintang-b-s/osm-search/pkg/datastructure"

type SearchService interface {
	Search(query string, k int, offset int) ([]datastructure.Node, error)
	Autocomplete(query string, k, offset int) ([]datastructure.Node, error)
	ReverseGeocoding(lat, lon float64) (datastructure.Node, error)
}
