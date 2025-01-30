package index

import (
	"container/heap"
	"fmt"
	"iter"
	"osm-search/pkg/datastructure"
)

type heapMergeOutput struct {
	TermID   int
	Postings []int
}

func NewHeapMergeOutput(termID int, postings []int) heapMergeOutput {
	return heapMergeOutput{
		TermID:   termID,
		Postings: postings,
	}
}

type MergeKArrayIterator struct {
	err               error
	indexes           []*InvertedIndex
	nextIndexIterator []func() (IndexIteratorItem, error, bool)
}

func NewMergeKArrayIterator(indexes []*InvertedIndex) *MergeKArrayIterator {
	return &MergeKArrayIterator{
		indexes:           indexes,
		err:               nil,
		nextIndexIterator: make([]func() (IndexIteratorItem, error, bool), len(indexes)),
	}
}

// mergeKSortedArray. merge k inverted index sorted by terms into one inverted index iterator. yield term & its postings lists in sorted order by termID from each inverted index. O(NlogK) where N is total number of terms in all inverted indexes.
func (it *MergeKArrayIterator) mergeKSortedArray() iter.Seq2[heapMergeOutput, error] {
	return func(yield func(heapMergeOutput, error) bool) {
		pq := datastructure.NewMinPriorityQueue[datastructure.HeapMergeItem, int]()
		heap.Init(pq)

		for i, index := range it.indexes {
			indexIterator := NewInvertedIndexIterator(index).IterateInvertedIndex()

			next, stop := iter.Pull2(indexIterator)
			defer stop()
			it.nextIndexIterator[i] = next

			item, err, valid := next()
			if !valid {
				continue
			}

			if err != nil {
				yield(NewHeapMergeOutput(-1, []int{}), fmt.Errorf("error when merge posting lists: %w", err))
				return
			}
			termID, termSize := item.GetTermID(), item.GetTermSize()

			currHeapMergeItem := datastructure.NewHeapMergeItem(termID, []int{i, 0, termSize}, item.GetPostingList())
			pqItem := datastructure.NewPriorityQueueNode[datastructure.HeapMergeItem](termID, currHeapMergeItem)
			heap.Push(pq, pqItem)
		}

		for pq.Len() > 0 {
			curr := heap.Pop(pq).(*datastructure.PriorityQueueNode[datastructure.HeapMergeItem, int])
			termID := curr.GetItem().TermID
			arrIndex := curr.GetItem().Metadata[0]
			insideIndex := curr.GetItem().Metadata[1]
			termSize := curr.GetItem().Metadata[2]
			postingList := curr.GetItem().Postings

			currOutput := NewHeapMergeOutput(termID, postingList)

			if !yield(currOutput, nil) {
				return
			}

			if (insideIndex + 1) < termSize {

				item, err, valid := it.nextIndexIterator[arrIndex]()
				if !valid {
					continue
				}
				if err != nil {
					yield(NewHeapMergeOutput(-1, []int{}), fmt.Errorf("error when merge posting lists: %w", err))
					return
				}
				termID, termSize := item.GetTermID(), item.GetTermSize()

				currHeapMergeItem := datastructure.NewHeapMergeItem(termID, []int{arrIndex, insideIndex + 1, termSize}, item.GetPostingList())
				pqItem := datastructure.NewPriorityQueueNode[datastructure.HeapMergeItem](termID, currHeapMergeItem)
				heap.Push(pq, pqItem)
			}
		}
	}
}
