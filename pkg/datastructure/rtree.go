package datastructure

import (
	"container/heap"
	"math"
	"osm-search/pkg/geo"
	"sort"
)

// cuma rewrite dari implementatasi c++ ini: https://github.com/virtuald/r-star-tree/
// + add k-nearest neighbors & Search.
// https://infolab.usc.edu/csci599/Fall2001/paper/rstar-tree.pdf
// https://dl.acm.org/doi/10.1145/971697.602266

func assertt(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}

const (
	CHOOSE_SUBTREE_P = 32
	REINSERT_P       = 0.3
)

type RtreeBoundingBox struct {
	// number of dimensions
	Dim int
	// Edges[i][0] = low value, Edges[i][1] = high value
	// i = 0,...,Dim
	Edges [][2]float64
}

func NewRtreeBoundingBox(dim int, minVal []float64, maxVal []float64) RtreeBoundingBox {
	b := RtreeBoundingBox{Dim: dim, Edges: make([][2]float64, dim)}
	for axis := 0; axis < dim; axis++ {
		b.Edges[axis] = [2]float64{minVal[axis], maxVal[axis]}
	}

	return b
}

// reset forces all edges to extremes so we can stretch them later.
func reset(b RtreeBoundingBox) RtreeBoundingBox {
	newBB := NewRtreeBoundingBox(b.Dim, make([]float64, b.Dim), make([]float64, b.Dim))
	for axis := 0; axis < b.Dim; axis++ {
		newBB.Edges[axis][0] = math.MaxFloat64
		newBB.Edges[axis][1] = math.Inf(-1)
	}
	return newBB
}

// // maximumBounds returns a new bounding box that has the maximum boundaries.
// func maximumBounds(dim int) RtreeBoundingBox {
// 	bound := RtreeBoundingBox{
// 		Dim:   dim,
// 		Edges: make([][2]float64, dim),
// 	}
// 	bound.reset()
// 	return bound
// }

// stretch fits another box inside this box, returns true if a stretch occurred.
func stretch(b RtreeBoundingBox, bb RtreeBoundingBox) RtreeBoundingBox {

	newBB := NewRtreeBoundingBox(b.Dim, make([]float64, b.Dim), make([]float64, b.Dim))
	for axis := 0; axis < b.Dim; axis++ {
		if b.Edges[axis][0] > bb.Edges[axis][0] {
			newBB.Edges[axis][0] = bb.Edges[axis][0]
		} else {
			newBB.Edges[axis][0] = b.Edges[axis][0]
		}

		if b.Edges[axis][1] < bb.Edges[axis][1] {
			newBB.Edges[axis][1] = bb.Edges[axis][1]
		} else {
			newBB.Edges[axis][1] = b.Edges[axis][1]
		}
	}
	return newBB
}

func boundingBox(b RtreeBoundingBox, bb RtreeBoundingBox) RtreeBoundingBox {
	newBound := NewRtreeBoundingBox(b.Dim, make([]float64, b.Dim), make([]float64, b.Dim))

	for axis := 0; axis < b.Dim; axis++ {
		if b.Edges[axis][0] <= bb.Edges[axis][0] {
			newBound.Edges[axis][0] = b.Edges[axis][0]
		} else {
			newBound.Edges[axis][0] = bb.Edges[axis][0]
		}

		if b.Edges[axis][1] >= bb.Edges[axis][1] {
			newBound.Edges[axis][1] = b.Edges[axis][1]
		} else {
			newBound.Edges[axis][1] = bb.Edges[axis][1]
		}
	}

	return newBound
}

// edgeDeltas returns the sum of all (high - low) for each dimension. (margin)
func edgeDeltas(b RtreeBoundingBox) float64 {
	//  Here the margin is the sum of the lengths of the
	// edges of a rectangle
	distance := 0.0
	for axis := 0; axis < b.Dim; axis++ {
		distance += b.Edges[axis][1] - b.Edges[axis][0]
	}
	return distance
}

// area calculates the area (in N dimensions) of a bounding box.
func area(b RtreeBoundingBox) float64 {
	area := 1.0
	for axis := 0; axis < b.Dim; axis++ {
		area *= b.Edges[axis][1] - b.Edges[axis][0]
	}
	return area
}

// encloses determines if b fully contains bb. return true if bb is fully contained in b.
func encloses(b RtreeBoundingBox, bb RtreeBoundingBox) bool {
	for axis := 0; axis < b.Dim; axis++ {

		if bb.Edges[axis][0] < b.Edges[axis][0] || b.Edges[axis][1] < bb.Edges[axis][1] {
			/*
				____________________
				|	b			    |
				|		________________
				|	   |	   		  	|
				|	   | bb	   		  	|
				|	   |________________|
				|					|
				|___________________|

				or


					____________________
					|	bb			    |
					|		________________
					|	   |	   		  	|
					|	   | b   		  	|
					|	   |________________|
					|					|
					|___________________|
			*/

			return false
		}
	}

	return true
}

// overlaps checks if two bounding boxes overlap.
func overlaps(b RtreeBoundingBox, bb RtreeBoundingBox) bool {
	for axis := 0; axis < b.Dim; axis++ {
		if !(b.Edges[axis][0] < bb.Edges[axis][1]) || !(bb.Edges[axis][0] < b.Edges[axis][1]) {
			/*


				____________________	______________________
				|	b				|   |					   |
				|					|   |			bb		   |
				|	   				|   |					   |
				____________________    |  ____________________

				or

				____________________	______________________
				|	bb				|   |					   |
				|					|   |			b		   |
				|	   				|   |					   |
				____________________    |  ____________________


			*/
			return false
		}
	}

	return true
}

// overlap calculates total overlapping region area (0 if no overlap).
func overlap(b RtreeBoundingBox, bb RtreeBoundingBox) float64 {
	area := 1.0

	for axis := 0; axis < b.Dim && area != 0; axis++ {
		bMin := b.Edges[axis][0]
		bMax := b.Edges[axis][1]
		bbMin := bb.Edges[axis][0]
		bbMax := bb.Edges[axis][1]

		if bMin < bbMin {
			if bbMax < bMax {
				/*
					____________________
					|	b				|
					|		_______		|
					|	   |	   |	|
					|	   | bb	   |	|
					|	   |_______|	|
					|					|
					|___________________|
				*/
				area *= float64(bbMax - bbMin)
			} else {

				/*
					____________________
					|	b			    |
					|		________________
					|	   |	   		  	|
					|	   | bb	   		  	|
					|	   |________________|
					|					|
					|___________________|
				*/
				area *= float64(bMax - bbMin)
			}
			continue
		} else if bMin < bbMax {

			if bMax < bbMax {
				/*
					____________________
					|	bb				|
					|		_______		|
					|	   |	   |	|
					|	   | b	   |	|
					|	   |_______|	|
					|					|
					|___________________|
				*/
				area *= float64(bMax - bMin)
			} else {
				/*
					____________________
					|	bb			    |
					|		________________
					|	   |	   		  	|
					|	   | b	   		  	|
					|	   |________________|
					|					|
					|___________________|
				*/
				area *= float64(bbMax - bMin)
			}
			continue
		}
		// no overlap
		return 0.0
	}

	return area
}

// distanceFromCenter  distances between the center of the bounding box and the center of the entry bb.
func (b *RtreeBoundingBox) distanceFromCenter(bb RtreeBoundingBox) float64 {
	// b = bounding box  of node N
	// bb = bounding box of the entry E (entries in the node)
	distance := 0.0

	// euclidian
	for axis := 0; axis < b.Dim; axis++ {
		centerB := float64(b.Edges[axis][0]+b.Edges[axis][1]) / 2.0
		centerBB := float64(bb.Edges[axis][0]+bb.Edges[axis][1]) / 2.0
		distance += math.Pow(centerB-centerBB, 2)
	}

	return distance
}

// isBBSame determines if two bounding boxes are identical
func (b *RtreeBoundingBox) isBBSame(bb RtreeBoundingBox) bool {
	for axis := 0; axis < b.Dim; axis++ {
		if b.Edges[axis][0] != bb.Edges[axis][0] || b.Edges[axis][1] != bb.Edges[axis][1] {
			return false
		}
	}

	return true
}

func stretchBoundingBox(mBound RtreeBoundingBox, item BoundedItem) RtreeBoundingBox {
	return stretch(mBound, item.getBound())
}

func sortBoundedItemsByFirstEdge(mAxis int, items []BoundedItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].getBound().Edges[mAxis][0] < items[j].getBound().Edges[mAxis][0]
	})
}

func sortBoundedItemsBySecondEdge(mAxis int, items []BoundedItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].getBound().Edges[mAxis][1] < items[j].getBound().Edges[mAxis][1]
	})
}

func sortIncreasingBoundedItemsByDistanceFromCenter(mCenter RtreeBoundingBox, items []BoundedItem) {
	sort.Slice(items, func(i, j int) bool {
		return mCenter.distanceFromCenter(items[i].getBound()) < mCenter.distanceFromCenter(items[j].getBound())
	})
}

func sortDecreasingBoundedItemsByDistanceFromCenter(mCenter RtreeBoundingBox, items []BoundedItem) {
	sort.Slice(items, func(i, j int) bool {
		return mCenter.distanceFromCenter(items[i].getBound()) > mCenter.distanceFromCenter(items[j].getBound())
	})
}

func sortBoundedItemsByAreaEnlargement(bbarea float64, items []BoundedItem) {
	sort.Slice(items, func(i, j int) bool {
		return bbarea-area(items[i].getBound()) < bbarea-area(items[j].getBound())

	})
}

func sortIncreasingBoundedItemsByOverlapEnlargement(items []BoundedItem, center RtreeBoundingBox) {
	sort.Slice(items, func(i, j int) bool {

		return overlap(items[i].getBound(), center) < overlap(items[j].getBound(), center)
		// return items[i].getBound().overlap(center) < items[j].getBound().overlap(center)
	})
}

func sortDecreasingBoundedItemsByOverlapEnlargement(items []BoundedItem, center RtreeBoundingBox) {
	sort.Slice(items, func(i, j int) bool {
		return overlap(items[i].getBound(), center) > overlap(items[j].getBound(), center)

		// return items[i].getBound().overlap(center) > items[j].getBound().overlap(center)
	})
}

type BoundedItem interface {
	getBound() RtreeBoundingBox
	isLeafNode() bool
}

// rtree node. can be either a leaf node or a internal node
type RtreeNode struct {
	// entries. can be either a leaf node or a  internal node.
	// leafNode has items in the form of a list of RtreeLeaf
	items  []BoundedItem
	parent *RtreeNode

	bound RtreeBoundingBox
	// isLeaf. true if  this node is a leafNode.
	isLeaf bool
}

// isLeaf. true if this node is a leafNode.
func (node *RtreeNode) isLeafNode() bool {
	return node.isLeaf
}

func (node *RtreeNode) getBound() RtreeBoundingBox {
	return node.bound
}

// leaf entry
type RtreeLeaf[G any] struct {
	leaf G

	bound RtreeBoundingBox
}

func (leaf *RtreeLeaf[G]) isLeafNode() bool {
	return false
}

func (leaf *RtreeLeaf[G]) getBound() RtreeBoundingBox {
	return leaf.bound
}

type Rtree[LeafType any] struct {
	mRoot         *RtreeNode
	size          int
	minChildItems int
	maxChildItems int
	dimensions    int
}

func NewRtree[LeafType any](minChildItems, maxChildItems, dimensions int) *Rtree[LeafType] {
	return &Rtree[LeafType]{
		mRoot:         nil,
		size:          0,
		minChildItems: minChildItems,
		maxChildItems: maxChildItems,
		dimensions:    dimensions,
	}
}

func (rt *Rtree[LeafType]) insertLeaf(bound RtreeBoundingBox, leaf LeafType) {

	newLeaf := &RtreeLeaf[LeafType]{}
	newLeaf.bound = bound
	newLeaf.leaf = leaf

	if rt.mRoot == nil {
		rt.mRoot = &RtreeNode{}
		rt.mRoot.isLeaf = true // set root as leaf node

		rt.mRoot.items = make([]BoundedItem, 0, rt.minChildItems)

		rt.mRoot.items = append(rt.mRoot.items, newLeaf) // add new leaf data to the root
		rt.mRoot.bound = bound
	} else {
		rt.insertInternal(newLeaf, rt.mRoot, true)
	}
	rt.size++

}

func (rt *Rtree[LeafType]) insertInternal(leaf *RtreeLeaf[LeafType], root *RtreeNode, firstInsert bool) *RtreeNode {

	// I1: Invoke ChooseSubtree. with the level as a parameter,
	// to find an appropriate node N, in which to place the new entry E

	leafNode := rt.chooseSubtree(root, leaf.bound)

	// if this node is a leafNode then add the leaf data to the  leafNode
	//I2: accommodate E in N.
	leafNode.items = append(leafNode.items, leaf)

	// I2: if node N has M+1 entries. invoke OverflowTreatment
	// with the level of N as a parameter [for reinsertion or split]
	if len(leafNode.items) > rt.maxChildItems {
		rt.overflowTreatment(leafNode, firstInsert)

	}

	return nil
}

func (rt *Rtree[LeafType]) overflowTreatment(level *RtreeNode, firstInsert bool) {
	// OT1: If the level is not the root level and this is the first
	// call of OverflowTreatment in the given level
	// during the Insertion of one data rectangle, then
	// invoke Reinsert
	if level != rt.mRoot && firstInsert {
		rt.reinsert(level)
		return
	}

	//else invoke Split
	newNode := rt.split(level)

	// I3:  If OverflowTreatment caused a split of the root, create a
	// new root whose children are the two resulting nodes (old root & newNode).
	if level == rt.mRoot {
		// benar
		newRoot := &RtreeNode{}
		newRoot.isLeaf = false

		newRoot.items = make([]BoundedItem, 0, rt.minChildItems)
		newRoot.items = append(newRoot.items, rt.mRoot)
		newRoot.items = append(newRoot.items, newNode)
		rt.mRoot.parent = newRoot
		newNode.parent = newRoot

		// I4: Adjust all covering rectangles in the insertion path
		// such that they are minimum bounding boxes
		// enclosing their children rectangles
		newRoot.bound = NewRtreeBoundingBox(rt.dimensions, make([]float64, rt.dimensions), make([]float64, rt.dimensions))

		newRoot.bound = reset(newRoot.bound)
		for i := 0; i < len(newRoot.items); i++ {
			newRoot.bound = stretchBoundingBox(newRoot.bound, newRoot.items[i])
		}

		rt.mRoot = newRoot
		return
	}

	newNode.parent = level.parent
	level.parent.items = append(level.parent.items, newNode)

	// I3: If OverflowTreatment was called and a split was
	// performed, propagate OverflowTreatment upwards
	// If necessary
	// return newNode
	if len(level.parent.items) > rt.maxChildItems {
		rt.overflowTreatment(level.parent, firstInsert)
	}
}

func (rt *Rtree[LeafType]) reinsert(node *RtreeNode) {
	var removedItems []BoundedItem

	nItems := len(node.items)
	var p int
	if float64(nItems)*REINSERT_P > 0 {
		// The experiments have
		// shown that p = 30% of M for leaf nodes as well as for nonleaf nodes yields the best performance
		p = int(float64(nItems) * REINSERT_P)
	} else {
		p = 1
	}

	assertt(nItems == rt.maxChildItems+1, "nItems must be equal to maxChildItems + 1")

	// (Reinsertion) RI1: For all M+1 entries of a node N, compute the distance
	//	between the centers of their rectangles and the center
	// of the bounding rectangle of N

	// RI2: Sort the entries in decreasing order of their distances
	// computed in RI1
	sortDecreasingBoundedItemsByDistanceFromCenter(node.bound, node.items[:len(node.items)-p])

	// RI3: Remove the first p entries from N and adjust the
	// bounding rectangle of N
	removedItems = node.items[p:]
	node.items = node.items[:p]

	// adjust the bounding rectangle of N
	node.bound = reset(node.bound)
	for i := 0; i < len(node.items); i++ {
		node.bound = stretchBoundingBox(node.bound, node.items[i])
	}

	// RI4: In the sort, defined in RI2, starting with the maximum
	// distance (= far reinsert) or minimum distance (= close
	// reinsert), invoke Insert to reinsert the entries
	for _, removedItem := range removedItems {
		rt.insertInternal(removedItem.(*RtreeLeaf[LeafType]), rt.mRoot, false)
	}
}

func (rt *Rtree[LeafType]) chooseSubtree(node *RtreeNode, bound RtreeBoundingBox) *RtreeNode {
	// Insert I4: Adjust all covering rectangles in the insertion path
	// such that they are minimum bounding boxes
	// enclosing their children rectangles

	// node.bound.stretch(bound)
	node.bound = stretch(node.bound, bound)

	var chosen *RtreeNode

	// CS2: If N 1s a leaf, return N
	if node.isLeafNode() {
		return node
	}

	// If the child pointers in N point to leaves (leaves = leaf node)
	if node.items[0].isLeafNode() {

		// If the childpointers in N point to leaves [determine
		// the minimum overlap cost],
		// choose the entry in N whose rectangle needs least
		// overlap enlargement to include the new data
		// rectangle Resolve ties by choosing the entry
		// whose rectangle needs least area enlargement

		minOverlapEnlargement := math.MaxFloat64
		idxEntryWithMinOverlapEnlargement := 0
		for i, item := range node.items {
			itembb := item.getBound()
			// bb := itembb.boundingBox(bound)
			bb := boundingBox(itembb, bound)

			// enlargement := item.getBound().overlap(bound)
			enlargement := overlap(item.getBound(), bound)

			if enlargement < minOverlapEnlargement || (enlargement == minOverlapEnlargement &&
				area(bb)-area(item.getBound()) < area(bb)-area(node.items[idxEntryWithMinOverlapEnlargement].getBound())) {
				minOverlapEnlargement = enlargement
				idxEntryWithMinOverlapEnlargement = i
			}
		}
		chosen = node.items[idxEntryWithMinOverlapEnlargement].(*RtreeNode)
		return rt.chooseSubtree(chosen, bound)
	}

	// (ChooseSubtree) CS2: if the childpointers in N do not point to leaves
	// [determine the minimum area cost],
	// choose the entry in N whose rectangle needs least
	// area enlargement to include the new data
	// rectangle Resolve ties by choosing the entry
	// with the rectangle of smallest area.

	minAreaEnlargement := math.MaxFloat64
	idxEntryWithMinAreaEnlargement := 0
	for i, item := range node.items {
		itembb := item.getBound()
		// bb := itembb.boundingBox(bound)
		bb := boundingBox(itembb, bound)

		// enlargement := bb.area() - item.getBound().area()
		enlargement := area(bb) - area(item.getBound())
		if enlargement < minAreaEnlargement ||
			(enlargement == minAreaEnlargement &&
				area(bb) < area(node.items[idxEntryWithMinAreaEnlargement].getBound())) {
			minAreaEnlargement = enlargement
			idxEntryWithMinAreaEnlargement = i
		}
	}

	chosen = node.items[idxEntryWithMinAreaEnlargement].(*RtreeNode)

	return rt.chooseSubtree(chosen, bound)
}

func (rt *Rtree[LeafType]) split(node *RtreeNode) *RtreeNode {
	newNode := &RtreeNode{}

	newNode.isLeaf = node.isLeaf

	nItems := len(node.items)
	distributionCount := nItems - 2*rt.minChildItems + 1
	minSplitMargin := math.MaxFloat64

	splitIndex := 0 // split index for the first group (m-1)+k

	firstGroup := RtreeBoundingBox{}  // first group [0,(m-1)+k) entries
	secondGroup := RtreeBoundingBox{} // second group [(m-1)+k,n) entries
	assertt(nItems == rt.maxChildItems+1, "nItems must be equal to maxChildItems + 1")
	assertt(distributionCount > 0, "distributionCount must be greater than 0")
	assertt(rt.minChildItems+distributionCount-1 <= nItems, "rt.minChildItems + distributionCount - 1 must be less than or equal to nItems")

	// the entries are first sorted by the lower
	// value, then sorted by the upper value of then rectangles For
	// each sort M-2m+2 distributions of the M+1 entries into two
	// groups are determined.

	// CSA1: For each axis
	// Sort the entries by the lower then by the upper
	// value of their rectangles and determine all
	// distributins as described above Compute S. the
	// sum of all margin-values of the different
	// distributions
	for axis := 0; axis < rt.dimensions; axis++ {
		margin := 0.0
		overlapVal := 0.0

		distribIndex := 0

		minArea := math.MaxFloat64
		minOverlap := math.MaxFloat64

		// ChooseSplitAxis (CSA1): Sort the items by the lower then by the upper
		// edge of their bounding box on this particular axis and
		// determine all distributions as described . Compute S. the
		// sum of all margin-values of the different
		// distributions

		// lower edge == 0 , upper edge == 1
		for edge := 0; edge < 2; edge++ {

			// Sort the entries by the lower then by the upper
			// value of their rectangles
			if edge == 0 {
				sortBoundedItemsByFirstEdge(axis, node.items)
			} else {
				sortBoundedItemsBySecondEdge(axis, node.items)
			}

			//  where the k-th distribution (k = 1,....,(M-2m+2))
			// 0-indexed jadi k=0,....,(M-2m+1)
			for k := 0; k < distributionCount; k++ {
				// k  = distribution value.
				bbArea := 0.0

				// calculate bounding box of the first group
				// firstGroup.reset()
				firstGroup = reset(firstGroup)
				for i := 0; i < (rt.minChildItems-1)+k; i++ { // (m-1)+k entries
					// firstGroup.stretch(node.items[i].getBound())
					firstGroup = stretch(firstGroup, node.items[i].getBound())
				}

				// calculate bounding box of the second group
				// secondGroup.reset()
				secondGroup = reset(secondGroup)
				for i := (rt.minChildItems - 1) + k; i < len(node.items); i++ {
					// secondGroup.stretch(node.items[i].getBound())
					secondGroup = stretch(secondGroup, node.items[i].getBound())
				}

				// // margin  = area[bb(first group)] +area[bb(second group)]
				// // Compute S. the sum of all margin-values of the different  distributions.
				// margin += firstGroup.edgeDeltas() + secondGroup.edgeDeltas()
				// // area = margin[bb(first group)] + margin[bb(second group)]
				// area += firstGroup.area() + secondGroup.area()
				// // overlap = area[bb(first group) n bb(second group)]. n = overlap
				// overlap = firstGroup.overlap(&secondGroup)

				// margin  = area[bb(first group)] +area[bb(second group)]
				// Compute S. the sum of all margin-values of the different  distributions.
				margin += edgeDeltas(firstGroup) + edgeDeltas(secondGroup)
				// area = margin[bb(first group)] + margin[bb(second group)]
				bbArea += area(firstGroup) + area(secondGroup)
				// overlap = area[bb(first group) n bb(second group)]. n = overlap
				overlap(firstGroup, secondGroup)

				//(ChooseSplitIndex) CSI1: Along the chosen split axis, choose the distribution with the minimum overlap-value
				// Resolve ties by choosing the distribution with
				// minimum area-value
				if overlapVal < minOverlap || overlapVal == minOverlap && bbArea < minArea {
					distribIndex = (rt.minChildItems - 1) + k //(m-1)+k // k = distribution value
					minOverlap = overlapVal
					minArea = bbArea
				}
			}
		}

		// CSA2: Choose the axis with the minimum S as split axis. S =  the sum of all margin-values of the different  distributions.
		if margin < minSplitMargin {
			minSplitMargin = margin
			splitIndex = distribIndex
		}

	}

	// S3: Distribute the items into two groups

	// distribute the end of the array node.items to the newNode. and erase them from the original node.
	newNode.items = make([]BoundedItem, 0, len(node.items)-splitIndex)
	// insert elements [(m-1)+k,len(node.items)) of the array node.items to the newNode.items
	for i := splitIndex; i < len(node.items); i++ {
		newNode.items = append(newNode.items, node.items[i])
	}
	node.items = node.items[:splitIndex] // erase the end [(m-1)+k,len(node.items)) of the array node.items

	// adjust the bounding box.
	// node.bound.reset()
	node.bound = reset(node.bound)
	for i := 0; i < len(node.items); i++ {
		// node.bound.stretch(node.items[i].getBound())
		node.bound = stretch(node.bound, node.items[i].getBound())
	}

	// adjust the bounding box.
	newNode.bound = NewRtreeBoundingBox(rt.dimensions, make([]float64, rt.dimensions), make([]float64, rt.dimensions))
	// newNode.bound.reset()
	newNode.bound = reset(newNode.bound)
	for i := 0; i < len(newNode.items); i++ {
		// newNode.bound.stretch(newNode.items[i].getBound())
		newNode.bound = stretch(newNode.bound, newNode.items[i].getBound())
	}

	return newNode
}

func (rt *Rtree[LeafType]) Search(bound RtreeBoundingBox) []RtreeLeaf[LeafType] {
	results := []RtreeLeaf[LeafType]{}
	return rt.search(rt.mRoot, bound, results)
}

func (rt *Rtree[LeafType]) search(node *RtreeNode, bound RtreeBoundingBox,
	results []RtreeLeaf[LeafType]) []RtreeLeaf[LeafType] {
	for _, e := range node.items {

		if !overlaps(e.getBound(), bound) {
			continue
		}

		if !e.isLeafNode() {
			// S1. [Search subtrees.] If T is not a leaf,
			// check each entry E to determine
			// whether E.I overlaps S. For all overlapping entries, invoke Search on the tree
			// whose root node is pointed to by E.p
			rt.search(e.(*RtreeNode), bound, results)
			continue
		}

		for _, leaf := range e.(*RtreeNode).items {
			if overlaps(leaf.getBound(), bound) {
				// S2. [Search leaf node.] If T is a leaf, check
				// all entries E to determine whether E.I
				// overlaps S. If so, E is a qualifying
				// record
				results = append(results, *leaf.(*RtreeLeaf[LeafType]))
			}
		}
	}
	return results
}

type Point struct {
	Lat float64
	Lon float64
}

// minDist computes the square of the distance from a point to a rectangle. If the point is contained in the rectangle then the distance is zero.
func (p Point) minDist(r RtreeBoundingBox) float64 {

	sum := 0.0

	// Edges[0] = {minLat, maxLat}
	// Edges[1] = {minLon, maxLon}
	if p.Lat < r.Edges[0][0] {
		sum += (p.Lat - r.Edges[0][0]) * (p.Lat - r.Edges[0][0])
	} else if p.Lat > r.Edges[0][1] {
		sum += (p.Lat - r.Edges[0][1]) * (p.Lat - r.Edges[0][1])
	}

	if p.Lon < r.Edges[1][0] {
		sum += (p.Lon - r.Edges[1][0]) * (p.Lon - r.Edges[1][0])
	} else if p.Lon > r.Edges[1][1] {
		sum += (p.Lon - r.Edges[1][1]) * (p.Lon - r.Edges[1][1])
	}

	return sum
}

func (rt *Rtree[LeafType]) KNearestNeighbors(k int, p Point) []RtreeLeaf[OSMObject] {
	nearestsListsPQ := NewMaxPriorityQueue[RtreeLeaf[OSMObject], float64]()

	root := rt.mRoot

	rt.kNearestNeighbors(k, p, root, nearestsListsPQ)
	nearestLists := make([]RtreeLeaf[OSMObject], 0, k)
	for nearestsListsPQ.Len() > 0 {
		nearestLists = append(nearestLists, heap.Pop(nearestsListsPQ).(*PriorityQueueNode[RtreeLeaf[OSMObject], float64]).item)
	}
	reverseG(nearestLists)
	return nearestLists
}

func reverseG[T any](arr []T) (result []T) {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

type OSMObject struct {
	id  int
	lat float64
	lon float64
}

func insertToNearestLists(nearestLists *priorityQueue[RtreeLeaf[OSMObject], float64], obj RtreeLeaf[OSMObject], dist float64, k int) {
	if nearestLists.Len() < k {
		heap.Push(nearestLists, &PriorityQueueNode[RtreeLeaf[OSMObject], float64]{rank: dist, item: obj})
	} else if dist < (*nearestLists)[0].rank {
		heap.Pop(nearestLists)
		heap.Push(nearestLists, &PriorityQueueNode[RtreeLeaf[OSMObject], float64]{rank: dist, item: obj})
	}
}

type activeBranch struct {
	entry BoundedItem
	Dist  float64
}

// https://dl.acm.org/doi/pdf/10.1145/320248.320255 (Fig. 7. k-nearest neighbor algorithm.)
// TODO: add pruning? idk
func (rt *Rtree[LeafType]) kNearestNeighbors(k int, p Point, n *RtreeNode,
	nearestLists *priorityQueue[RtreeLeaf[OSMObject], float64]) {
	var nearestListMaxDist float64 = math.Inf(1)

	pq := *nearestLists

	if nearestLists.Len() < k && nearestLists.Len() > 0 {
		nearestListMaxDist = pq[0].rank
	}

	if n.isLeaf {
		for _, item := range n.items {
			dist := geo.HaversineDistance(p.Lat, p.Lon,
				item.(*RtreeLeaf[OSMObject]).leaf.lat, item.(*RtreeLeaf[OSMObject]).leaf.lon)
			if dist < nearestListMaxDist {
				insertToNearestLists(nearestLists, *item.(*RtreeLeaf[OSMObject]), dist, k)
			}
		}
	} else {
		activeBranchLists := make([]activeBranch, len(n.items))
		for i, e := range n.items {
			activeBranchLists[i] = activeBranch{e, p.minDist(e.getBound())}
		}

		// sort entries based on the distance from point p to the minimum bounding rectangle of each entry.
		sort.Slice(activeBranchLists, func(i, j int) bool {
			return activeBranchLists[i].Dist < activeBranchLists[j].Dist
		})

		for _, e := range activeBranchLists {

			if e.Dist < nearestListMaxDist {
				// recursion to children node entry e.
				rt.kNearestNeighbors(k, p, e.entry.(*RtreeNode), nearestLists)
			} else {
				// if the distance from the point p to the minimum bounding rectangle of entry e is greater than the maximum distance in the nearestListsPQ
				// then we can stop the recursion to the children node entry e.
				break 
			}
		}
	}
}

