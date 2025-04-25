package index

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"sort"
	"sync"
	"testing"

	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/geo"
	"github.com/lintang-b-s/osm-search/pkg/kvdb"
	"github.com/paulmach/osm"
	"github.com/stretchr/testify/assert"
	bolt "go.etcd.io/bbolt"
)

func init() {
	_, err := os.Stat("test")

	if errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir("test", 0700)
		if err != nil {
			panic(err)
		}
	}

}

func TestSpimiParseOSMNode(t *testing.T) {
	cases := []struct {
		inputNodes    datastructure.Node
		field         string
		lenDF         map[int]int
		expectedPairs [][]int
	}{
		{
			inputNodes: datastructure.Node{ID: 1, Name: "Jalan Sentosa Harapan Jalan Dunia Baru Jalan Mulwo Apel Jalan Kebun Jeruk Apel Jalan Pantai Ancol"},

			field: "name",
			lenDF: map[int]int{},
			expectedPairs: [][]int{
				{0, 1},
				{1, 1},
				{2, 1},
				{0, 1},
				{3, 1},
				{4, 1},
				{0, 1},
				{5, 1},
				{6, 1},
				{0, 1},
				{7, 1},
				{8, 1},
				{6, 1},
				{0, 1},
				{9, 1},
				{10, 1},
			},
		},
	}

	t.Run("Test Spimi Parse OSM Nodes", func(t *testing.T) {
		for _, c := range cases {
			spimi, err := NewDynamicIndex("test", 500, false, nil, NewIndexedData([]geo.OSMWay{}, []geo.OSMNode{}, geo.NodeMapContainer{},
				nil, geo.OSMSpatialIndex{}, []geo.Boundary{}), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			results := spimi.SpimiParseOSMNode(c.inputNodes, c.lenDF, &sync.RWMutex{}, c.field)
			assert.Equal(t, c.expectedPairs, results)
			assert.Equal(t, 1, len(c.lenDF))
			assert.Equal(t, 16, c.lenDF[1])
		}
	})
}

func TestSpimiParseOSMNodes(t *testing.T) {
	cases := []struct {
		inputNodes    []datastructure.Node
		field         string
		lenDF         map[int]int
		expectedPairs [][]int
	}{
		{
			inputNodes: []datastructure.Node{
				{ID: 1, Name: "Jalan Sentosa Harapan"},
				{ID: 2, Name: "Jalan Dunia Baru"},
				{ID: 3, Name: "Jalan Mulwo Apel"},
				{ID: 4, Name: "Jalan Kebun Jeruk Apel"},
				{ID: 5, Name: "Jalan Pantai Ancol"},
			},
			field: "name",
			lenDF: map[int]int{},
			expectedPairs: [][]int{
				{0, 1},
				{1, 1},
				{2, 1},
				{0, 2},
				{3, 2},
				{4, 2},
				{0, 3},
				{5, 3},
				{6, 3},
				{0, 4},
				{7, 4},
				{8, 4},
				{6, 4},
				{0, 5},
				{9, 5},
				{10, 5},
			},
		},
	}

	t.Run("Test Spimi Parse OSM Nodes", func(t *testing.T) {
		for _, c := range cases {
			spimi, err := NewDynamicIndex("test", 500, false, nil, NewIndexedData([]geo.OSMWay{}, []geo.OSMNode{}, geo.NodeMapContainer{},
				nil, geo.OSMSpatialIndex{}, []geo.Boundary{}), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			results := spimi.SpimiParseOSMNodes(c.inputNodes, &sync.RWMutex{}, c.field, c.lenDF, context.Background())
			assert.Equal(t, c.expectedPairs, results)
			assert.Equal(t, 5, len(c.lenDF))
			assert.Equal(t, 3, c.lenDF[1])
			assert.Equal(t, 3, c.lenDF[2])
			assert.Equal(t, 3, c.lenDF[3])
			assert.Equal(t, 4, c.lenDF[4])
			assert.Equal(t, 3, c.lenDF[5])
		}
	})
}

func TestSpimiInvert(t *testing.T) {
	cases := []struct {
		inputNodes       []datastructure.Node
		field            string
		lenDF            map[int]int
		expectedPairs    [][]int
		expectedTermIDs  []int
		expectedPostings map[int][]int
		expectedTermSize int
	}{
		{
			inputNodes: []datastructure.Node{
				{ID: 1, Name: "Jalan Sentosa Harapan"},
				{ID: 2, Name: "Jalan Dunia Baru"},
				{ID: 3, Name: "Jalan Mulwo Apel"},
				{ID: 4, Name: "Jalan Kebun Jeruk Apel"},
				{ID: 5, Name: "Jalan Pantai Ancol"},
			},
			field: "name",

			expectedTermIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expectedPostings: map[int][]int{
				0:  {1, 2, 3, 4, 5},
				1:  {1},
				2:  {1},
				3:  {2},
				4:  {2},
				5:  {3},
				6:  {3, 4},
				7:  {4},
				8:  {4},
				9:  {5},
				10: {5},
			},
			expectedTermSize: 11,
		}}

	t.Run("Test Spimi Invert", func(t *testing.T) {
		for _, c := range cases {
			spimi, err := NewDynamicIndex("test", 500, false, nil, NewIndexedData([]geo.OSMWay{}, []geo.OSMNode{}, geo.NodeMapContainer{},
				nil, geo.OSMSpatialIndex{}, []geo.Boundary{}), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			block := 0
			errResults := spimi.SpimiInvert(c.inputNodes, &block, &sync.RWMutex{}, c.field, context.TODO())
			assert.Nil(t, errResults)

			indexID := spimi.intermediateIndices[0]
			index := NewInvertedIndex(indexID, spimi.outputDir, spimi.workingDir)
			err = index.OpenReader()
			if err != nil {
				t.Error(err)
			}
			defer index.Close()
			indexIterator := NewInvertedIndexIterator(index).IterateInvertedIndex()
			idx := 0
			for item, err := range indexIterator {
				termID := item.GetTermID()
				termSize := item.GetTermSize()
				postingList := item.GetPostingList()

				assert.Equal(t, c.expectedTermIDs[idx], termID)
				idx++
				assert.Equal(t, c.expectedTermSize, termSize)
				assert.Equal(t, c.expectedPostings[termID], postingList)
				assert.Nil(t, err)

			}
		}
	})
}

func TestSpimiMerge(t *testing.T) {
	cases := []struct {
		inputNodesIndexOne []datastructure.Node
		inputNodesIndexTwo []datastructure.Node

		field            string
		lenDF            map[int]int
		expectedPairs    [][]int
		expectedTermIDs  []int
		expectedPostings map[int][]int
		expectedTermSize int
	}{
		{
			inputNodesIndexOne: []datastructure.Node{
				{ID: 1, Name: "Jalan Sentosa Harapan"},
				{ID: 2, Name: "Jalan Dunia Baru"},
				{ID: 3, Name: "Jalan Mulwo Apel"},
				{ID: 4, Name: "Jalan Kebun Jeruk Apel"},
				{ID: 5, Name: "Jalan Pantai Ancol"},
			},
			inputNodesIndexTwo: []datastructure.Node{
				{ID: 6, Name: "Jalan Gambir"},
				{ID: 7, Name: "Jalan Pasar Minggu"},
				{ID: 8, Name: "Jalan Adi Sucipto"},
				{ID: 9, Name: "Jalan Ahmad Yani"},
				{ID: 10, Name: "Jalan Dani"},
				{ID: 11, Name: "Jalan Dani Jadul"},
			},
			field: "name",

			expectedTermIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			expectedPostings: map[int][]int{
				0:  {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
				1:  {1},
				2:  {1},
				3:  {2},
				4:  {2},
				5:  {3},
				6:  {3, 4},
				7:  {4},
				8:  {4},
				9:  {5},
				10: {5},
				11: {6},
				12: {7},
				13: {7},
				14: {8},
				15: {8},
				16: {9},
				17: {9},
				18: {10, 11},
				19: {11},
			},
			expectedTermSize: 20,
		}}

	t.Run("Test Spimi Merge", func(t *testing.T) {
		for _, c := range cases {
			spimi, err := NewDynamicIndex("test", 500, false, nil, NewIndexedData([]geo.OSMWay{}, []geo.OSMNode{}, geo.NodeMapContainer{},
				nil, geo.OSMSpatialIndex{}, []geo.Boundary{}), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			block := 0
			errResults := spimi.SpimiInvert(c.inputNodesIndexOne, &block, &sync.RWMutex{}, c.field, context.TODO())
			assert.Nil(t, errResults)

			errResults = spimi.SpimiInvert(c.inputNodesIndexTwo, &block, &sync.RWMutex{}, c.field, context.TODO())
			assert.Nil(t, errResults)

			indexIDOne := spimi.intermediateIndices[0]

			indexIDTwo := spimi.intermediateIndices[1]

			indexOne := NewInvertedIndex(indexIDOne, spimi.outputDir, spimi.workingDir)
			err = indexOne.OpenReader()
			if err != nil {
				t.Error(err)
			}
			defer indexOne.Close()

			indexTwo := NewInvertedIndex(indexIDTwo, spimi.outputDir, spimi.workingDir)
			err = indexTwo.OpenReader()
			if err != nil {
				t.Error(err)
			}

			mergedIndex := NewInvertedIndex("merged_name_index", spimi.outputDir, spimi.workingDir)

			err = mergedIndex.OpenWriter()
			if err != nil {
				t.Error(err)
			}
			defer mergedIndex.Close()

			err = spimi.Merge([]*InvertedIndex{indexOne, indexTwo}, mergedIndex)
			if err != nil {
				t.Error(err)
			}

			indexIterator := NewInvertedIndexIterator(mergedIndex).IterateInvertedIndex()
			idx := 0
			for item, err := range indexIterator {
				termID := item.GetTermID()
				termSize := item.GetTermSize()
				postingList := item.GetPostingList()

				assert.Equal(t, c.expectedTermIDs[idx], termID)
				idx++
				assert.Equal(t, c.expectedTermSize, termSize)
				assert.Equal(t, c.expectedPostings[termID], postingList)
				assert.Nil(t, err)
			}
		}
	})
}

func TestMergeFieldLength(t *testing.T) {
	cases := []struct {
		inputNodesIndexOne []datastructure.Node
		inputNodesIndexTwo []datastructure.Node

		field            string
		expectedMapLenDF map[int]int
		expectedDocSize  int
	}{
		{
			inputNodesIndexOne: []datastructure.Node{
				{ID: 1, Name: "Jalan Sentosa Harapan"},
				{ID: 2, Name: "Jalan Dunia Baru"},
				{ID: 3, Name: "Jalan Mulwo Apel"},
				{ID: 4, Name: "Jalan Kebun Jeruk Apel"},
				{ID: 5, Name: "Jalan Pantai Ancol"},
			},
			inputNodesIndexTwo: []datastructure.Node{
				{ID: 6, Name: "Jalan Gambir"},
				{ID: 7, Name: "Jalan Pasar Minggu"},
				{ID: 8, Name: "Jalan Adi Sucipto"},
				{ID: 9, Name: "Jalan Ahmad Yani"},
				{ID: 10, Name: "Jalan Dani"},
				{ID: 11, Name: "Jalan Dani Jadul"},
			},
			field: "name",

			expectedMapLenDF: map[int]int{
				1:  3,
				2:  3,
				3:  3,
				4:  4,
				5:  3,
				6:  2,
				7:  3,
				8:  3,
				9:  3,
				10: 2,
				11: 3,
			},
			expectedDocSize: 11,
		}}

	t.Run("Test Spimi Merge", func(t *testing.T) {
		for _, c := range cases {
			spimi, err := NewDynamicIndex("test", 500, false, nil, NewIndexedData([]geo.OSMWay{}, []geo.OSMNode{}, geo.NodeMapContainer{},
				nil, geo.OSMSpatialIndex{}, []geo.Boundary{}), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			block := 0
			errResults := spimi.SpimiInvert(c.inputNodesIndexOne, &block, &sync.RWMutex{}, c.field, context.TODO())
			assert.Nil(t, errResults)

			errResults = spimi.SpimiInvert(c.inputNodesIndexTwo, &block, &sync.RWMutex{}, c.field, context.TODO())
			assert.Nil(t, errResults)

			indexIDOne := spimi.intermediateIndices[0]

			indexIDTwo := spimi.intermediateIndices[1]

			indexOne := NewInvertedIndex(indexIDOne, spimi.outputDir, spimi.workingDir)
			err = indexOne.OpenReader()
			if err != nil {
				t.Error(err)
			}
			defer indexOne.Close()

			indexTwo := NewInvertedIndex(indexIDTwo, spimi.outputDir, spimi.workingDir)
			err = indexTwo.OpenReader()
			if err != nil {
				t.Error(err)
			}

			results := spimi.MergeFieldLengths([]*InvertedIndex{indexOne, indexTwo})
			assert.Equal(t, c.expectedDocSize, len(results))
			assert.Equal(t, c.expectedMapLenDF, results)
		}
	})
}

func TestSpimiBatchIndex(t *testing.T) {
	cases := []struct {
		inputWays          []geo.OSMWay
		inputNodes         []geo.OSMNode
		expectedOsmObjects []datastructure.Node

		nodeMap map[int64]*osm.Node

		expectedPostings map[string][]int
		expectedTermIDs  []int
		expectedTermSize int
	}{
		{
			inputWays: []geo.OSMWay{
				{ID: 1, TagMap: map[string]string{"addr:street": "Jalan Sentosa Harapan",
					"name": "Jalan Sentosa Harapan"},
					NodeIDs: []int64{1}},
				{ID: 2, TagMap: map[string]string{"addr:street": "Jalan Dunia Baru",
					"name": "Jalan Dunia Baru",
				},
					NodeIDs: []int64{2}},
				{ID: 3, TagMap: map[string]string{"addr:street": "Jalan Mulwo Apel",
					"name": "Jalan Mulwo Apel",
				},
					NodeIDs: []int64{3}},
				{ID: 4, TagMap: map[string]string{"addr:street": "Jalan Kebun Jeruk Apel",
					"name": "Jalan Kebun Jeruk Apel",
				},
					NodeIDs: []int64{4}},
				{ID: 5, TagMap: map[string]string{"addr:street": "Jalan Pantai Ancol",
					"name": "Jalan Pantai Ancol"},
					NodeIDs: []int64{5}},
			},

			inputNodes: []geo.OSMNode{
				{ID: 6, TagMap: map[string]string{"addr:street": "Jalan Gambir",
					"name": "Jalan Gambir",
				},
					Lat: 1.0, Lon: 1.0,
				},
				{ID: 7, TagMap: map[string]string{"addr:street": "Jalan Pasar Minggu",
					"name": "Jalan Pasar Minggu",
				},
					Lat: 3.0, Lon: 3.0},
				{ID: 8, TagMap: map[string]string{"addr:street": "Jalan Adi Sucipto",
					"name": "Jalan Adi Sucipto",
				},
					Lat: 4.0, Lon: 4.0},
				{ID: 9, TagMap: map[string]string{"addr:street": "Jalan Ahmad Yani",
					"name": "Jalan Ahmad Yani",
				},
					Lat: 5.0, Lon: 5.0},
				{ID: 10, TagMap: map[string]string{"addr:street": "Jalan Dani",
					"name": "Jalan Dani",
				}, Lat: 6.0, Lon: 6.0},
				{ID: 11, TagMap: map[string]string{"addr:street": "Jalan Dani Jadul",
					"name": "Jalan Dani Jadul",
				}, Lat: 6.0, Lon: 6.0,
				},
			},

			nodeMap: map[int64]*osm.Node{
				1: {
					ID: 1, Lat: 1.0, Lon: 1.0,
				},
				2: {
					ID: 2, Lat: 2.0, Lon: 2.0,
				},
				3: {
					ID: 3, Lat: 3.0, Lon: 3.0,
				},
				4: {
					ID: 4, Lat: 4.0, Lon: 4.0,
				},
				5: {
					ID: 5, Lat: 5.0, Lon: 5.0,
				},
			},

			expectedOsmObjects: []datastructure.Node{
				{ID: 0, Name: "Jalan Sentosa Harapan",
					Lat: 1.0, Lon: 1.0, Address: "Jalan Sentosa Harapan",
					Tipe: "",
				},
				{ID: 1, Name: "Jalan Dunia Baru",
					Lat: 2.0, Lon: 2.0, Address: "Jalan Dunia Baru",
					Tipe: "",
				},
				{ID: 2, Name: "Jalan Mulwo Apel",
					Lat: 3.0, Lon: 3.0, Address: "Jalan Mulwo Apel",
					Tipe: "",
				},
				{ID: 3, Name: "Jalan Kebun Jeruk Apel",
					Lat: 4.0, Lon: 4.0, Address: "Jalan Kebun Jeruk Apel",
					Tipe: "",
				},
				{ID: 4, Name: "Jalan Pantai Ancol",
					Lat: 5.0, Lon: 5.0, Address: "Jalan Pantai Ancol",
					Tipe: "",
				},
				{ID: 5, Name: "Jalan Gambir",
					Lat: 1.0, Lon: 1.0, Address: "Jalan Gambir",
					Tipe: "",
				},
				{ID: 6, Name: "Jalan Pasar Minggu",
					Lat: 3.0, Lon: 3.0, Address: "Jalan Pasar Minggu",
					Tipe: "",
				},
				{ID: 7, Name: "Jalan Adi Sucipto",
					Lat: 4.0, Lon: 4.0, Address: "Jalan Adi Sucipto",
					Tipe: "",
				},
				{ID: 8, Name: "Jalan Ahmad Yani",
					Lat: 5.0, Lon: 5.0, Address: "Jalan Ahmad Yani",
					Tipe: "",
				},
				{ID: 9, Name: "Jalan Dani",
					Lat: 6.0, Lon: 6.0, Address: "Jalan Dani",
					Tipe: "",
				},
				{ID: 10, Name: "Jalan Dani Jadul",
					Lat: 6.0, Lon: 6.0, Address: "Jalan Dani Jadul",
					Tipe: "",
				},
			},

			expectedTermIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			expectedPostings: map[string][]int{
				"jalan":   {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				"sentosa": {0},
				"harapan": {0},
				"dunia":   {1},
				"baru":    {1},
				"mulwo":   {2},
				"apel":    {2, 3},
				"kebun":   {3},
				"jeruk":   {3},
				"pantai":  {4},
				"ancol":   {4},
				"gambir":  {5},
				"pasar":   {6},
				"minggu":  {6},
				"adi":     {7},
				"sucipto": {7},
				"ahmad":   {8},
				"yani":    {8},
				"dani":    {9, 10},
				"jadul":   {10},
			},
			expectedTermSize: 20,
		},
	}

	t.Run("Test Spimi Batch Index", func(t *testing.T) {

		for _, c := range cases {
			db, err := bolt.Open("docs_store.db", 0600, nil)
			if err != nil {
				t.Error(err)
			}
			err = db.Update(func(tx *bolt.Tx) error {
				_, err := tx.CreateBucketIfNotExists([]byte(kvdb.BBOLTDB_BUCKET))
				return err
			})
			if err != nil {
				t.Error(err)
			}

			bboltKV := kvdb.NewKVDB(db)
			defer db.Close()

			spatialIndex := geo.OSMSpatialIndex{
				StreetRtree:                 datastructure.NewRtree(25, 50, 2),
				AdministrativeBoundaryRtree: datastructure.NewRtree(25, 50, 2),
			}

			spimi, err := NewDynamicIndex("test", 500, false, nil, NewIndexedData([]geo.OSMWay{}, []geo.OSMNode{}, geo.NodeMapContainer{},
				nil, spatialIndex, []geo.Boundary{}), bboltKV)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			spimi.IndexedData.Ctr = geo.NodeMapContainer{}
			spimi.IndexedData.Ctr.SetNodeMap(c.nodeMap)
			spimi.IndexedData.Ways = c.inputWays
			spimi.IndexedData.Nodes = c.inputNodes

			nodes, errResults := spimi.SpimiBatchIndex(context.Background())
			assert.Nil(t, errResults)

			docIDOriginalMap := make(map[int]int, len(c.expectedOsmObjects))
			for _, node := range nodes {
				isThere := false

				for _, expectedNode := range c.expectedOsmObjects {
					if node.Name == expectedNode.Name {
						isThere = true
						assert.Equal(t, expectedNode.Name, node.Name)
						assert.Equal(t, expectedNode.Lat, node.Lat)
						assert.Equal(t, expectedNode.Lon, node.Lon)
						assert.Equal(t, expectedNode.Address, node.Address)
						assert.Equal(t, expectedNode.Tipe, node.Tipe)
						docIDOriginalMap[expectedNode.ID] = node.ID
					}
				}

				assert.True(t, true, isThere)
			}

			// test posting list merged index
			mergedIndex := NewInvertedIndex("merged_name_index", spimi.outputDir, spimi.workingDir)

			err = mergedIndex.OpenReader()
			if err != nil {
				t.Error(err)
			}
			defer mergedIndex.Close()

			indexIterator := NewInvertedIndexIterator(mergedIndex).IterateInvertedIndex()
			idx := 0
			for item, err := range indexIterator {
				termID := item.GetTermID()
				termSize := item.GetTermSize()
				postingList := item.GetPostingList()

				assert.Equal(t, c.expectedTermIDs[idx], termID)
				idx++
				assert.Equal(t, c.expectedTermSize, termSize)

				termIDOriginal := spimi.TermIDMap.GetStr(termID)
				for i, posting := range c.expectedPostings[termIDOriginal] {
					c.expectedPostings[termIDOriginal][i] = docIDOriginalMap[posting]
				}
				sort.Ints(c.expectedPostings[termIDOriginal])
				assert.Equal(t, c.expectedPostings[termIDOriginal], postingList)
				assert.Nil(t, err)
			}
		}
	})
}

// func TestGetFullAdress(t *testing.T) {
// 	streetRtree := datastructure.NewRtree(25, 50, 2)
// 	kelurahanRtree := datastructure.NewRtree(25, 50, 2)
// 	kecamatanRtree := datastructure.NewRtree(25, 50, 2)
// 	kotaKabupatenRtree := datastructure.NewRtree(25, 50, 2)
// 	provinsiRtree := datastructure.NewRtree(25, 50, 2)
// 	countryRtree := datastructure.NewRtree(25, 50, 2)

// 	type osmBoundary struct {
// 		place    datastructure.OSMObject
// 		boundary datastructure.RtreeBoundingBox
// 	}

// 	osmRelations := []geo.OsmRelation{
// 		{
// 			Name:        "Kelurahan Mangga Besar",
// 			AdminLevel:  "7",
// 			PostalCode:  "11180",
// 			BoundaryLat: []float64{-6.150087792302976, -6.149334703270099, -6.139710940139253, -6.142485523543515},
// 			BoundaryLon: []float64{106.81683956211604, 106.82018827886075, 106.82100794970584, 106.81445004620737},
// 		},
// 		{
// 			Name:        "Kelurahan Petojo Utara",
// 			AdminLevel:  "7",
// 			PostalCode:  "10130",
// 			BoundaryLat: []float64{-6.170986510265271, -6.16743409672024, -6.159339043240257, -6.1606650624505805},
// 			BoundaryLon: []float64{106.81099946273496, 106.8203044150482, 106.81880894021228, 106.81041761905553},
// 		},
// 		{
// 			Name:        "Kelurahan Menteng Atas",
// 			AdminLevel:  "7",
// 			PostalCode:  "12960",
// 			BoundaryLat: []float64{-6.222528201611587, -6.2204967095184145, -6.213553083230103, -6.209623703723849, -6.210549031146272},
// 			BoundaryLon: []float64{106.83550065309862, 106.84590286191634, 106.84254587229012, 106.83279977337527, 106.83021793477158},
// 		},
// 		{
// 			Name:        "Kecamatan Gambir",
// 			AdminLevel:  "6",
// 			PostalCode:  "banyak",
// 			BoundaryLat: []float64{-6.184521644571109, -6.180121967780558, -6.157995294320933, -6.16053006020675},
// 			BoundaryLon: []float64{106.81027784986242, 106.8374185382924, 106.82758144473829, 106.79763805856818},
// 		},
// 		{
// 			Name:        "Kecamatan Setiabudi",
// 			AdminLevel:  "6",
// 			PostalCode:  "banyak",
// 			BoundaryLat: []float64{-6.21866534054827, -6.2406003849573795, -6.208119320220383, -6.202297826509481},
// 			BoundaryLon: []float64{106.81249248529357, 106.83430322573325, 106.84796676334275, 106.8215732212394},
// 		},
// 		{
// 			Name:        "Kecamatan Tanah Abang",
// 			AdminLevel:  "6",
// 			PostalCode:  "banyak",
// 			BoundaryLat: []float64{-6.2291611315253475, -6.212006282753908, -6.182945339244465, -6.180759112449729, -6.20747829652909},
// 			BoundaryLon: []float64{106.795713737478, 106.82104309729623, 106.82173105536356, 106.81352886327147, 106.79066270506156},
// 		},
// 		{
// 			Name:        "Jakarta Selatan",
// 			AdminLevel:  "5",
// 			PostalCode:  "banyak",
// 			BoundaryLat: []float64{-6.363344071653244, -6.305727317833632, -6.2024960565859795, -6.221846573279071},
// 			BoundaryLon: []float64{106.79354954403723, 106.8554323623846, 106.8518758794823, 106.73060566764467},
// 		},
// 		{Name: "Jakarta Pusat",
// 			AdminLevel: "5",
// 			PostalCode: "banyak",
// 			BoundaryLat: []float64{-6.227359764809661, -6.201079247125575, -6.204833687115288, -6.161144007079768, -6.134518843105471, -6.159437305800751,
// 				-6.158754623752873, -6.209270718072721},
// 			BoundaryLon: []float64{106.7966347038682, 106.82581713608188, 106.84710314546128, 106.88109209592191, 106.82238390876262, 106.82513049061804,
// 				106.79766467206396, 106.79251483108509},
// 		},
// 		{
// 			Name:       "Jakarta Barat",
// 			AdminLevel: "5",
// 			PostalCode: "banyak",
// 			BoundaryLat: []float64{-6.22123850189862, -6.220835199156882, -6.1843563008187274, -6.157762619800127,
// 				-6.158785478347846, -6.130334859909201, -6.096173916135376},
// 			BoundaryLon: []float64{106.71877123078804, 106.78187568695678, 106.80931022877958, 106.79696468495932,
// 				106.82645681741882, 106.81470653560596, 106.68957182762774},
// 		},
// 		{
// 			Name:        "DKI Jakarta",
// 			AdminLevel:  "4",
// 			PostalCode:  "banyak",
// 			BoundaryLat: []float64{-6.337003118166669, -6.36878821684634, -6.090705089605275, -6.101613067086238},
// 			BoundaryLon: []float64{106.75911197706, 106.91269214098934, 106.97576970831747, 106.6768368892407},
// 		},
// 	}
// 	streets := []osmBoundary{
// 		{place: datastructure.OSMObject{ID: 100, Lat: -6.198026959830097, Lon: 106.83693911690615}}, // Jalan Teuku Cik di Tiro
// 		{place: datastructure.OSMObject{ID: 102, Lat: -6.165490037050512, Lon: 106.81519795637621}}, // Jl. KH.  Hasyim Ashari , Petojo Utara
// 		{place: datastructure.OSMObject{ID: 103, Lat: -6.217789642359141, Lon: 106.83946785381423}}, //  Jalan Mentas Selatan III , Menteng Atas
// 	}
// 	kelurahans := []osmBoundary{
// 		{place: datastructure.OSMObject{ID: 0, Lat: -6.144336098110183, Lon: 106.81768539406508}},
// 		{place: datastructure.OSMObject{ID: 1, Lat: -6.164757271285033, Lon: 106.8148880181382}},
// 		{place: datastructure.OSMObject{ID: 2, Lat: -6.216727645737386, Lon: 106.83972388707002}},
// 	}
// 	kecamatans := []osmBoundary{
// 		{place: datastructure.OSMObject{ID: 3, Lat: -6.169973964189839, Lon: 106.81693941209289}},
// 		{place: datastructure.OSMObject{ID: 4, Lat: -6.19346174518342, Lon: 106.83112464014177}},
// 		{place: datastructure.OSMObject{ID: 5, Lat: -6.171111, Lon: 106.815278}},
// 	}
// 	kotaKabupatens := []osmBoundary{
// 		{place: datastructure.OSMObject{ID: 6, Lat: -6.265667490682931, Lon: 106.81048366226537}},
// 		{place: datastructure.OSMObject{ID: 7, Lat: -6.1775280599143025, Lon: 106.82822039520536}},
// 		{place: datastructure.OSMObject{ID: 8, Lat: -6.2088, Lon: 106.8456}},
// 	}

// 	provinsis := []osmBoundary{
// 		{place: datastructure.OSMObject{ID: 9, Lat: -6.1915953751289505, Lon: 106.83864456195198}},
// 	}

// 	cases := []struct {
// 		inputStreet      string
// 		inputPostalCode  string
// 		inputHouseNumber string
// 		inputCenterLat   float64
// 		inputCenterLon   float64

// 		expectedFullAddress string
// 		expectedCity        string
// 	}{
// 		//(1) mentas selatan III
// 		// (2) KH. Hasyim Ashari
// 		// (3) Teuku A.M Sangaji
// 		{

// 			inputStreet:      "",
// 			inputPostalCode:  "",
// 			inputHouseNumber: "1010",

// 			inputCenterLat:      -6.217665707038372,
// 			inputCenterLon:      106.83936892434181,
// 			expectedFullAddress: "Jalan Mentas Selatan III, 1010, Kelurahan Menteng Atas, Kecamatan Setiabudi, Jakarta Selatan, DKI Jakarta, 12960",
// 			expectedCity:        "Jakarta Selatan",
// 		}, {
// 			inputStreet:         "",
// 			inputPostalCode:     "",
// 			inputHouseNumber:    "2301",
// 			inputCenterLat:      -6.1654085595503885,
// 			inputCenterLon:      106.81374553119,
// 			expectedFullAddress: "Jalan KH. Hasyim Ashari, 2301, Kelurahan Petojo Utara, Kecamatan Gambir, Jakarta Pusat, DKI Jakarta, 10130",
// 			expectedCity:        "Jakarta Pusat",
// 		},
// 		{
// 			inputStreet:         "Jalan A.M Sangaji",
// 			inputPostalCode:     "",
// 			inputHouseNumber:    "2034",
// 			inputCenterLat:      -6.166592591581767,
// 			inputCenterLon:      106.81378118987627,
// 			expectedFullAddress: "Jalan A.M Sangaji, 2034, Kelurahan Petojo Utara, Kecamatan Gambir, Jakarta Pusat, DKI Jakarta, 10130",
// 			expectedCity:        "Jakarta Pusat",
// 		},
// 	}

// 	t.Run("Test Get Full Address", func(t *testing.T) {
// 		for _, c := range cases {
// 			for _, street := range streets {
// 				bound := datastructure.NewRtreeBoundingBox(2, []float64{street.place.Lat - 0.0001, street.place.Lon - 0.0001}, []float64{street.place.Lat + 0.0001, street.place.Lon + 0.0001})

// 				streetRtree.InsertLeaf(bound, street.place, false)
// 			}

// 			for _, kelurahan := range kelurahans {
// 				boundaryLat := make([]float64, len(osmRelations[kelurahan.place.ID].BoundaryLat))
// 				boundaryLon := make([]float64, len(osmRelations[kelurahan.place.ID].BoundaryLon))
// 				copy(boundaryLat, osmRelations[kelurahan.place.ID].BoundaryLat)
// 				copy(boundaryLon, osmRelations[kelurahan.place.ID].BoundaryLon)
// 				sort.Float64s(boundaryLat)
// 				sort.Float64s(boundaryLon)
// 				bound := datastructure.NewRtreeBoundingBox(2, []float64{boundaryLat[0], boundaryLon[0]},
// 					[]float64{boundaryLat[len(boundaryLat)-1], boundaryLon[len(boundaryLon)-1]})
// 				kelurahanRtree.InsertLeaf(bound, kelurahan.place, false)
// 			}

// 			for _, kecamatan := range kecamatans {
// 				boundaryLat := make([]float64, len(osmRelations[kecamatan.place.ID].BoundaryLat))
// 				boundaryLon := make([]float64, len(osmRelations[kecamatan.place.ID].BoundaryLon))
// 				copy(boundaryLat, osmRelations[kecamatan.place.ID].BoundaryLat)
// 				copy(boundaryLon, osmRelations[kecamatan.place.ID].BoundaryLon)
// 				sort.Float64s(boundaryLat)
// 				sort.Float64s(boundaryLon)
// 				bound := datastructure.NewRtreeBoundingBox(2, []float64{boundaryLat[0], boundaryLon[0]},
// 					[]float64{boundaryLat[len(boundaryLat)-1], boundaryLon[len(boundaryLon)-1]})
// 				kecamatanRtree.InsertLeaf(bound, kecamatan.place, false)
// 			}

// 			for _, kotaKabupaten := range kotaKabupatens {
// 				boundaryLat := make([]float64, len(osmRelations[kotaKabupaten.place.ID].BoundaryLat))
// 				boundaryLon := make([]float64, len(osmRelations[kotaKabupaten.place.ID].BoundaryLon))
// 				copy(boundaryLat, osmRelations[kotaKabupaten.place.ID].BoundaryLat)
// 				copy(boundaryLon, osmRelations[kotaKabupaten.place.ID].BoundaryLon)
// 				sort.Float64s(boundaryLat)
// 				sort.Float64s(boundaryLon)
// 				bound := datastructure.NewRtreeBoundingBox(2, []float64{boundaryLat[0], boundaryLon[0]},
// 					[]float64{boundaryLat[len(boundaryLat)-1], boundaryLon[len(boundaryLon)-1]})
// 				kotaKabupatenRtree.InsertLeaf(bound, kotaKabupaten.place, false)
// 			}

// 			for _, provinsi := range provinsis {
// 				boundaryLat := make([]float64, len(osmRelations[provinsi.place.ID].BoundaryLat))
// 				boundaryLon := make([]float64, len(osmRelations[provinsi.place.ID].BoundaryLon))
// 				copy(boundaryLat, osmRelations[provinsi.place.ID].BoundaryLat)
// 				copy(boundaryLon, osmRelations[provinsi.place.ID].BoundaryLon)
// 				sort.Float64s(boundaryLat)
// 				sort.Float64s(boundaryLon)
// 				bound := datastructure.NewRtreeBoundingBox(2, []float64{boundaryLat[0], boundaryLon[0]},
// 					[]float64{boundaryLat[len(boundaryLat)-1], boundaryLon[len(boundaryLon)-1]})
// 				provinsiRtree.InsertLeaf(bound, provinsi.place, false)
// 			}

// 			osmSpatialIndex := geo.OSMSpatialIndex{
// 				StreetRtree:        streetRtree,
// 				KelurahanRtree:     kelurahanRtree,
// 				KecamatanRtree:     kecamatanRtree,
// 				KotaKabupatenRtree: kotaKabupatenRtree,
// 				ProvinsiRtree:      provinsiRtree,
// 				CountryRtree:       countryRtree,
// 			}

// 			spimi, err := NewDynamicIndex("test", 500, false, nil, NewIndexedData([]geo.OSMWay{}, []geo.OSMNode{}, geo.NodeMapContainer{}, nil,
// 				osmSpatialIndex, osmRelations), nil)
// 			if err != nil {
// 				t.Errorf("Error creating new dynamic index: %v", err)
// 			}

// 			spimi.IndexedData.Ways = make([]geo.OSMWay, 105)
// 			spimi.IndexedData.Ways[102] = geo.OSMWay{ID: 102, TagMap: map[string]string{"name": "Jalan KH. Hasyim Ashari"}}
// 			spimi.IndexedData.Ways[103] = geo.OSMWay{ID: 103, TagMap: map[string]string{"name": "Jalan Mentas Selatan III"}}
// 			spimi.IndexedData.Ways[100] = geo.OSMWay{ID: 100, TagMap: map[string]string{"name": "Jalan Teuku Cik di Tiro"}}

// 			fullAddress, city := spimi.GetFullAdress(c.inputStreet, c.inputPostalCode, c.inputHouseNumber,
// 				c.inputCenterLat, c.inputCenterLon)

// 			assert.Equal(t, c.expectedFullAddress, fullAddress)
// 			assert.Equal(t, c.expectedCity, city)
// 		}
// 	})

// }
