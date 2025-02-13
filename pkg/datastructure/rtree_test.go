package datastructure

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
)

// this is trash
func traverseRtreeAndTestIfRtreePropertiesCorrect(rt *Rtree, node *RtreeNode, countLeaf *int,
	expectedLeafLevel int, level int, t *testing.T) {
	if node == rt.Root && level != expectedLeafLevel {
		if len(node.Items) < 2 {
			t.Errorf("The root node must has at least two children unless it is a leaf.")
		}
	}
	if node.IsLeaf {
		height := float64(level - 1)
		logmN := math.Log(float64(rt.Size)) / math.Log(float64(rt.MinChildItems))
		assert.LessOrEqual(t, height, math.Ceil(logmN)-1, fmt.Sprintf("The height of an R-tree containing N index records is at most ceil(logmN)-1"))

		if level != expectedLeafLevel {
			t.Errorf("All leaves not appear on the same level")
		}

		if rt.Root != node && (len(node.Items) < rt.MinChildItems || len(node.Items) > rt.MaxChildItems) {
			t.Errorf(" Every leaf node has between m and M children unless it is the root.")
		}

		maxBB := node.Items[0].GetBound()
		for _, item := range node.Items {
			*countLeaf++
			bb := item.GetBound()
			maxBB = boundingBox(maxBB, bb)
		}

		if !node.Bound.isBBSame(maxBB) {
			t.Errorf("For each index record(I, tuple-identifier) in a leaf node, I is the smallest rectangle that spatially contains the n-dimensional data object represented by the indicated tuple.")
		}
	} else {
		maxBB := node.Items[0].GetBound()

		if rt.Root != node && (len(node.Items) < rt.MinChildItems || len(node.Items) > rt.MaxChildItems) {
			t.Errorf(" Every non-leaf node has between m and M children unless it is the root.")
		}

		for _, item := range node.Items {
			bb := item.GetBound()
			maxBB = boundingBox(maxBB, bb)
			traverseRtreeAndTestIfRtreePropertiesCorrect(rt, item, countLeaf, expectedLeafLevel, level+1, t)
		}

		if !node.Bound.isBBSame(maxBB) {
			t.Errorf("(4) For each entry (I, child —pointer ) in a non-leaf node, I is the smallest rectangle that spatially contains the rectangles in tleafhe child node")
		}
	}
}

func TestInsertLeaftree(t *testing.T) {
	itemsData := []OSMObject{}
	for i := 1; i < 100; i++ {
		itemsData = append(itemsData, OSMObject{
			ID:  i,
			Lat: float64(i),
			Lon: float64(i),
		})
	}

	tests := []struct {
		name        string
		items       []OSMObject
		expectItems int
	}{
		{
			name: "Insert 100 item",
			items: append(itemsData, []OSMObject{
				{
					ID:  100,
					Lat: 0,
					Lon: -5,
				},
				{
					ID:  101,
					Lat: 2,
					Lon: -10,
				},
				{
					ID:  102,
					Lat: 3,
					Lon: -15,
				},
			}...),
			expectItems: 102,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := NewRtree(25, 50, 2)
			for _, item := range tt.items {
				bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})

				rt.InsertLeaf(bound, item, false)

			}
			assert.Equal(t, 102, rt.Size)

			countLeaf := 0

			expectedLeafLevel := 2
			traverseRtreeAndTestIfRtreePropertiesCorrect(rt, rt.Root, &countLeaf, expectedLeafLevel, 1, t)
			assert.Equal(t, tt.expectItems, countLeaf)
		})
	}

	t.Run("Insert 5 items", func(t *testing.T) {
		rt := NewRtree(25, 50, 2)
		for i := 0; i < 5; i++ {
			item := itemsData[i]
			bound := NewRtreeBoundingBox(2, []float64{itemsData[i].Lat - 0.0001,
				itemsData[i].Lon - 0.0001}, []float64{itemsData[i].Lat + 0.0001, itemsData[i].Lon + 0.0001})

			rt.InsertLeaf(bound, item, false)
		}
		assert.Equal(t, 5, rt.Size)
		root := rt.Root
		for i, item := range root.Items {
			assert.Equal(t, itemsData[i].ID, item.Leaf.ID)
		}

		countLeaf := 0
		expectedLeafLevel := 1
		traverseRtreeAndTestIfRtreePropertiesCorrect(rt, rt.Root, &countLeaf, expectedLeafLevel, 1, t)
		assert.Equal(t, 5, countLeaf)
	})
}

func TestSearch(t *testing.T) {

	t.Run("Search", func(t *testing.T) {
		itemsData := []OSMObject{}
		for i := 1; i < 100; i++ {
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: float64(i),
				Lon: float64(i),
			})
		}

		rt := NewRtree(10, 25, 2)
		for i, item := range itemsData {
			bound := NewRtreeBoundingBox(2, []float64{itemsData[i].Lat - 0.0001, itemsData[i].Lon - 0.0001},
				[]float64{itemsData[i].Lat + 0.0001, itemsData[i].Lon + 0.0001})

			rt.InsertLeaf(bound, item, false)

		}

		countLeaf := 0

		expectedLeafLevel := 2
		traverseRtreeAndTestIfRtreePropertiesCorrect(rt, rt.Root, &countLeaf, expectedLeafLevel, 1, t)

		results := rt.Search(NewRtreeBoundingBox(2, []float64{0, 0}, []float64{50, 50}))

		for _, item := range results {

			itembb := item.GetBound()
			if !overlaps(itembb, NewRtreeBoundingBox(2, []float64{0, 0}, []float64{50, 50})) {
				t.Errorf("Bounding box is not correct")
			}
		}
	})
}

func TestSplit(t *testing.T) {
	t.Run("Split", func(t *testing.T) {
		itemsData := []OSMObject{}
		for i := 1; i < 27; i++ {
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: float64(i),
				Lon: float64(i),
			})
		}

		rt := NewRtree(10, 25, 2)

		bound := NewRtreeBoundingBox(2, []float64{itemsData[0].Lat - 0.0001, itemsData[0].Lon - 0.0001}, []float64{itemsData[0].Lat + 0.0001, itemsData[0].Lon + 0.0001})

		rt.InsertLeaf(bound, itemsData[0], false)
		for i := 1; i < 26; i++ {
			item := itemsData[i]
			bound := NewRtreeBoundingBox(2, []float64{itemsData[i].Lat - 0.0001, itemsData[i].Lon - 0.0001}, []float64{itemsData[i].Lat + 0.0001, itemsData[i].Lon + 0.0001})

			newLeaf := &RtreeNode{Leaf: item, Bound: bound}
			rt.Root.Items = append(rt.Root.Items, newLeaf)
		}

		old, newNode := rt.splitNode(rt.Root)

		assert.Less(t, len(newNode.Items), 25)
		assert.Less(t, len(rt.Root.Items), 25)
		assert.Less(t, len(old.Items), 25)

	})
}

func randomLatLon(minLat, maxLat, minLon, maxLon float64) (float64, float64) {
	rand.Seed(uint64(time.Now().UnixNano()))
	lat := minLat + rand.Float64()*(maxLat-minLat)
	lon := minLon + rand.Float64()*(maxLon-minLon)
	return lat, lon
}

func TestNNearestNeighborsPQ(t *testing.T) {
	t.Run("Test N Nearest Neighbors kota surakarta", func(t *testing.T) {
		itemsData := []OSMObject{
			{
				ID:  7,
				Lat: -7.546392935195944,
				Lon: 110.77718220472673,
			},
			{
				ID:  6,
				Lat: -7.5559986670115675,
				Lon: 110.79466621171177,
			},
			{
				ID:  5,
				Lat: -7.555869730414206,
				Lon: 110.80500875243253,
			},
			{
				ID:  4,
				Lat: -7.571289544570394,
				Lon: 110.8301500772816,
			},
			{
				ID:  3,
				Lat: -7.7886707815273155,
				Lon: 110.361625035987,
			}, {
				ID:  2,
				Lat: -7.8082872068169475,
				Lon: 110.35793427899466,
			},
			{
				ID:  1,
				Lat: -7.759889166547908,
				Lon: 110.36689459108496,
			},
		}

		for i := 8; i < 100001; i++ {
			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: lat,
				Lon: lon,
			})
		}

		rt := NewRtree(25, 50, 2)
		for _, item := range itemsData {
			bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})

			rt.InsertLeaf(bound, item, false)
		}

		countLeaf := 0
		expectedLeafLevel := 4
		traverseRtreeAndTestIfRtreePropertiesCorrect(rt, rt.Root, &countLeaf, expectedLeafLevel, 1, t)

		assert.Equal(t, 100000, countLeaf)

		myLocation := Point{-7.548263971398246, 110.78226484631368}
		results := rt.NearestNeighboursPQ(5, myLocation)

		assert.Equal(t, 5, len(results))

		assert.Equal(t, 7, results[0].ID)
		assert.Equal(t, 6, results[1].ID)
		assert.Equal(t, 5, results[2].ID)
		assert.Equal(t, 4, results[3].ID)
		assert.Equal(t, 1, results[4].ID)
	})
}

func TestNearestNeighbor(t *testing.T) {
	t.Run("Test N Nearest Neighbors kota surakarta", func(t *testing.T) {
		itemsData := []OSMObject{
			{
				ID:  7,
				Lat: -7.546392935195944,
				Lon: 110.77718220472673,
			},
			{
				ID:  6,
				Lat: -7.5559986670115675,
				Lon: 110.79466621171177,
			},
			{
				ID:  5,
				Lat: -7.555869730414206,
				Lon: 110.80500875243253,
			},
			{
				ID:  4,
				Lat: -7.571289544570394,
				Lon: 110.8301500772816,
			},
			{
				ID:  3,
				Lat: -7.7886707815273155,
				Lon: 110.361625035987,
			}, {
				ID:  2,
				Lat: -7.8082872068169475,
				Lon: 110.35793427899466,
			},
			{
				ID:  1,
				Lat: -7.759889166547908,
				Lon: 110.36689459108496,
			},
			{
				ID:  1000,
				Lat: -7.550561079106621,
				Lon: 110.7837156929654,
			},
			{
				ID:  1001,
				Lat: -7.700002453207869,
				Lon: 110.37712514761436,
			},
		}

		for i := 8; i < 100000; i++ {
			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: lat,
				Lon: lon,
			})
		}

		rt := NewRtree(25, 50, 2)
		for _, item := range itemsData {
			bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})
			rt.InsertLeaf(bound, item, false)
		}

		countLeaf := 0
		expectedLeafLevel := 4
		traverseRtreeAndTestIfRtreePropertiesCorrect(rt, rt.Root, &countLeaf, expectedLeafLevel, 1, t)

		myLocation := Point{-7.760335932763678, 110.37671195413539}

		result := rt.ImprovedNearestNeighbor(myLocation)
		assert.Equal(t, 1, result.ID)
	})
}

func TestNearestNeighborRadiusFilterOsmFeature(t *testing.T) {
	t.Run("Test N Nearest Neighbors dengan tag tertentu dan radius 3km", func(t *testing.T) {
		itemsData := []OSMObject{
			{
				ID:  7,
				Lat: -7.546392935195944,
				Lon: 110.77718220472673,
				Tag: map[int]int{1: 1},
			},
			{
				ID:  6,
				Lat: -7.5559986670115675,
				Lon: 110.79466621171177,
				Tag: map[int]int{1: 1},
			},
			{
				ID:  5,
				Lat: -7.555869730414206,
				Lon: 110.80500875243253,
				Tag: map[int]int{1: 1},
			},
			{
				ID:  4,
				Lat: -7.571289544570394,
				Lon: 110.8301500772816,
				Tag: map[int]int{1: 1},
			},
			{
				ID:  3,
				Lat: -7.7886707815273155,
				Lon: 110.361625035987,
				Tag: map[int]int{10: 10},
			}, {
				ID:  2,
				Lat: -7.8082872068169475,
				Lon: 110.35793427899466,
				Tag: map[int]int{10: 10},
			},
			{
				ID:  1,
				Lat: -7.759889166547908,
				Lon: 110.36689459108496,
				Tag: map[int]int{1: 1},
			},
			{
				ID:  1000,
				Lat: -7.550561079106621,
				Lon: 110.7837156929654,
				Tag: map[int]int{10: 10},
			},
			{
				ID:  1001,
				Lat: -7.700002453207869,
				Lon: 110.37712514761436,
				Tag: map[int]int{1: 1},
			},
			{
				ID:  1002,
				Lat: -7.760860864556355,
				Lon: 110.37510209125597,
				Tag: map[int]int{1: 1},
			},
			{
				ID:  1003,
				Lat: -7.759614617476093,
				Lon: 110.37787347463819,
				Tag: map[int]int{3: 2},
			},
			{
				ID:  1003,
				Lat: -7.761846768918608,
				Lon: 110.38114368428886,
				Tag: map[int]int{3: 2},
			},
		}

		for i := 8; i < 100000; i++ {
			tag := rand.Intn(10)
			val := rand.Intn(10)

			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: lat,
				Lon: lon,
				Tag: map[int]int{tag: val},
			})
		}

		rt := NewRtree(25, 50, 2)
		for _, item := range itemsData {
			bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})

			rt.InsertLeaf(bound, item, false)
		}

		myLocation := Point{-7.760335932763678, 110.37671195413539}

		results := rt.NearestNeighboursRadiusFilterOSM(5, 0, myLocation, 3.0, 1)
		for _, item := range results {
			if _, ok := item.Tag[1]; HaversineDistance(myLocation.Lat, myLocation.Lon, item.Lat, item.Lon) > 3.0 ||
				!ok {
				t.Errorf("Distance is more than 3.0 and tag not valid")
			}
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("Test Delete", func(t *testing.T) {
		itemsData := []OSMObject{}

		for i := 1; i < 100000; i++ {
			tag := rand.Intn(10)
			val := rand.Intn(10)

			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: lat,
				Lon: lon,
				Tag: map[int]int{tag: val},
			})
		}

		rt := NewRtree(25, 50, 2)
		for _, item := range itemsData {
			bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001},
				[]float64{item.Lat + 0.0001, item.Lon + 0.0001})

			rt.InsertLeaf(bound, item, false)
		}

		for i := 0; i < len(itemsData); i++ {
			if i%2 == 0 {

				bound := NewRtreeBoundingBox(2, []float64{itemsData[i].Lat - 0.0001,
					itemsData[i].Lon - 0.0001}, []float64{itemsData[i].Lat + 0.0001, itemsData[i].Lon + 0.0001})
				itemsData[i].SetBound(bound)
				rt.Delete(itemsData[i])

			}
		}

		for i := 0; i < len(itemsData); i++ {
			if i%2 == 0 {
				node, _ := rt.FindLeaf(itemsData[i], rt.Root, 1)
				assert.Nil(t, node)
			}
		}

		countLeaf := 0
		expectedLeafLevel := 3

		traverseRtreeAndTestIfRtreePropertiesCorrect(rt, rt.Root, &countLeaf, expectedLeafLevel, 1, t)

	})

	t.Run("Test Delete small data", func(t *testing.T) {
		itemsData := []OSMObject{}

		for i := 1; i < 100; i++ {
			tag := rand.Intn(10)
			val := rand.Intn(10)

			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: lat,
				Lon: lon,
				Tag: map[int]int{tag: val},
			})
		}

		rt := NewRtree(10, 25, 2)
		for _, item := range itemsData {
			bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001},
				[]float64{item.Lat + 0.0001, item.Lon + 0.0001})

			rt.InsertLeaf(bound, item, false)

		}

		for i := 0; i < len(itemsData); i++ {
			if i%2 == 0 {

				bound := NewRtreeBoundingBox(2, []float64{itemsData[i].Lat - 0.0001,
					itemsData[i].Lon - 0.0001}, []float64{itemsData[i].Lat + 0.0001, itemsData[i].Lon + 0.0001})
				itemsData[i].SetBound(bound)
				rt.Delete(itemsData[i])

			}
		}

		for i := 0; i < len(itemsData); i++ {
			if i%2 == 0 {
				leafNode, _ := rt.FindLeaf(itemsData[i], rt.Root, 1)

				assert.Nil(t, leafNode)
			}
		}

		countLeaf := 0
		expectedLeafLevel := 2
		traverseRtreeAndTestIfRtreePropertiesCorrect(rt, rt.Root, &countLeaf, expectedLeafLevel, 1, t)

	})
}

func TestInsertLevel(t *testing.T) {
	t.Run("Test Insert Level", func(t *testing.T) {
		var itemsData = []OSMObject{}

		for i := 0; i < 30; i++ {
			tag := rand.Intn(10)
			val := rand.Intn(10)

			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: lat,
				Lon: lon,
				Tag: map[int]int{tag: val},
			})
		}

		rt := NewRtree(10, 25, 2)
		for _, item := range itemsData {
			bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001},
				[]float64{item.Lat + 0.0001, item.Lon + 0.0001})

			rt.InsertLeaf(bound, item, false)
		}

		// insert level
		lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
		newLeaf := OSMObject{
			ID:  31,
			Lat: lat,
			Lon: lon,
			Tag: map[int]int{1: 1},
		}

		bound := NewRtreeBoundingBox(2, []float64{newLeaf.Lat - 0.0001, newLeaf.Lon - 0.0001},
			[]float64{newLeaf.Lat + 0.0001, newLeaf.Lon + 0.0001})

		newLeaf.SetBound(bound)

		newNode := &RtreeNode{
			Leaf:   newLeaf,
			Bound:  bound,
			IsLeaf: false,
		}
		rt.insertLevel(newNode, 2)

		leafNode, resultLevel := rt.FindLeaf(newLeaf, rt.Root, 1)

		isLeafFound := false

		for _, entry := range leafNode.Items {
			if entry.Leaf.ID == newLeaf.ID {
				isLeafFound = true
			}
		}

		assert.Equal(t, true, isLeafFound)
		assert.Equal(t, 2, resultLevel)
		countLeaf := 0
		traverseRtreeAndTestIfRtreePropertiesCorrect(rt, rt.Root, &countLeaf, 2, 1, t)

	})
}

func TestChooseLevel(t *testing.T) {
	t.Run("Test Choose Level 2  after inserted", func(t *testing.T) {
		var itemsData = []OSMObject{}

		for i := 1; i < 30; i++ {
			tag := rand.Intn(10)
			val := rand.Intn(10)

			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				ID:  i,
				Lat: lat,
				Lon: lon,
				Tag: map[int]int{tag: val},
			})
		}

		rt := NewRtree(10, 25, 2)
		for _, item := range itemsData {
			bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001},
				[]float64{item.Lat + 0.0001, item.Lon + 0.0001})

			rt.InsertLeaf(bound, item, false)
		}

		// insert level
		lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
		newLeaf := OSMObject{
			ID:  31,
			Lat: lat,
			Lon: lon,
			Tag: map[int]int{1: 1},
		}

		bound := NewRtreeBoundingBox(2, []float64{newLeaf.Lat - 0.0001, newLeaf.Lon - 0.0001},
			[]float64{newLeaf.Lat + 0.0001, newLeaf.Lon + 0.0001})

		newLeaf.SetBound(bound)

		newNode := &RtreeNode{
			Leaf:   newLeaf,
			Bound:  bound,
			IsLeaf: false,
		}
		rt.insertLevel(newNode, 2)

		choosedNode := rt.chooseLevel(rt.Root, newNode.GetBound(), 2, 1)

		isNewNodeThere := false
		for i := 0; i < len(choosedNode.Items); i++ {
			if choosedNode.Items[i] == newNode {
				isNewNodeThere = true
			}
		}

		assert.Equal(t, true, isNewNodeThere)

		choosedNodeRoot := rt.chooseLevel(rt.Root, newNode.GetBound(), 1, 1)

		isChoosenNodeThere := false
		for i := 0; i < len(choosedNodeRoot.Items); i++ {
			if choosedNodeRoot.Items[i] == choosedNode {
				isChoosenNodeThere = true
			}
		}
		assert.Equal(t, true, isChoosenNodeThere)

	})
}

// go test  . -bench=[nama_benchmark] -v -benchmem -cpuprofile=cpu.out
// BenchmarkNNearestNeighbors-12    	   47371	     34230 ns/op	     29214 ops/sec	   70560 B/op	      12 allocs/op
func BenchmarkNNearestNeighbors(b *testing.B) {
	itemsData := []OSMObject{}

	for i := 0; i < 100000; i++ {

		lat, lon := randomLatLon(-6.809629930307937, -6.896578040216839, 105.99351536809907, 112.60245825180131)
		itemsData = append(itemsData, OSMObject{
			ID:  i,
			Lat: lat,
			Lon: lon,
		})
	}

	rt := NewRtree(25, 50, 2)
	for _, item := range itemsData {
		bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})

		rt.InsertLeaf(bound, item, false)
	}

	myLocation := Point{-7.548263971398246, 110.78226484631368}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.NearestNeighboursPQ(5, myLocation)
	}

	b.StopTimer()
	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "ops/sec")
}

// BenchmarkInsert-12    	   85011	     20896 ns/op	     47856 ops/sec	   16932 B/op	     784 allocs/op
func BenchmarkInsert(b *testing.B) {
	itemsData := []OSMObject{}

	for i := 0; i < 100000; i++ {

		lat, lon := randomLatLon(-6.809629930307937, -6.896578040216839, 105.99351536809907, 112.60245825180131)
		itemsData = append(itemsData, OSMObject{
			ID:  i,
			Lat: lat,
			Lon: lon,
		})
	}

	rt := NewRtree(25, 50, 2)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		randInt := rand.Intn(100000)
		item := itemsData[randInt]
		bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})

		rt.InsertLeaf(bound, item, false)
	}
	b.StopTimer()
	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "ops/sec")
}

// BenchmarkImprovedNearestNeighbor-12    	   49046	     21880 ns/op	     45703 ops/sec	   37472 B/op	      10 allocs/op
func BenchmarkImprovedNearestNeighbor(b *testing.B) {
	itemsData := []OSMObject{}

	for i := 0; i < 100000; i++ {

		lat, lon := randomLatLon(-6.809629930307937, -6.896578040216839, 105.99351536809907, 112.60245825180131)
		itemsData = append(itemsData, OSMObject{
			ID:  i,
			Lat: lat,
			Lon: lon,
		})
	}

	rt := NewRtree(25, 50, 2)
	for _, item := range itemsData {
		bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})

		rt.InsertLeaf(bound, item, false)
	}
	myLocation := Point{-7.548263971398246, 110.78226484631368}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.ImprovedNearestNeighbor(myLocation)
	}

	b.StopTimer()
	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "ops/sec")
}

// BenchmarkNearestNeighborRadiusFilterOsmFeature-12    	   53122	     20858 ns/op	     47944 ops/sec	   39904 B/op	      13 allocs/op
func BenchmarkNearestNeighborRadiusFilterOsmFeature(b *testing.B) {

	var itemsData []OSMObject = []OSMObject{}
	for i := 0; i < 100000; i++ {
		tag := rand.Intn(10)
		val := rand.Intn(10)

		lat, lon := randomLatLon(-7.764433230190314, -6.666039357161423, 110.36250037487716, 111.43103761218967)
		itemsData = append(itemsData, OSMObject{
			ID:  i,
			Lat: lat,
			Lon: lon,
			Tag: map[int]int{tag: val},
		})
	}

	rt := NewRtree(25, 50, 2)
	for _, item := range itemsData {
		bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})
		rt.InsertLeaf(bound, item, false)
	}

	myLocation := Point{-7.760335932763678, 110.37671195413539}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.NearestNeighboursRadiusFilterOSM(5, 0, myLocation, 3.0, 1)

	}

	b.StopTimer()
	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "ops/sec")

}

// BenchmarkSearch-12    	36976773	        33.50 ns/op	  29852521 ops/sec	      32 B/op	       1 allocs/op
func BenchmarkSearch(b *testing.B) {
	itemsData := []OSMObject{}

	for i := 0; i < 100000; i++ {

		lat, lon := randomLatLon(-6.809629930307937, -6.896578040216839, 105.99351536809907, 112.60245825180131)
		itemsData = append(itemsData, OSMObject{
			ID:  i,
			Lat: lat,
			Lon: lon,
		})
	}

	rt := NewRtree(25, 50, 2)
	for _, item := range itemsData {
		bound := NewRtreeBoundingBox(2, []float64{item.Lat - 0.0001, item.Lon - 0.0001}, []float64{item.Lat + 0.0001, item.Lon + 0.0001})
		rt.InsertLeaf(bound, item, false)
	}
	myLocation := Point{-7.548263971398246, 110.78226484631368}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.Search(NewRtreeBoundingBox(2, []float64{myLocation.Lat - 0.0001, myLocation.Lon - 0.0001}, []float64{myLocation.Lat + 0.0001, myLocation.Lon + 0.0001}))
	}

	b.StopTimer()
	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "ops/sec")
}