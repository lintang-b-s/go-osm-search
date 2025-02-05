package index

import (
	"context"
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
				nil), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			results := spimi.SpimiParseOSMNode(c.inputNodes, c.lenDF, &sync.Mutex{}, c.field)
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
				nil), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			results := spimi.SpimiParseOSMNodes(c.inputNodes, &sync.Mutex{}, c.field, c.lenDF, context.Background())
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
				nil), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			block := 0
			errResults := spimi.SpimiInvert(c.inputNodes, &block, &sync.Mutex{}, c.field, context.TODO())
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
				nil), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			block := 0
			errResults := spimi.SpimiInvert(c.inputNodesIndexOne, &block, &sync.Mutex{}, c.field, context.TODO())
			assert.Nil(t, errResults)

			errResults = spimi.SpimiInvert(c.inputNodesIndexTwo, &block, &sync.Mutex{}, c.field, context.TODO())
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
				nil), nil)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			block := 0
			errResults := spimi.SpimiInvert(c.inputNodesIndexOne, &block, &sync.Mutex{}, c.field, context.TODO())
			assert.Nil(t, errResults)

			errResults = spimi.SpimiInvert(c.inputNodesIndexTwo, &block, &sync.Mutex{}, c.field, context.TODO())
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
				"harap":   {0},
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
				"dan":     {9, 10},
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

			spimi, err := NewDynamicIndex("test", 500, false, nil, NewIndexedData([]geo.OSMWay{}, []geo.OSMNode{}, geo.NodeMapContainer{},
				nil), bboltKV)
			if err != nil {
				t.Errorf("Error creating new dynamic index: %v", err)
			}
			spimi.IndexedData.Ctr = geo.NodeMapContainer{}
			spimi.IndexedData.Ctr.SetNodeMap(c.nodeMap)
			spimi.IndexedData.Ways = c.inputWays
			spimi.IndexedData.Nodes = c.inputNodes

			spatialIndex := geo.OSMSpatialIndex{
				StreetRtree:        datastructure.NewRtree(25, 50, 2),
				KelurahanRtree:     datastructure.NewRtree(25, 50, 2),
				KecamatanRtree:     datastructure.NewRtree(25, 50, 2),
				KotaKabupatenRtree: datastructure.NewRtree(25, 50, 2),
				ProvinsiRtree:      datastructure.NewRtree(25, 50, 2),
				CountryRtree:       datastructure.NewRtree(25, 50, 2),
				PostalCodeRtree:    datastructure.NewRtree(25, 50, 2),
			}
			nodes, errResults := spimi.SpimiBatchIndex(context.Background(), spatialIndex, []geo.OsmRelation{})
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
