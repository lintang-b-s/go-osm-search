package geo

import (
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/paulmach/osm"
)

type NodeMapContainer struct {
	nodeMap map[int64]*osm.Node
}

func (nm *NodeMapContainer) SetNodeMap(nodeMap map[int64]*osm.Node) {
	nm.nodeMap = nodeMap
}

func (nm *NodeMapContainer) GetNode(id int64) *osm.Node {
	return nm.nodeMap[id]
}

type OSMWay struct {
	ID              int64
	NodeIDs         []int64
	TagMap          map[string]string
	ContainWikidata bool
}

func NewOSMWay(id int64, nodeIDs []int64, tagMap map[string]string, wikiData bool) OSMWay {
	return OSMWay{
		ID:              id,
		NodeIDs:         nodeIDs,
		TagMap:          tagMap,
		ContainWikidata: wikiData,
	}
}

type OSMNode struct {
	ID              int64
	Lat             float64
	Lon             float64
	TagMap          map[string]string
	ContainWikiData bool
}

func NewOSMNode(id int64, lat float64, lon float64, tagMap map[string]string, wikiData bool) OSMNode {
	return OSMNode{
		Lat:             lat,
		Lon:             lon,
		TagMap:          tagMap,
		ID:              id,
		ContainWikiData: wikiData,
	}
}

type OSMSpatialIndex struct {
	StreetRtree                 *datastructure.Rtree
	AdministrativeBoundaryRtree *datastructure.Rtree
}

type Boundary struct {
	Province    string      `json:"province"`
	District    string      `json:"district"`
	SubDistrict string      `json:"sub_district"`
	Village     string      `json:"village"`
	PostalCode  string      `json:"postal_code"`
	Border      [][]float64 `json:"border"`
}

func NewBoundary(province, district, subDistrict, village, postalCode string, border [][]float64) Boundary {
	return Boundary{
		Province:    province,
		District:    district,
		SubDistrict: subDistrict,
		Village:     village,
		PostalCode:  postalCode,
		Border:      border,
	}
}
