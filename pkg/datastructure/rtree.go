package datastructure

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"sort"
)

// cuma rewrite dari implementatasi c++ ini: https://github.com/virtuald/r-star-tree/
// + add N-nearest neighbors & Search.
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

func sortBoundedItemsByFirstEdge(mAxis int, items []*RtreeNode) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].getBound().Edges[mAxis][0] < items[j].getBound().Edges[mAxis][0]
	})
}

func sortBoundedItemsBySecondEdge(mAxis int, items []*RtreeNode) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].getBound().Edges[mAxis][1] < items[j].getBound().Edges[mAxis][1]
	})
}

func sortIncreasingBoundedItemsByDistanceFromCenter(mCenter RtreeBoundingBox, items []*RtreeNode) {
	sort.Slice(items, func(i, j int) bool {
		return mCenter.distanceFromCenter(items[i].getBound()) < mCenter.distanceFromCenter(items[j].getBound())
	})
}

func sortDecreasingBoundedItemsByDistanceFromCenter(mCenter RtreeBoundingBox, items []*RtreeNode) {
	sort.Slice(items, func(i, j int) bool {
		return mCenter.distanceFromCenter(items[i].getBound()) > mCenter.distanceFromCenter(items[j].getBound())
	})
}

func sortBoundedItemsByAreaEnlargement(bbarea float64, items []*RtreeNode) {
	sort.Slice(items, func(i, j int) bool {
		return bbarea-area(items[i].getBound()) < bbarea-area(items[j].getBound())

	})
}

func sortIncreasingBoundedItemsByOverlapEnlargement(items []*RtreeNode, center RtreeBoundingBox) {
	sort.Slice(items, func(i, j int) bool {

		return overlap(items[i].getBound(), center) < overlap(items[j].getBound(), center)
		// return items[i].getBound().overlap(center) < items[j].getBound().overlap(center)
	})
}

func sortDecreasingBoundedItemsByOverlapEnlargement(items []*RtreeNode, center RtreeBoundingBox) {
	sort.Slice(items, func(i, j int) bool {
		return overlap(items[i].getBound(), center) > overlap(items[j].getBound(), center)

		// return items[i].getBound().overlap(center) > items[j].getBound().overlap(center)
	})
}

type BoundedItem interface {
	getBound() RtreeBoundingBox
	isLeafNode() bool
}

// rtree node. can be either a leaf node or a internal node or leafData.
type RtreeNode struct {
	// entries. can be either a leaf node or a  internal node.
	// leafNode has items in the form of a list of RtreeLeaf
	Items  []*RtreeNode
	Parent *RtreeNode

	Bound RtreeBoundingBox
	// isLeaf. true if  this node is a leafNode.
	IsLeaf bool

	Leaf OSMObject // if this node is a leafData
}

// isLeaf. true if this node is a leafNode.
func (node *RtreeNode) isLeafNode() bool {
	return node.IsLeaf
}

func (node *RtreeNode) getBound() RtreeBoundingBox {
	return node.Bound
}

type Rtree struct {
	// semuanya exported biar bisa diencode gobencoder
	Root          *RtreeNode
	Size          int
	MinChildItems int
	MaxChildItems int
	Dimensions    int
	Height        int
}

func NewRtree(minChildItems, maxChildItems, dimensions int) *Rtree {
	return &Rtree{
		Root:          nil,
		Size:          0,
		Height:        0,
		MinChildItems: minChildItems,
		MaxChildItems: maxChildItems,
		Dimensions:    dimensions,
	}
}

func (rt *Rtree) InsertLeaf(bound RtreeBoundingBox, leaf OSMObject) {

	newLeaf := &RtreeNode{}
	newLeaf.Bound = bound
	newLeaf.Leaf = leaf

	if rt.Root == nil {
		rt.Root = &RtreeNode{}
		rt.Root.IsLeaf = true // set root as leaf node

		rt.Root.Items = make([]*RtreeNode, 0, rt.MinChildItems)

		rt.Root.Items = append(rt.Root.Items, newLeaf) // add new leaf data to the root
		rt.Root.Bound = bound
	} else {
		rt.insertInternal(newLeaf, rt.Root, true)
	}
	rt.Size++

}

func (rt *Rtree) insertInternal(leaf *RtreeNode, root *RtreeNode, firstInsert bool) *RtreeNode {

	// I1: Invoke ChooseSubtree. with the level as a parameter,
	// to find an appropriate node N, in which to place the new entry E

	leafNode := rt.chooseSubtree(root, leaf.Bound)

	// if this node is a leafNode then add the leaf data to the  leafNode
	//I2: accommodate E in N.
	leafNode.Items = append(leafNode.Items, leaf)

	// I2: if node N has M+1 entries. invoke OverflowTreatment
	// with the level of N as a parameter [for reinsertion or split]
	if len(leafNode.Items) > rt.MaxChildItems {
		rt.overflowTreatment(leafNode, firstInsert)

	}

	return nil
}

func (rt *Rtree) overflowTreatment(level *RtreeNode, firstInsert bool) {
	// OT1: If the level is not the root level and this is the first
	// call of OverflowTreatment in the given level
	// during the Insertion of one data rectangle, then
	// invoke Reinsert
	if level != rt.Root && firstInsert {
		rt.reinsert(level)
		return
	}

	//else invoke Split
	newNode := rt.split(level)

	// I3:  If OverflowTreatment caused a split of the root, create a
	// new root whose children are the two resulting nodes (old root & newNode).
	if level == rt.Root {
		// benar
		newRoot := &RtreeNode{}
		newRoot.IsLeaf = false

		newRoot.Items = make([]*RtreeNode, 0, rt.MinChildItems)
		newRoot.Items = append(newRoot.Items, rt.Root)
		newRoot.Items = append(newRoot.Items, newNode)
		rt.Root.Parent = newRoot
		newNode.Parent = newRoot

		rt.Height++

		// I4: Adjust all covering rectangles in the insertion path
		// such that they are minimum bounding boxes
		// enclosing their children rectangles
		newRoot.Bound = NewRtreeBoundingBox(rt.Dimensions, make([]float64, rt.Dimensions), make([]float64, rt.Dimensions))

		newRoot.Bound = reset(newRoot.Bound)
		for i := 0; i < len(newRoot.Items); i++ {
			newRoot.Bound = stretchBoundingBox(newRoot.Bound, newRoot.Items[i])
		}

		rt.Root = newRoot
		return
	}

	newNode.Parent = level.Parent
	level.Parent.Items = append(level.Parent.Items, newNode)

	level.Parent.Bound = reset(level.Parent.Bound)
	for i := 0; i < len(level.Parent.Items); i++ {
		level.Parent.Bound = stretch(level.Parent.Bound, level.Parent.Items[i].getBound())
	}

	// I3: If OverflowTreatment was called and a split was
	// performed, propagate OverflowTreatment upwards
	// If necessary
	// return newNode
	if len(level.Parent.Items) > rt.MaxChildItems {
		rt.overflowTreatment(level.Parent, firstInsert)
	}
}

func (rt *Rtree) reinsert(node *RtreeNode) {
	var removedItems []*RtreeNode

	nItems := len(node.Items)
	var p int
	if float64(nItems)*REINSERT_P > 0 {
		// The experiments have
		// shown that p = 30% of M for leaf nodes as well as for nonleaf nodes yields the best performance
		p = int(float64(nItems) * REINSERT_P)
	} else {
		p = 1
	}

	assertt(nItems == rt.MaxChildItems+1, "nItems must be equal to maxChildItems + 1")

	// (Reinsertion) RI1: For all M+1 entries of a node N, compute the distance
	//	between the centers of their rectangles and the center
	// of the bounding rectangle of N

	// RI2: Sort the entries in decreasing order of their distances
	// computed in RI1
	sortDecreasingBoundedItemsByDistanceFromCenter(node.Bound, node.Items[:len(node.Items)-p])

	// RI3: Remove the first p entries from N and adjust the
	// bounding rectangle of N
	removedItems = node.Items[p:]
	node.Items = node.Items[:p]

	// adjust the bounding rectangle of N
	node.Bound = reset(node.Bound)
	for i := 0; i < len(node.Items); i++ {
		node.Bound = stretchBoundingBox(node.Bound, node.Items[i])
	}

	// RI4: In the sort, defined in RI2, starting with the maximum
	// distance (= far reinsert) or minimum distance (= close
	// reinsert), invoke Insert to reinsert the entries
	for _, removedItem := range removedItems {
		rt.insertInternal(removedItem, rt.Root, false)
	}
}

func (rt *Rtree) chooseSubtree(node *RtreeNode, bound RtreeBoundingBox) *RtreeNode {
	// Insert I4: Adjust all covering rectangles in the insertion path
	// such that they are minimum bounding boxes
	// enclosing their children rectangles

	node.Bound = stretch(node.Bound, bound)

	var chosen *RtreeNode

	// CS2: If N 1s a leaf, return N
	if node.isLeafNode() {
		return node
	}

	// If the child pointers in N point to leaves (leaves = leaf node)
	if node.Items[0].isLeafNode() {

		// If the childpointers in N point to leaves [determine
		// the minimum overlap cost],
		// choose the entry in N whose rectangle needs least
		// overlap enlargement to include the new data
		// rectangle Resolve ties by choosing the entry
		// whose rectangle needs least area enlargement

		minOverlapEnlargement := math.MaxFloat64
		idxEntryWithMinOverlapEnlargement := 0
		for i, item := range node.Items {
			itembb := item.getBound()
			// bb := itembb.BoundingBox(bound)
			bb := boundingBox(itembb, bound)

			// enlargement := item.getBound().overlap(bound)
			enlargement := overlap(item.getBound(), bound)

			if enlargement < minOverlapEnlargement || (enlargement == minOverlapEnlargement &&
				area(bb)-area(item.getBound()) < area(bb)-area(node.Items[idxEntryWithMinOverlapEnlargement].getBound())) {
				minOverlapEnlargement = enlargement
				idxEntryWithMinOverlapEnlargement = i
			}
		}
		chosen = node.Items[idxEntryWithMinOverlapEnlargement]
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
	for i, item := range node.Items {
		itembb := item.getBound()
		// bb := itembb.BoundingBox(bound)
		bb := boundingBox(itembb, bound)

		// enlargement := bb.area() - item.getBound().area()
		enlargement := area(bb) - area(item.getBound())
		if enlargement < minAreaEnlargement ||
			(enlargement == minAreaEnlargement &&
				area(bb) < area(node.Items[idxEntryWithMinAreaEnlargement].getBound())) {
			minAreaEnlargement = enlargement
			idxEntryWithMinAreaEnlargement = i
		}
	}

	chosen = node.Items[idxEntryWithMinAreaEnlargement]

	return rt.chooseSubtree(chosen, bound)
}

func (rt *Rtree) split(node *RtreeNode) *RtreeNode {
	newNode := &RtreeNode{}

	newNode.IsLeaf = node.IsLeaf

	nItems := len(node.Items)
	distributionCount := nItems - 2*rt.MinChildItems + 1
	minSplitMargin := math.MaxFloat64

	splitIndex := 0 // split index for the first group (m-1)+k

	firstGroup := RtreeBoundingBox{}  // first group [0,(m-1)+k) entries
	secondGroup := RtreeBoundingBox{} // second group [(m-1)+k,n) entries
	assertt(nItems == rt.MaxChildItems+1, "nItems must be equal to maxChildItems + 1")
	assertt(distributionCount > 0, "distributionCount must be greater than 0")
	assertt(rt.MinChildItems+distributionCount-1 <= nItems, "rt.MinChildItems + distributionCount - 1 must be less than or equal to nItems")

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
	for axis := 0; axis < rt.Dimensions; axis++ {
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
				sortBoundedItemsByFirstEdge(axis, node.Items)
			} else {
				sortBoundedItemsBySecondEdge(axis, node.Items)
			}

			//  where the k-th distribution (k = 1,....,(M-2m+2))
			// 0-indexed jadi k=0,....,(M-2m+1)
			for k := 0; k < distributionCount; k++ {
				// k  = distribution value.
				bbArea := 0.0

				// calculate bounding box of the first group

				firstGroup = reset(firstGroup)
				for i := 0; i < (rt.MinChildItems-1)+k; i++ { // (m-1)+k entries

					firstGroup = stretch(firstGroup, node.Items[i].getBound())
				}

				// calculate bounding box of the second group

				secondGroup = reset(secondGroup)
				for i := (rt.MinChildItems - 1) + k; i < len(node.Items); i++ {

					secondGroup = stretch(secondGroup, node.Items[i].getBound())
				}

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
					distribIndex = (rt.MinChildItems - 1) + k //(m-1)+k // k = distribution value
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

	// distribute the end of the array node.Items to the newNode. and erase them from the original node.
	newNode.Items = make([]*RtreeNode, 0, len(node.Items)-splitIndex)
	// insert elements [(m-1)+k,len(node.Items)) of the array node.Items to the newNode.Items
	for i := splitIndex; i < len(node.Items); i++ {
		newNode.Items = append(newNode.Items, node.Items[i])
	}
	node.Items = node.Items[:splitIndex] // erase the end [(m-1)+k,len(node.Items)) of the array node.Items

	// adjust the bounding box.

	node.Bound = reset(node.Bound)
	for i := 0; i < len(node.Items); i++ {
		node.Bound = stretch(node.Bound, node.Items[i].getBound())
	}

	// adjust the bounding box.
	newNode.Bound = NewRtreeBoundingBox(rt.Dimensions, make([]float64, rt.Dimensions), make([]float64, rt.Dimensions))

	newNode.Bound = reset(newNode.Bound)
	for i := 0; i < len(newNode.Items); i++ {
		newNode.Bound = stretch(newNode.Bound, newNode.Items[i].getBound())
	}

	return newNode
}

func (rt *Rtree) Search(bound RtreeBoundingBox) []RtreeNode {
	results := []RtreeNode{}
	return rt.search(rt.Root, bound, results)
}

func (rt *Rtree) search(node *RtreeNode, bound RtreeBoundingBox,
	results []RtreeNode) []RtreeNode {
	for _, e := range node.Items {

		if !overlaps(e.getBound(), bound) {
			continue
		}

		if !node.isLeafNode() {
			// S1. [Search subtrees.] If T is not a leaf,
			// check each entry E to determine
			// whether E.I overlaps S. For all overlapping entries, invoke Search on the tree
			// whose root node is pointed to by E.p
			results = rt.search(e, bound, results)
			continue
		}

		if overlaps(e.getBound(), bound) {
			// S2. [Search leaf node.] If T is a leaf, check
			// all entries E to determine whether E.I
			// overlaps S. If so, E is a qualifying
			// record
			results = append(results, *e)

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

	// Edges[0] = {minLat, maxLat}
	// Edges[1] = {minLon, maxLon}
	sum := 0.0
	rLat, rLon := 0.0, 0.0
	if p.Lat < r.Edges[0][0] {
		rLat = r.Edges[0][0]
	} else if p.Lat > r.Edges[0][1] {
		rLat = r.Edges[0][1]
	} else {
		rLat = p.Lat
	}

	if p.Lon < r.Edges[1][0] {
		rLon = r.Edges[1][0]
	} else if p.Lon > r.Edges[1][1] {
		rLon = r.Edges[1][1]
	} else {
		rLon = p.Lon
	}

	sum = haversineDistance(p.Lat, p.Lon, rLat, rLon)

	return sum
}

// https://infolab.usc.edu/csci599/Fall2007/papers/a-1.pdf. cmiiw
func (p Point) minMaxDist(r RtreeBoundingBox) float64 {

	rmk := 0.0
	rMi := 0.0

	// lat dimension
	if p.Lat <= (r.Edges[0][0]+r.Edges[0][1])/2.0 {
		rmk = r.Edges[0][0]
	} else {
		rmk = r.Edges[0][1]
	}

	minMaxDistLatDim := math.Pow(p.Lat-rmk, 2)

	if p.Lon >= (r.Edges[1][0]+r.Edges[1][1])/2.0 {
		rMi = r.Edges[1][0]
	} else {
		rMi = r.Edges[1][1]
	}
	minMaxDistLatDim += math.Pow(p.Lon-rMi, 2)

	// lon dimension
	if p.Lon <= (r.Edges[1][0]+r.Edges[1][1])/2.0 {
		rmk = r.Edges[1][0]
	} else {
		rmk = r.Edges[1][1]
	}

	minMaxDistLonDim := math.Pow(p.Lon-rmk, 2)

	if p.Lat >= (r.Edges[0][0]+r.Edges[0][1])/2.0 {
		rMi = r.Edges[0][0]
	} else {
		rMi = r.Edges[0][1]
	}
	minMaxDistLonDim += math.Pow(p.Lat-rMi, 2)

	if minMaxDistLatDim < minMaxDistLonDim {
		return minMaxDistLatDim
	}
	return minMaxDistLonDim
}

type OSMObject struct {
	ID  int
	Lat float64
	Lon float64
}

func (o *OSMObject) GetBound() RtreeBoundingBox {
	return NewRtreeBoundingBox(2, []float64{o.Lat - 0.0001, o.Lon - 0.0001}, []float64{o.Lat + 0.0001, o.Lon + 0.0001})
}

type activeBranch struct {
	entry BoundedItem
	Dist  float64
}

func (rt *Rtree) FastNNearestNeighbors(k int, p Point) []RtreeNode {
	nearestsLists := make([]RtreeNode, 0, k)

	root := rt.Root

	dists := make([]float64, 0, k)

	nearestsLists, _ = rt.fastNNearestNeighbors(k, p, root, nearestsLists, dists)

	return nearestsLists
}

func fastInsertToNearestLists(nearestLists []RtreeNode, obj RtreeNode, dist float64, k int,
	dists []float64) ([]RtreeNode, []float64) {
	idx := sort.SearchFloat64s(dists, dist)
	for idx < len(nearestLists) && dist >= dists[idx] {
		idx++
	}

	if idx >= k {
		return nearestLists, dists
	}

	if len(nearestLists) < k {
		dists = append(dists, 0)
		nearestLists = append(nearestLists, RtreeNode{})
	}

	copy(dists, dists[:idx])
	copy(dists[idx+1:], dists[idx:len(dists)-1])
	dists[idx] = dist

	copy(nearestLists, nearestLists[:idx])
	copy(nearestLists[idx+1:], nearestLists[idx:len(nearestLists)-1])
	nearestLists[idx] = obj

	return nearestLists, dists
}

func (rt *Rtree) fastNNearestNeighbors(k int, p Point, n *RtreeNode,
	nearestLists []RtreeNode, nNearestDists []float64) ([]RtreeNode, []float64) {

	var nearestDist float64 = math.Inf(1)
	if len(nearestLists) > 0 {
		nearestDist = nNearestDists[0]
	}

	if n.IsLeaf {

		for _, item := range n.Items {
			dist := haversineDistance(p.Lat, p.Lon, item.Leaf.Lat, item.Leaf.Lon)

			if dist < nearestDist {
				nearestLists, nNearestDists = fastInsertToNearestLists(nearestLists, *item, dist, k, nNearestDists)
			}
		}
	} else {
		dists := make([]float64, 0, len(n.Items))
		for _, e := range n.Items {
			dists = append(dists, p.minDist(e.getBound()))
		}

		entries := make([]*RtreeNode, len(n.Items))
		copy(entries, n.Items)
		sort.Sort(activeBranchSlice{entries, dists})

		var cutBranchIdx int
		if len(nNearestDists) >= k {
			for i := 0; i < len(entries); i++ {
				if dists[i] > nNearestDists[len(nNearestDists)-1] {
					cutBranchIdx = i
				}
			}
			entries = entries[:cutBranchIdx]

		}

		for i := 0; i < len(entries); i++ {
			// recursion to children node entry e.
			nearestLists, nNearestDists = rt.fastNNearestNeighbors(k, p, entries[i], nearestLists, nNearestDists)
		}
	}
	return nearestLists, nNearestDists
}

func (rt *Rtree) ImprovedNearestNeighbor(p Point) RtreeNode {

	nearest := RtreeNode{}

	nnDistTemp := math.Inf(1)
	root := rt.Root

	nearest, _ = rt.nearestNeighbor(p, root, nearest, nnDistTemp)
	return nearest
}

type activeBranchSlice struct {
	entries []*RtreeNode
	dists   []float64
}

func (s activeBranchSlice) Len() int { return len(s.entries) }

func (s activeBranchSlice) Swap(i, j int) {
	s.entries[i], s.entries[j] = s.entries[j], s.entries[i]
	s.dists[i], s.dists[j] = s.dists[j], s.dists[i]
}

func (s activeBranchSlice) Less(i, j int) bool {
	return s.dists[i] < s.dists[j]
}

func (rt *Rtree) nearestNeighbor(p Point, n *RtreeNode,
	nearest RtreeNode, nnDistTemp float64) (RtreeNode, float64) {

	if n.IsLeaf {
		for _, item := range n.Items {

			dist := haversineDistance(p.Lat, p.Lon, item.Leaf.Lat, item.Leaf.Lon)

			if dist < nnDistTemp {
				nnDistTemp = dist
				nearest = *item
			}
		}
	} else {
		minMaxDistM := math.Inf(1)
		for _, e := range n.Items {
			minMaxDistM = math.Min(minMaxDistM, p.minMaxDist(e.getBound()))
		}

		last := len(n.Items)
		for i := 0; i < last; i++ {

			//an MBR M with MINDIST(P,M) greater than the
			//MINMAXDIST(P,M’) of another MBR M’ is discarded because it cannot contain the NN
			if p.minDist(n.Items[i].getBound()) <= minMaxDistM {
				nearest, nnDistTemp = rt.nearestNeighbor(p, n.Items[i], nearest, nnDistTemp)
				for j := i + 1; j < last; j++ {
					// upward pruning
					if p.minDist(n.Items[j].getBound()) > nnDistTemp {
						//every MBR M with MINDIST(P,M) greater than
						// the actual distance from P to a given object O is
						// discarded because it cannot enclose an object nearer
						// than O (theorem 1). We use this in upward pruning.
						last = j
					}
				}
			}

		}
	}
	return nearest, nnDistTemp
}

func SerializeRtreeData(workingDir string, outputDir string, items []OSMObject) error {

	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(items)
	if err != nil {
		return err
	}

	var rtreeFile *os.File
	if workingDir != "/" {
		rtreeFile, err = os.OpenFile(workingDir+"/"+outputDir+"/"+"rtree.dat", os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	} else {
		rtreeFile, err = os.OpenFile(outputDir+"/"+"rtree.dat", os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
	}
	_, err = rtreeFile.Write(buf.Bytes())

	return err
}

func (rt *Rtree) Deserialize(workingDir string, outputDir string) error {

	var rtreeFile *os.File
	var err error
	if workingDir != "/" {
		rtreeFile, err = os.Open(workingDir + "/" + outputDir + "/" + "rtree.dat")
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
	} else {
		rtreeFile, err = os.Open(outputDir + "/" + "rtree.dat")
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
	}

	stat, err := os.Stat(rtreeFile.Name())
	if err != nil {
		return fmt.Errorf("error when getting metadata file stat: %w", err)
	}

	buf := make([]byte, stat.Size()*2)

	_, err = rtreeFile.Read(buf)
	if err != nil {
		return err
	}

	gobDec := gob.NewDecoder(bytes.NewBuffer(buf))

	items := []OSMObject{}
	err = gobDec.Decode(&items)
	if err != nil {
		return err
	}

	for _, item := range items {
		rt.InsertLeaf(item.GetBound(), item)
	}

	return nil
}
