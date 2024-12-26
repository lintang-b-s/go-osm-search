package pkg

import (
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

func heapMergeKArray(indexes []InvertedIndex) iter.Seq[heapMergeOutput] {
	return func(yield func(heapMergeOutput) bool) {
		pq := NewPriorityQueue[heapMergeItem, int]()


		for i, index := range indexes {
			indexIterator := index.IterateInvertedIndex()

			next, stop := iter.Pull2(indexIterator)
			item, postingList, _ := next()
			termID, termSize := item.TermID, item.TermSize

			currHeapMergeItem := NewHeapMergeItem(termID, []int{i, 0, termSize}, postingList)
			pqItem := NewPriorityQueueNode[heapMergeItem](termID, currHeapMergeItem)
			pq.Push(pqItem)
			stop()
		}

		for pq.Len() > 0 {
			curr := pq.Pop().(*priorityQueueNode[heapMergeItem, int])
			termID := curr.item.TermID
			arrIndex := curr.item.Metadata[0]
			insideIndex := curr.item.Metadata[1]
			termSize := curr.item.Metadata[2]
			postingList := curr.item.Postings

			currOutput := NewHeapMergeOutput(termID, postingList)

			if !yield(currOutput) {
				return
			}

			if (insideIndex + 1) < termSize {
				indexIterator := indexes[arrIndex].IterateInvertedIndex()
				next, stop := iter.Pull2(indexIterator)
				item, postingList, _ := next()
				termID, termSize := item.TermID, item.TermSize

				currHeapMergeItem := NewHeapMergeItem(termID, []int{arrIndex, insideIndex + 1, termSize}, postingList)
				pqItem := NewPriorityQueueNode[heapMergeItem](termID, currHeapMergeItem)
				pq.Push(pqItem)
				stop()
			}

		}
	}

}
