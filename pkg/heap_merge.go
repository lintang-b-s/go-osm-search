package pkg

import (
	"container/heap"
	"fmt"
	"iter"
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

func (it *MergeKArrayIterator) mergeKArray() iter.Seq2[heapMergeOutput, error] {
	return func(yield func(heapMergeOutput, error) bool) {
		pq := NewMinPriorityQueue[heapMergeItem, int]()
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
				yield(NewHeapMergeOutput(-1, []int{}),  fmt.Errorf("error when merge posting lists: %w", err))
				return
			}
			termID, termSize := item.GetTermID(), item.GetTermSize()

			currHeapMergeItem := NewHeapMergeItem(termID, []int{i, 0, termSize}, item.GetPostingList())
			pqItem := NewPriorityQueueNode[heapMergeItem](termID, currHeapMergeItem)
			heap.Push(pq, pqItem)

		}

		for pq.Len() > 0 {
			curr := heap.Pop(pq).(*priorityQueueNode[heapMergeItem, int])
			termID := curr.item.TermID
			arrIndex := curr.item.Metadata[0]
			insideIndex := curr.item.Metadata[1]
			termSize := curr.item.Metadata[2]
			postingList := curr.item.Postings

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

				currHeapMergeItem := NewHeapMergeItem(termID, []int{arrIndex, insideIndex + 1, termSize}, item.GetPostingList())
				pqItem := NewPriorityQueueNode[heapMergeItem](termID, currHeapMergeItem)
				heap.Push(pq, pqItem)
			}
		}
	}
}
