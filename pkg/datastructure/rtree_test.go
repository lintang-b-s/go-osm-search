package datastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
)

// this is trash

type doc struct {
	id int
	x  float64
	y  float64
}

func traverseRtreeAndTestIfBoundingBoxCorrect(node *RtreeNode, countLeaf *int, t *testing.T) {
	if node.isLeaf {
		maxBB := node.items[0].getBound()
		for _, item := range node.items {
			*countLeaf++
			bb := item.getBound()
			maxBB = stretch(maxBB, bb)
		}

		if !node.bound.isBBSame(maxBB) {
			t.Errorf("Bounding box not same")
		}
	} else {
		maxBB := node.items[0].getBound()

		for _, item := range node.items {
			bb := item.getBound()
			maxBB = stretch(maxBB, bb)
			traverseRtreeAndTestIfBoundingBoxCorrect(item.(*RtreeNode), countLeaf, t)
		}

		if !node.bound.isBBSame(maxBB) {
			t.Errorf("Bounding box not same")
		}
	}
}

func TestInsertRtree(t *testing.T) {
	itemsData := []doc{}
	for i := 0; i < 100; i++ {
		itemsData = append(itemsData, doc{
			id: i,
			x:  float64(i),
			y:  float64(i),
		})
	}

	tests := []struct {
		name        string
		items       []doc
		expectItems int
	}{
		{
			name: "Insert 100 item",
			items: append(itemsData, []doc{
				{
					id: 100,
					x:  0,
					y:  -5,
				},
				{
					id: 101,
					x:  2,
					y:  -10,
				},
				{
					id: 102,
					x:  3,
					y:  -15,
				},
			}...),
			expectItems: 103,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := NewRtree[doc](25, 50, 2)
			for _, item := range tt.items {
				minVal, maxVal := []float64{item.x - 1, item.y - 1}, []float64{item.x + 1, item.y + 1}
				rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), item)

			}
			assert.Equal(t, 103, rt.size)

			countLeaf := 0
			traverseRtreeAndTestIfBoundingBoxCorrect(rt.mRoot, &countLeaf, t)
			assert.Equal(t, tt.expectItems, countLeaf)
		})
	}

	t.Run("Insert 5 items", func(t *testing.T) {
		rt := NewRtree[doc](25, 50, 2)
		for i := 0; i < 5; i++ {
			item := itemsData[i]
			minVal, maxVal := []float64{item.x - 1, item.y - 1}, []float64{item.x + 1, item.y + 1}
			rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), item)

		}
		assert.Equal(t, 5, rt.size)
		root := rt.mRoot
		for i, item := range root.items {
			assert.Equal(t, itemsData[i].id, item.(*RtreeLeaf[doc]).leaf.id)
		}

		countLeaf := 0
		traverseRtreeAndTestIfBoundingBoxCorrect(rt.mRoot, &countLeaf, t)
		assert.Equal(t, 5, countLeaf)
	})

}

func TestChooseSubtree(t *testing.T) {
	t.Run("Choose subtree", func(t *testing.T) {
		items := []BoundedItem{
			&RtreeNode{
				bound: NewRtreeBoundingBox(2, []float64{-1, -1}, []float64{1, 1}),
				items: []BoundedItem{
					&RtreeNode{
						bound:  NewRtreeBoundingBox(2, []float64{-1, -1}, []float64{1, 1}),
						items:  []BoundedItem{},
						isLeaf: true,
					},
					&RtreeNode{
						bound:  NewRtreeBoundingBox(2, []float64{0, 0}, []float64{0, 0}),
						items:  []BoundedItem{},
						isLeaf: true,
					},
				},
				isLeaf: false,
			},

			&RtreeNode{
				bound: NewRtreeBoundingBox(2, []float64{10, 10}, []float64{20, 20}),
				items: []BoundedItem{
					&RtreeNode{
						bound:  NewRtreeBoundingBox(2, []float64{10, 10}, []float64{20, 20}),
						items:  []BoundedItem{},
						isLeaf: true,
					},
				},
				isLeaf: false,
			},
		}

		rt := NewRtree[doc](1, 2, 2)
		rt.mRoot = &RtreeNode{
			bound: NewRtreeBoundingBox(2, []float64{0, 0}, []float64{0, 0}),
			items: items,
		}

		for _, item := range items {
			rt.mRoot.bound = stretch(rt.mRoot.bound, item.getBound())
		}

		newBB := NewRtreeBoundingBox(2, []float64{12, 12}, []float64{18, 18})

		rt.mRoot.bound = stretch(rt.mRoot.bound, newBB)

		choosedNode := rt.chooseSubtree(rt.mRoot, newBB)
		assert.Equal(t, items[1].(*RtreeNode).items[0], choosedNode)
	})

}

func TestSearch(t *testing.T) {

	t.Run("Search", func(t *testing.T) {
		itemsData := []doc{}
		for i := 0; i < 100; i++ {
			itemsData = append(itemsData, doc{
				id: i,
				x:  float64(i),
				y:  float64(i),
			})
		}

		rt := NewRtree[doc](10, 25, 2)
		for _, item := range itemsData {
			minVal, maxVal := []float64{item.x - 1, item.y - 1}, []float64{item.x + 1, item.y + 1}
			rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), item)

		}

		countLeaf := 0
		traverseRtreeAndTestIfBoundingBoxCorrect(rt.mRoot, &countLeaf, t)

		results := rt.Search(NewRtreeBoundingBox(2, []float64{0, 0}, []float64{50, 50}))

		for _, item := range results {

			itembb := item.getBound()
			if !overlaps(itembb, NewRtreeBoundingBox(2, []float64{0, 0}, []float64{50, 50})) {
				t.Errorf("Bounding box is not correct")

			}
		}
	})
}

func TestSplit(t *testing.T) {
	t.Run("Split", func(t *testing.T) {
		itemsData := []doc{}
		for i := 0; i < 26; i++ {
			itemsData = append(itemsData, doc{
				id: i,
				x:  float64(i),
				y:  float64(i),
			})
		}

		rt := NewRtree[doc](10, 25, 2)
		minVal, maxVal := []float64{itemsData[0].x - 1, itemsData[0].y - 1}, []float64{itemsData[0].x + 1, itemsData[0].y + 1}

		rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), itemsData[0])
		for i := 1; i < 26; i++ {
			item := itemsData[i]
			minVal, maxVal := []float64{item.x - 1, item.y - 1}, []float64{item.x + 1, item.y + 1}
			newLeaf := &RtreeLeaf[doc]{item, NewRtreeBoundingBox(2, minVal, maxVal)}
			rt.mRoot.items = append(rt.mRoot.items, newLeaf)
		}

		newNode := rt.split(rt.mRoot)

		assert.Less(t, len(newNode.items), 25)
		assert.Less(t, len(rt.mRoot.items), 25)

		countLeaf := 0
		traverseRtreeAndTestIfBoundingBoxCorrect(rt.mRoot, &countLeaf, t)
		traverseRtreeAndTestIfBoundingBoxCorrect(newNode, &countLeaf, t)
	})
}

func TestOverflowTreatment(t *testing.T) {
	t.Run("Overflow treatment", func(t *testing.T) {
		itemsData := []doc{}
		for i := 0; i < 26; i++ {
			itemsData = append(itemsData, doc{
				id: i,
				x:  float64(i),
				y:  float64(i),
			})
		}

		rt := NewRtree[doc](10, 25, 2)
		minVal, maxVal := []float64{itemsData[0].x - 1, itemsData[0].y - 1}, []float64{itemsData[0].x + 1, itemsData[0].y + 1}

		rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), itemsData[0])
		for i := 1; i < 26; i++ {
			item := itemsData[i]
			minVal, maxVal := []float64{item.x - 1, item.y - 1}, []float64{item.x + 1, item.y + 1}
			newLeaf := &RtreeLeaf[doc]{item, NewRtreeBoundingBox(2, minVal, maxVal)}
			rt.mRoot.items = append(rt.mRoot.items, newLeaf)
		}

		oldRoot := rt.mRoot
		rt.overflowTreatment(rt.mRoot, true)

		assert.NotEqual(t, oldRoot, rt.mRoot)
		assert.Equal(t, 2, len(rt.mRoot.items))

		countLeaf := 0
		traverseRtreeAndTestIfBoundingBoxCorrect(rt.mRoot, &countLeaf, t)
	})

}

func randomLatLon(minLat, maxLat, minLon, maxLon float64) (float64, float64) {
	rand.Seed(uint64(time.Now().UnixNano()))
	lat := minLat + rand.Float64()*(maxLat-minLat)
	lon := minLon + rand.Float64()*(maxLon-minLon)
	return lat, lon
}

func TestNNearestNeighbors(t *testing.T) {
	t.Run("Test N Nearest Neighbors kota surakarta", func(t *testing.T) {
		itemsData := []OSMObject{
			{
				id:  7,
				lat: -7.546392935195944,
				lon: 110.77718220472673,
			},
			{
				id:  6,
				lat: -7.5559986670115675,
				lon: 110.79466621171177,
			},
			{
				id:  5,
				lat: -7.555869730414206,
				lon: 110.80500875243253,
			},
			{
				id:  4,
				lat: -7.571289544570394,
				lon: 110.8301500772816,
			},
			{
				id:  3,
				lat: -7.7886707815273155,
				lon: 110.361625035987,
			}, {
				id:  2,
				lat: -7.8082872068169475,
				lon: 110.35793427899466,
			},
			{
				id:  1,
				lat: -7.759889166547908,
				lon: 110.36689459108496,
			},
			{
				id:  0,
				lat: -7.550561079106621,
				lon: 110.7837156929654,
			},
		}

		for i := 8; i < 500; i++ {
			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				id:  i,
				lat: lat,
				lon: lon,
			})
		}

		rt := NewRtree[OSMObject](25, 50, 2)
		for _, item := range itemsData {
			minVal, maxVal := []float64{item.lat - 0.0001, item.lon - 0.0001}, []float64{item.lat + 0.0001, item.lon + 0.0001}
			rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), item)
		}

		myLocation := Point{-7.548263971398246, 110.78226484631368}
		results := rt.FastNNearestNeighbors(5, myLocation)

		assert.Equal(t, 5, len(results))
		assert.Equal(t, 0, results[0].id)
		assert.Equal(t, 7, results[1].id)
		assert.Equal(t, 6, results[2].id)
		assert.Equal(t, 5, results[3].id)
		assert.Equal(t, 4, results[4].id)
	})
}

func TestNearestNeighbor(t *testing.T) {
	t.Run("Test N Nearest Neighbors kota surakarta", func(t *testing.T) {
		itemsData := []OSMObject{
			{
				id:  7,
				lat: -7.546392935195944,
				lon: 110.77718220472673,
			},
			{
				id:  6,
				lat: -7.5559986670115675,
				lon: 110.79466621171177,
			},
			{
				id:  5,
				lat: -7.555869730414206,
				lon: 110.80500875243253,
			},
			{
				id:  4,
				lat: -7.571289544570394,
				lon: 110.8301500772816,
			},
			{
				id:  3,
				lat: -7.7886707815273155,
				lon: 110.361625035987,
			}, {
				id:  2,
				lat: -7.8082872068169475,
				lon: 110.35793427899466,
			},
			{
				id:  1,
				lat: -7.759889166547908,
				lon: 110.36689459108496,
			},
			{
				id:  1000,
				lat: -7.550561079106621,
				lon: 110.7837156929654,
			},
			{
				id: 1001,
				lat: -7.755002453207869,
				lon: 110.37712514761436,
			},
		}

		for i := 8; i < 500; i++ {
			lat, lon := randomLatLon(-6.107481038495567, -5.995288834299442, 106.13128828884481, 107.0509652831274)
			itemsData = append(itemsData, OSMObject{
				id:  i,
				lat: lat,
				lon: lon,
			})
		}

		rt := NewRtree[OSMObject](25, 50, 2)
		for _, item := range itemsData {
			minVal, maxVal := []float64{item.lat - 0.0001, item.lon - 0.0001}, []float64{item.lat + 0.0001, item.lon + 0.0001}
			rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), item)
		}

		myLocation := Point{-7.760335932763678, 110.37671195413539}

		result := rt.ImprovedNearestNeighbor(myLocation)
		assert.Equal(t, 1001, result.id)

	})
}

func BenchmarkNNearestNeighbors(b *testing.B) {
	itemsData := []OSMObject{}

	for i := 0; i < 100000; i++ {

		lat, lon := randomLatLon(-6.809629930307937, -6.896578040216839, 105.99351536809907, 112.60245825180131)
		itemsData = append(itemsData, OSMObject{
			id:  i,
			lat: lat,
			lon: lon,
		})
	}

	rt := NewRtree[OSMObject](25, 50, 2)
	for _, item := range itemsData {
		minVal, maxVal := []float64{item.lat - 0.0001, item.lon - 0.0001}, []float64{item.lat + 0.0001, item.lon + 0.0001}
		rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), item)
	}

	myLocation := Point{-7.548263971398246, 110.78226484631368}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.FastNNearestNeighbors(5, myLocation)
	}

}

func BenchmarkInsert(b *testing.B) {
	itemsData := []OSMObject{}

	for i := 0; i < 100000; i++ {

		lat, lon := randomLatLon(-6.809629930307937, -6.896578040216839, 105.99351536809907, 112.60245825180131)
		itemsData = append(itemsData, OSMObject{
			id:  i,
			lat: lat,
			lon: lon,
		})
	}

	rt := NewRtree[OSMObject](25, 50, 2)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		randInt := rand.Intn(100000)
		item := itemsData[randInt]
		minVal, maxVal := []float64{item.lat - 0.0001, item.lon - 0.0001}, []float64{item.lat + 0.0001, item.lon + 0.0001}
		rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), item)
	}

}

func BenchmarkImprovedNearestNeighbor(b *testing.B) {
	itemsData := []OSMObject{}

	for i := 0; i < 100000; i++ {

		lat, lon := randomLatLon(-6.809629930307937, -6.896578040216839, 105.99351536809907, 112.60245825180131)
		itemsData = append(itemsData, OSMObject{
			id:  i,
			lat: lat,
			lon: lon,
		})
	}

	rt := NewRtree[OSMObject](25, 50, 2)
	for _, item := range itemsData {
		minVal, maxVal := []float64{item.lat - 0.0001, item.lon - 0.0001}, []float64{item.lat + 0.0001, item.lon + 0.0001}
		rt.insertLeaf(NewRtreeBoundingBox(2, minVal, maxVal), item)
	}
	myLocation := Point{-7.548263971398246, 110.78226484631368}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.ImprovedNearestNeighbor(myLocation)
	}
}
