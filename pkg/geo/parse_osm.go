package geo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"

	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
)

func ParseOSM(mapfile string, mapBoundaryFile string) ([]OSMWay, []OSMNode, NodeMapContainer, *pkg.IDMap, OSMSpatialIndex, []Boundary, error) {
	var TagIDMap *pkg.IDMap = pkg.NewIDMap()

	streetRtree := datastructure.NewRtree(25, 50, 2)
	regionRtree := datastructure.NewRtree(25, 50, 2)

	count := 0

	ctr := NodeMapContainer{
		nodeMap: make(map[int64]osm.Node),
	}

	ways := []OSMWay{}
	log.Printf("Parsing osm way objects...\n")

	regionBoundaries := make([]Boundary, 0)

	onlyOsmNodes := []OSMNode{}

	// process osm ways
	wayNodesMap := make(map[int64]bool)

	fWay, err := os.Open(mapfile)
	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, regionBoundaries, err
	}
	defer fWay.Close()

	scannerWay := osmpbf.New(context.Background(), fWay, 1)

	fmt.Printf("\n")
	log.Printf("Parsing osm way objects...\n")
	for scannerWay.Scan() {
		o := scannerWay.Object()
		tipe := o.ObjectID().Type()
		switch tipe {
		case osm.TypeNode:
			{
				node := o.(*osm.Node)

				name, _, _, _, _ := GetNameAddressTypeFromOSMWay(node.TagMap())
				if name == "" {
					continue
				}
				if checkIsNodeAlowed(node.TagMap()) {
					lat := node.Lat
					lon := node.Lon
					tag := node.TagMap()

					containWikiData := containWikiData(o.(*osm.Node).Tags)

					onlyOsmNodes = append(onlyOsmNodes, NewOSMNode(int64(o.(*osm.Node).ID), lat, lon, tag, containWikiData))
				}
			}
		case osm.TypeWay:
			{

				tag := o.(*osm.Way).TagMap()

				name, _, _, _, _ := GetNameAddressTypeFromOSMWay(tag)
				if _, ok := tag["highway"]; !ok && name == "" {
					continue
				}

				if !checkIsWayAlowed(tag) {
					continue
				}

				nodeIDs := []int64{}
				for _, node := range o.(*osm.Way).Nodes {
					wayNodesMap[int64(node.ID)] = true
					nodeIDs = append(nodeIDs, int64(node.ID))
				}

				containWikiData := containWikiData(o.(*osm.Way).Tags)

				way := NewOSMWay(int64(o.(*osm.Way).ID), nodeIDs, tag, containWikiData)
				ways = append(ways, way)

				count++
			}
		}

	}

	scanErr := scannerWay.Err()
	if scanErr != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, regionBoundaries, err
	}
	scannerWay.Close()

	_, err = fWay.Seek(0, io.SeekStart)
	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, regionBoundaries, err
	}

	scannerWay = osmpbf.New(context.Background(), fWay, 1)
	defer scannerWay.Close()

	for scannerWay.Scan() {
		o := scannerWay.Object()
		tipe := o.ObjectID().Type()
		switch tipe {
		case osm.TypeNode:
			{
				node := o.(*osm.Node)
				if _, ok := wayNodesMap[int64(node.ID)]; ok {
					ctr.nodeMap[int64(node.ID)] = *node
				}
			}
		}
	}

	scanErr = scannerWay.Err()
	if scanErr != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, regionBoundaries, err
	}

	log.Printf("Parsing osm way objects done\n")

	// process poligon administrative boundary & rtree administrative boundary
	indoBoundaryFile, err := os.Open(mapBoundaryFile)
	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, regionBoundaries, err
	}

	defer indoBoundaryFile.Close()

	var indoRegionsBoundary []Boundary

	indoBoundaryFileStat, err := indoBoundaryFile.Stat()
	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, regionBoundaries, err
	}

	buffer := bytes.NewBuffer(make([]byte, indoBoundaryFileStat.Size()))
	_, err = indoBoundaryFile.Read(buffer.Bytes())
	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, regionBoundaries, err
	}

	err = json.Unmarshal(buffer.Bytes(), &indoRegionsBoundary)
	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, regionBoundaries, err
	}

	for relID, village := range indoRegionsBoundary {
		regionBoundaries = append(regionBoundaries, NewBoundary(
			village.Province, village.District, village.SubDistrict, village.Village,
			village.PostalCode, village.Border,
		))

		boundaryLat, boundaryLon := []float64{}, []float64{}
		for _, relway := range village.Border {
			boundaryLat = append(boundaryLat, relway[1])
			boundaryLon = append(boundaryLon, relway[0])
		}

		if len(boundaryLat) == 0 || len(boundaryLon) == 0 {
			continue
		}

		sortedBoundaryLat, sortedBoundaryLon := make([]float64, len(boundaryLat)), make([]float64, len(boundaryLon))

		copy(sortedBoundaryLat, boundaryLat)
		copy(sortedBoundaryLon, boundaryLon)

		sort.Float64s(sortedBoundaryLat)
		sort.Float64s(sortedBoundaryLon)
		centerLat, centerLon := sortedBoundaryLat[len(sortedBoundaryLat)/2], sortedBoundaryLon[len(sortedBoundaryLon)/2]

		rtreeLeaf := datastructure.OSMObject{
			ID:       relID,
			Lat:      centerLat,
			Lon:      centerLon,
			OsmBound: [2][]float64{boundaryLat, boundaryLon},
		}

		// // bound = [minLat, minLon, maxLat, maxLon]
		bound := datastructure.NewRtreeBoundingBox(2, []float64{sortedBoundaryLat[0], sortedBoundaryLon[0]},
			[]float64{sortedBoundaryLat[len(sortedBoundaryLat)-1], sortedBoundaryLon[len(sortedBoundaryLon)-1]})

		// insert r-tree per administrative level
		regionRtree.InsertLeaf(bound, rtreeLeaf, false)
	}

	// process osm streets & rtree streets. buat menentukan nama jalan dari osm way kalau di tag "addr:street" gak ada.
	for idx, way := range ways {
		lat, lon := []float64{}, []float64{}
		latLons := [][]float64{}
		for _, nodeID := range way.NodeIDs {
			node := ctr.GetNode(nodeID)
			lat = append(lat, node.Lat)
			lon = append(lon, node.Lon)
			latLons = append(latLons, []float64{node.Lat, node.Lon})
		}
		sort.Float64s(lat)
		sort.Float64s(lon)

		midLat, midLon := MidPoint(lat[0], lon[0], lat[len(lat)-1], lon[len(lon)-1])

		rtreeLeaf := datastructure.OSMObject{
			ID:              idx,
			Lat:             midLat,
			Lon:             midLon,
			BoundaryLatLons: latLons,
		}

		highway, ok := way.TagMap["highway"]
		if ok && (highway == "motorway" ||
			highway == "trunk" ||
			highway == "primary" ||
			highway == "secondary" ||
			highway == "tertiary" ||
			highway == "unclassified" ||
			highway == "residential" ||
			highway == "living_street" ||
			highway == "service" ||
			highway == "motorway_link" ||
			highway == "trunk_link" ||
			highway == "primary_link" ||
			highway == "secondary_link" ||
			highway == "tertiary_link") {
			bound := datastructure.NewRtreeBoundingBox(2, []float64{rtreeLeaf.Lat - 0.0001,
				rtreeLeaf.Lon - 0.0001}, []float64{rtreeLeaf.Lat + 0.0001, rtreeLeaf.Lon + 0.0001})
			rtreeLeaf.Tag = make(map[int]int)
			rtreeLeaf.Tag[ROAD_PRIORITY_KEY] = roadTypeMaxSpeed[highway]
			streetRtree.InsertLeaf(bound, rtreeLeaf, false)
		}
	}

	// update adress dari osm ways dan osm nodes
	spatialIndex := OSMSpatialIndex{
		StreetRtree:                 streetRtree,
		AdministrativeBoundaryRtree: regionRtree,
	}

	fmt.Printf("\n")
	log.Printf("processing osm relation & way objects done \n")

	return ways, onlyOsmNodes, ctr, TagIDMap, spatialIndex, indoRegionsBoundary, nil
}

func containWikiData(tags osm.Tags) bool {
	return tags.Find("wikidata") != "" ||
		tags.Find("wikipedia") != "" ||
		tags.Find("wikimedia_commons") != ""
}

// TODO: ngikutin Nominatim, infer dari administrative boundary & nearest street dari osm way.
func GetNameAddressTypeFromOSMWay(tag map[string]string) (string, string, string, string, string) {
	name := tag["name"]
	shortName, ok := tag["short_name"]
	if ok {
		name = fmt.Sprintf("%s (%s)", name, shortName)
	}

	street, ok := tag["addr:street"]

	postalCode, ok := tag["addr:postcode"]

	houseNumber := tag["addr:housenumber"]

	tipe := GetOSMObjectType(tag)
	return name, street, tipe, postalCode, houseNumber
}

func GetOSMObjectType(tag map[string]string) string {
	tipe, ok := tag["amenity"]
	if ok {
		return tipe
	}
	tipe, ok = tag["highway"]
	if ok {
		return tipe
	}
	// building tidak include (karena cuma yes/no)
	tipe, ok = tag["historic"]
	if ok {
		return tipe
	}
	tipe, ok = tag["sport"]
	if ok {
		return tipe
	}
	tipe, ok = tag["tourism"]
	if ok {
		return tipe
	}
	tipe, ok = tag["leisure"]
	if ok {
		return tipe
	}
	tipe, ok = tag["landuse"]
	if ok {
		return tipe
	}
	tipe, ok = tag["craft"]
	if ok {
		return tipe
	}
	tipe, ok = tag["aeroway"]
	if ok {
		return tipe
	}
	tipe, ok = tag["residential"]
	if ok {
		return tipe
	}

	tipe, ok = tag["industrial"]
	if ok {
		return tipe
	}
	tipe, ok = tag["shop"]
	if ok {
		return tipe
	}
	return ""
}

func checkIsWayAlowed(tag map[string]string) bool {
	for k, _ := range tag {

		if ValidSearchTags[k] {
			return true
		}

	}
	return false
}

func checkIsNodeAlowed(tag map[string]string) bool {
	for k, _ := range tag {
		if ValidNodeSearchTag[k] {
			return true
		}
	}
	return false
}
