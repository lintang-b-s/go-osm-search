package pkg

type heapMergeItem struct {
	Metadata  []int
	TermID   int
	Postings []int
}

func NewHeapMergeItem(termID int, metadata []int, postings []int) heapMergeItem {
	return heapMergeItem{
		Metadata: metadata,
		TermID:   termID,
		Postings: postings,
	}
}

type Item interface {
	int | heapMergeItem
}

type Rank interface {
	int | float64
}

type priorityQueueNode[T Item, G Rank] struct {
	rank  G
	index int
	item  T
}

func NewPriorityQueueNode[T Item, G Rank](rank G, item T) *priorityQueueNode[T, G] {
	return &priorityQueueNode[T, G]{rank: rank, item: item}
}

type priorityQueue[T Item, G Rank] []*priorityQueueNode[T, G]

func NewPriorityQueue[T Item, G Rank]() *priorityQueue[T, G] {
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
	no := x.(*priorityQueueNode[Item, Rank])
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
