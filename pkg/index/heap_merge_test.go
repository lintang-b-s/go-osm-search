package index

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeKArray(t *testing.T) {
	prepare(t)
	t.Run("success merge k sorted index", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}

		// inverted index 1
		invIndex := NewInvertedIndex("test", "test", pwd)
		err = invIndex.OpenWriter()
		if err != nil {
			t.Error(err)
		}
		defer invIndex.Close()

		err = invIndex.AppendPostingList(1, []int{1, 2, 3, 4, 5})
		if err != nil {
			t.Error(err)
		}
		err = invIndex.AppendPostingList(2, []int{6, 7, 8, 9, 10})
		if err != nil {
			t.Error(err)
		}

		// inverted index 2
		invIndex2 := NewInvertedIndex("test2", "test", pwd)
		err = invIndex2.OpenWriter()
		if err != nil {
			t.Error(err)
		}
		defer invIndex2.Close()

		err = invIndex2.AppendPostingList(2, []int{16, 17, 18, 19, 20})
		if err != nil {
			t.Error(err)
		}
		err = invIndex2.AppendPostingList(3, []int{11, 12, 13, 14, 15})
		if err != nil {
			t.Error(err)
		}

		mergeIterator := NewMergeKArrayIterator([]*InvertedIndex{invIndex, invIndex2}).mergeKSortedArray()
		merged := []int{}

		expectedPostings := []int{1, 2, 3, 4, 5, 16, 17, 18, 19, 20, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
		for item, err := range mergeIterator {
			if err != nil {
				t.Error(err)
			}
			merged = append(merged, item.Postings...)
		}

		assert.Equal(t, expectedPostings, merged)

	})

	t.Run("fail merge k sorted index. file closed", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		prepare(t)
		// inverted index 1
		invIndex := NewInvertedIndex("test", "test", pwd)
		err = invIndex.OpenWriter()
		if err != nil {
			t.Error(err)
		}

		err = invIndex.AppendPostingList(1, []int{1, 2, 3, 4, 5})
		if err != nil {
			t.Error(err)
		}
		err = invIndex.AppendPostingList(2, []int{6, 7, 8, 9, 10})
		if err != nil {
			t.Error(err)
		}

		// inverted index 2
		invIndex2 := NewInvertedIndex("test2", "test", pwd)
		err = invIndex2.OpenWriter()
		if err != nil {
			t.Error(err)
		}

		err = invIndex2.AppendPostingList(2, []int{16, 17, 18, 19, 20})
		if err != nil {
			t.Error(err)
		}
		err = invIndex2.AppendPostingList(3, []int{11, 12, 13, 14, 15})
		if err != nil {
			t.Error(err)
		}

		mergeIterator := NewMergeKArrayIterator([]*InvertedIndex{invIndex, invIndex2}).mergeKSortedArray()
		merged := []int{}

		invIndex.Close()
		invIndex2.Close()

		expectedPostings := []int{}
		for item, err := range mergeIterator {

			assert.Error(t, err)
			merged = append(merged, item.Postings...)

		}

		assert.Equal(t, expectedPostings, merged)

	})
}
