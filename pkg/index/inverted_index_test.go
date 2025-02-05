package index

import (
	"errors"
	"io/fs"
	"iter"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func prepare(t *testing.T) {
	_, err := os.Stat("test")

	if errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir("test", 0700)
		if err != nil {
			t.Error(err)
		}
	}

	pwd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	indexFilePath := pwd + "/" + "test" + "/" + "test" + ".index"
	metadataFilePath := pwd + "/" + "test" + "/" + "test" + ".metadata"

	_, err = os.Stat(indexFilePath)
	if err == nil {
		err = os.Remove(indexFilePath)
		if err != nil {
			t.Error(err)
		}
		err = os.Remove(metadataFilePath)
		if err != nil {
			t.Error(err)
		}
	}

}

func TestAppendPostingsList(t *testing.T) {
	t.Run("success append posting list", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		prepare(t)

		invIndex := NewInvertedIndex("test", "test", pwd)
		invIndex.OpenWriter()
		defer invIndex.Close()

		err = invIndex.AppendPostingList(1, []int{1, 2, 3, 4, 5})
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, 5, invIndex.postingMetadata[1][1])
		assert.Equal(t, 0, invIndex.postingMetadata[1][0])

		indexIterator := NewInvertedIndexIterator(invIndex).IterateInvertedIndex()
		next, stop := iter.Pull2(indexIterator)
		defer stop()
		item, err, valid := next()
		if !valid {
			t.Errorf("expected valid item")
		}
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, 1, item.GetTermID())
		assert.Equal(t, 1, item.GetTermSize())
		assert.Equal(t, []int{1, 2, 3, 4, 5}, item.GetPostingList())
	})

	t.Run("error append posting list . file not open", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		prepare(t)

		invIndex := NewInvertedIndex("test", "test", pwd)
		invIndex.OpenWriter()
		invIndex.Close()

		err = invIndex.AppendPostingList(1, []int{1, 2, 3, 4, 5})
		assert.Error(t, err)
	})
}

func TestGetPostingList(t *testing.T) {
	t.Run("success get posting list", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		prepare(t)

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

		postings, err := invIndex.GetPostingList(1)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, []int{1, 2, 3, 4, 5}, postings)
	})

	t.Run("error get posting list . file not open", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		prepare(t)

		invIndex := NewInvertedIndex("test", "test", pwd)
		err = invIndex.OpenWriter()
		if err != nil {
			t.Error(err)
		}

		err = invIndex.AppendPostingList(1, []int{1, 2, 3, 4, 5})
		if err != nil {
			t.Error(err)
		}
		invIndex.Close()

		_, err = invIndex.GetPostingList(1)
		assert.Error(t, err)
	})

	t.Run("termID not found in index", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		prepare(t)

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

		postings, err := invIndex.GetPostingList(2)
		assert.Nil(t, err)
		assert.Empty(t, postings)
	})

}

func TestIterateInvertedIndex(t *testing.T) {
	t.Run("success iterate inverted index", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		prepare(t)

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

		indexIterator := NewInvertedIndexIterator(invIndex).IterateInvertedIndex()
		next, stop := iter.Pull2(indexIterator)
		defer stop()
		item, err, valid := next()
		if !valid {
			t.Errorf("expected valid item")
		}
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, 1, item.GetTermID())
		assert.Equal(t, 2, item.GetTermSize())
		assert.Equal(t, []int{1, 2, 3, 4, 5}, item.GetPostingList())

		item, err, valid = next()
		if !valid {
			t.Errorf("expected valid item")
		}

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, 2, item.GetTermID())
		assert.Equal(t, 2, item.GetTermSize())
		assert.Equal(t, []int{6, 7, 8, 9, 10}, item.GetPostingList())

	})

	t.Run("error iterate inverted index . file not open", func(t *testing.T) {
		pwd, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		prepare(t)

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

		invIndex.Close()

		indexIterator := NewInvertedIndexIterator(invIndex).IterateInvertedIndex()
		next, stop := iter.Pull2(indexIterator)
		defer stop()
		item, err, valid := next()
		assert.Error(t, err)
		assert.Equal(t, NewIndexIteratorItem(-1, -1, []int{}), item)
		assert.True(t, valid)
	})

}
