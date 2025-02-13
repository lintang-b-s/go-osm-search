package datastructure

type HeapMergeItem struct {
	Metadata []int
	TermID   int
	Postings []int
}

func NewHeapMergeItem(termID int, metadata []int, postings []int) HeapMergeItem {
	return HeapMergeItem{
		Metadata: metadata,
		TermID:   termID,
		Postings: postings,
	}
}

type Item interface {
	int | HeapMergeItem | interface{}
}

type Rank interface {
	int | float64
}

type PriorityQueueNode[T Item, G Rank] struct {
	rank  G
	index int
	item  T
}

func (pq *PriorityQueueNode[Item, Rank]) GetRank() Rank {
	return pq.rank
}

func (pq *PriorityQueueNode[Item, Rank]) GetItem() Item {
	return pq.item
}

func (pq *PriorityQueueNode[Item, Rank]) GetIndex() int {
	return pq.index
}

func NewPriorityQueueNode[T Item, G Rank](rank G, item T) *PriorityQueueNode[T, G] {
	return &PriorityQueueNode[T, G]{rank: rank, item: item}
}

type priorityQueue[T Item, G Rank] []*PriorityQueueNode[T, G]

func NewMaxPriorityQueue[T Item, G Rank]() *priorityQueue[T, G] {
	return &priorityQueue[T, G]{}
}

func (pq priorityQueue[Item, Rank]) Len() int {
	return len(pq)
}

func (pq priorityQueue[Item, Rank]) Less(i, j int) bool {
	return pq[i].rank > pq[j].rank
}

func (pq priorityQueue[Item, Rank]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue[Item, Rank]) Push(x interface{}) {
	n := len(*pq)
	no := x.(*PriorityQueueNode[Item, Rank])
	no.index = n
	*pq = append(*pq, no)
}

func (pq *priorityQueue[Item, Rank]) Pop() interface{} {
	old := *pq
	n := len(old)
	no := old[n-1]
	no.index = -1
	*pq = old[0 : n-1]
	return no
}

// Min Priority queue
type minPriorityQueue[T Item, G Rank] []*PriorityQueueNode[T, G]

func NewMinPriorityQueue[T Item, G Rank]() *minPriorityQueue[T, G] {
	return &minPriorityQueue[T, G]{}
}

func (pq minPriorityQueue[Item, Rank]) Len() int {
	return len(pq)
}

func (pq minPriorityQueue[Item, Rank]) Less(i, j int) bool {
	return pq[i].rank < pq[j].rank
}

func (pq minPriorityQueue[Item, Rank]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *minPriorityQueue[Item, Rank]) Push(x interface{}) {
	n := len(*pq)
	no := x.(*PriorityQueueNode[Item, Rank])
	no.index = n
	*pq = append(*pq, no)
}

func (pq *minPriorityQueue[Item, Rank]) Pop() interface{} {
	old := *pq
	n := len(old)
	no := old[n-1]
	no.index = -1
	*pq = old[0 : n-1]
	return no
}

type PriorityQueueNodeRtree struct {
	rank  float64
	index int
	item  BoundedItem
}

func NewPriorityQueueNodeRtree(rank float64, item BoundedItem) *PriorityQueueNodeRtree {
	return &PriorityQueueNodeRtree{rank: rank, item: item}
}

// Min Priority queue
type minPriorityQueueRtree []*PriorityQueueNodeRtree

func NewMinPriorityQueueRtree() minPriorityQueueRtree {
	return minPriorityQueueRtree{}
}

func (pq minPriorityQueueRtree) Len() int {
	return len(pq)
}

func (pq minPriorityQueueRtree) Less(i, j int) bool {
	return pq[i].rank < pq[j].rank
}

func (pq minPriorityQueueRtree) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *minPriorityQueueRtree) Push(x interface{}) {
	n := len(*pq)
	no := x.(*PriorityQueueNodeRtree)
	no.index = n
	*pq = append(*pq, no)
}

func (pq *minPriorityQueueRtree) Pop() interface{} {
	old := *pq
	n := len(old)
	no := old[n-1]
	no.index = -1
	*pq = old[0 : n-1]
	return no
}
