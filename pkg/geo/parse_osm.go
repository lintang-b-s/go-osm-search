package geo

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"

	"github.com/k0kubun/go-ansi"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/schollz/progressbar/v3"
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

var ValidSearchTags = map[string]bool{
	"amenity":          true,
	"building":         true,
	"sport":            true,
	"tourism":          true,
	"leisure":          true,
	"boundary":         true,
	"landuse":          true,
	"craft":            true,
	"aeroway":          true,
	"historic":         true,
	"residential":      true,
	"railway":          true,
	"shop":             true,
	"junction":         true,
	"route":            true,
	"ferry":            true,
	"highway":          true,
	"motorcar":         true,
	"motor_vehicle":    true,
	"access":           true,
	"industrial":       true,
	"service":          true,
	"healthcare":       true,
	"office":           true,
	"public_transport": true,
	"waterway":         true,
	"water":            true,
	"telecom":          true,
	"power":            true,
	"place":            true,
	"geological":       true,
	"emergency":        true,
	"bulding":          true,
	"aerialway":        true,
	"barrier":          true,
}

var ValidNodeSearchTag = map[string]bool{
	"historic": true,
	"name":     true,
}

type OSMWay struct {
	ID      int64
	NodeIDs []int64
	TagMap  map[string]string
}

func NewOSMWay(id int64, nodeIDs []int64, tagMap map[string]string) OSMWay {
	return OSMWay{
		NodeIDs: nodeIDs,
		TagMap:  tagMap,
	}
}

type OSMNode struct {
	ID     int64
	Lat    float64
	Lon    float64
	TagMap map[string]string
}

func NewOSMNode(id int64, lat float64, lon float64, tagMap map[string]string) OSMNode {
	return OSMNode{
		Lat:    lat,
		Lon:    lon,
		TagMap: tagMap,
	}
}

type OSMSpatialIndex struct {
	StreetRtree        *datastructure.Rtree
	KelurahanRtree     *datastructure.Rtree
	KecamatanRtree     *datastructure.Rtree
	KotaKabupatenRtree *datastructure.Rtree
	ProvinsiRtree      *datastructure.Rtree
	CountryRtree       *datastructure.Rtree
}

type OsmRelation struct {
	Name        string
	ways        []int64
	AdminLevel  string
	BoundaryLat []float64
	BoundaryLon []float64
	PostalCode  string
}

func ParseOSM(mapfile string) ([]OSMWay, []OSMNode, NodeMapContainer, *pkg.IDMap, OSMSpatialIndex, []OsmRelation, error) {
	var TagIDMap *pkg.IDMap = pkg.NewIDMap()

	streetRtree := datastructure.NewRtree(25, 50, 2)
	kelurahanRtree := datastructure.NewRtree(25, 50, 2)
	kecamatanRtree := datastructure.NewRtree(25, 50, 2)
	kotaKabupatenRtree := datastructure.NewRtree(25, 50, 2)
	provinsiRtree := datastructure.NewRtree(25, 50, 2)
	countryRtree := datastructure.NewRtree(25, 50, 2)

	f, err := os.Open(mapfile)

	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, []OsmRelation{}, err
	}

	defer f.Close()

	count := 0

	ctr := NodeMapContainer{
		nodeMap: make(map[int64]*osm.Node),
	}

	ways := []OSMWay{}
	bar := progressbar.NewOptions(5,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][1/3]Parsing osm objects..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	bar.Add(1)

	// process relation administrative Boundaries
	wayAdministrativeBoundary := make(map[int64]bool)

	relations := []OsmRelation{}

	scanner := osmpbf.New(context.Background(), f, 1)

	for scanner.Scan() {
		o := scanner.Object()
		if o.ObjectID().Type() == osm.TypeRelation {
			rel := o.(*osm.Relation)
			isAdministrativeBoundary := false
			for _, tag := range rel.Tags {
				if tag.Key == "boundary" && tag.Value == "administrative" {
					isAdministrativeBoundary = true
				}
			}

			if !isAdministrativeBoundary {
				continue
			}

			name := rel.Tags.Find("name")
			if name == "" || strings.Contains(name, "UNKNOWN") {
				continue
			}
			adminLevel, err := strconv.Atoi(rel.Tags.Find("admin_level"))
			if err != nil || adminLevel < 2 || adminLevel > 7 {
				continue
			}

			postalCode := rel.Tags.Find("postal_code")

			wayIDs := []int64{}
			for _, m := range rel.Members {
				if m.Type == osm.TypeWay && m.Role == "outer" {

					wayAdministrativeBoundary[m.Ref] = true
					wayIDs = append(wayIDs, m.Ref)

				}
			}

			relations = append(relations, OsmRelation{
				Name:        rel.Tags.Find("name"),
				ways:        wayIDs,
				AdminLevel:  rel.Tags.Find("admin_level"),
				BoundaryLat: []float64{},
				BoundaryLon: []float64{},
				PostalCode:  postalCode,
			})

		}
	}
	bar.Add(1)

	scanErr := scanner.Err()
	if scanErr != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, relations, err
	}

	scanner.Close()

	// process osm ways
	wayNodesMap := make(map[osm.NodeID]bool)

	boundaryWayMap := make(map[int64]OSMWay)

	fWay, err := os.Open(mapfile)
	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, relations, err
	}
	defer fWay.Close()

	scannerWay := osmpbf.New(context.Background(), fWay, 1)
	defer scannerWay.Close()

	for scannerWay.Scan() {
		o := scannerWay.Object()
		tipe := o.ObjectID().Type()

		if tipe != osm.TypeWay {
			continue
		}

		tag := o.(*osm.Way).TagMap()

		_, ok := wayAdministrativeBoundary[int64(o.(*osm.Way).ID)]
		if ok {
			nodeIDs := []int64{}
			for _, node := range o.(*osm.Way).Nodes {
				wayNodesMap[node.ID] = true
				nodeIDs = append(nodeIDs, int64(node.ID))
			}

			way := NewOSMWay(int64(o.(*osm.Way).ID), nodeIDs, tag)
			boundaryWayMap[int64(o.(*osm.Way).ID)] = way
		}

		name, _, _, _, _ := GetNameAddressTypeFromOSMWay(tag)
		if name == "" {
			continue
		}

		if !checkIsWayAlowed(tag) {
			continue
		}

		nodeIDs := []int64{}
		for _, node := range o.(*osm.Way).Nodes {
			wayNodesMap[node.ID] = true
			nodeIDs = append(nodeIDs, int64(node.ID))
		}
		way := NewOSMWay(int64(o.(*osm.Way).ID), nodeIDs, tag)
		ways = append(ways, way)

		count++
	}

	scanErr = scannerWay.Err()
	if scanErr != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, relations, err
	}
	scannerWay.Close()

	bar.Add(1)
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, relations, err
	}

	scanner = osmpbf.New(context.Background(), f, 1)
	defer scanner.Close()

	onlyOsmNodes := []OSMNode{}
	for scanner.Scan() {
		o := scanner.Object()
		if o.ObjectID().Type() == osm.TypeNode {
			node := o.(*osm.Node)
			if _, ok := wayNodesMap[node.ID]; ok {
				ctr.nodeMap[int64(o.(*osm.Node).ID)] = o.(*osm.Node)
			}
			name, _, _, _, _ := GetNameAddressTypeFromOSMWay(node.TagMap())
			if name == "" {
				continue
			}
			if checkIsNodeAlowed(node.TagMap()) {
				lat := node.Lat
				lon := node.Lon
				tag := node.TagMap()

				onlyOsmNodes = append(onlyOsmNodes, NewOSMNode(int64(o.(*osm.Node).ID), lat, lon, tag))
			}

		}
	}

	bar.Add(1)

	scanErr = scanner.Err()
	if scanErr != nil {
		return []OSMWay{}, []OSMNode{}, NodeMapContainer{}, &pkg.IDMap{}, OSMSpatialIndex{}, relations, err
	}

	// process poligon administrative boundary & rtree administrative boundary
	for relID, rel := range relations {

		boundaryLat, boundaryLon := []float64{}, []float64{}
		for _, relway := range rel.ways {
			wway, ok := boundaryWayMap[relway]
			if !ok {
				continue
			}
			for _, nodeID := range wway.NodeIDs {
				node := ctr.GetNode(nodeID)
				boundaryLat = append(boundaryLat, node.Lat)
				boundaryLon = append(boundaryLon, node.Lon)
			}
		}

		if len(boundaryLat) == 0 || len(boundaryLon) == 0 {
			continue
		}

		relations[relID].BoundaryLat = append(relations[relID].BoundaryLat, boundaryLat...)
		relations[relID].BoundaryLon = append(relations[relID].BoundaryLon, boundaryLon...)

		sortedBoundaryLat, sortedBoundaryLon := make([]float64, len(boundaryLat)), make([]float64, len(boundaryLon))

		copy(sortedBoundaryLat, boundaryLat)
		copy(sortedBoundaryLon, boundaryLon)

		sort.Float64s(sortedBoundaryLat)
		sort.Float64s(sortedBoundaryLon)
		centerLat, centerLon := sortedBoundaryLat[len(sortedBoundaryLat)/2], sortedBoundaryLon[len(sortedBoundaryLon)/2]

		rtreeLeaf := datastructure.OSMObject{
			ID:  relID,
			Lat: centerLat,
			Lon: centerLon,
		}

		// // bound = [minLat, minLon, maxLat, maxLon]
		bound := datastructure.NewRtreeBoundingBox(2, []float64{sortedBoundaryLat[0], sortedBoundaryLon[0]},
			[]float64{sortedBoundaryLat[len(sortedBoundaryLat)-1], sortedBoundaryLon[len(sortedBoundaryLon)-1]})

		// insert r-tree per administrative level
		if rel.AdminLevel == "7" {
			kelurahanRtree.InsertR(bound, rtreeLeaf)
		} else if rel.AdminLevel == "6" {
			kecamatanRtree.InsertR(bound, rtreeLeaf)
		} else if rel.AdminLevel == "5" {
			kotaKabupatenRtree.InsertR(bound, rtreeLeaf)
		} else if rel.AdminLevel == "4" {
			provinsiRtree.InsertR(bound, rtreeLeaf)
		} else if rel.AdminLevel == "2" {
			countryRtree.InsertR(bound, rtreeLeaf)
		}

	}

	// process osm streets & rtree streets. buat menentukan nama jalan dari osm way kalau di tag "addr:street" gak ada.
	for idx, way := range ways {
		lat, lon := []float64{}, []float64{}
		for _, nodeID := range way.NodeIDs {
			node := ctr.GetNode(nodeID)
			lat = append(lat, node.Lat)
			lon = append(lon, node.Lon)
		}
		sort.Float64s(lat)
		sort.Float64s(lon)

		midLat, midLon := MidPoint(lat[0], lon[0], lat[len(lat)-1], lon[len(lon)-1])

		rtreeLeaf := datastructure.OSMObject{
			ID:  idx,
			Lat: midLat,
			Lon: midLon,
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

			streetRtree.InsertR(bound, rtreeLeaf)
		}
	}

	// update adress dari osm ways dan osm nodes
	spatialIndex := OSMSpatialIndex{
		StreetRtree:        streetRtree,
		KelurahanRtree:     kelurahanRtree,
		KecamatanRtree:     kecamatanRtree,
		KotaKabupatenRtree: kotaKabupatenRtree,
		ProvinsiRtree:      provinsiRtree,
		CountryRtree:       countryRtree,
	}

	bar.Add(1)
	return ways, onlyOsmNodes, ctr, TagIDMap, spatialIndex, relations, nil
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
