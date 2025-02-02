package controllers

import "osm-search/pkg/datastructure"

type SearchService interface {
	Search(query string, k int) ([]datastructure.Node, error)
	Autocomplete(query string) ([]datastructure.Node, error)
	ReverseGeocoding(lat, lon float64) (datastructure.Node, error)
}
