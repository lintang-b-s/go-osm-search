package pkg

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipList(t *testing.T) {

	t.Run("test skip list in-mem", func(t *testing.T) {
		sl := NewSkipLists()
		sl.Insert(3)
		sl.Insert(9)
		sl.Insert(7)
		sl.Insert(6)
		sl.Insert(12)
		sl.Insert(17)
		sl.Insert(19)
		sl.Insert(21)
		sl.Insert(25)
		sl.Insert(26)

		x := sl.Search(17)
		assert.Equal(t, 17, x.key)

		x = sl.Search(18)
		assert.Nil(t, x)

		x = sl.Search(9)
		assert.Equal(t, 9, x.key)

		sl.Erase(9)
		x = sl.Search(9)
		assert.Nil(t, x)

		sl.Erase(17)
		x = sl.Search(17)
		assert.Nil(t, x)

		x = sl.Search(26)
		assert.Equal(t, 26, x.key)

		for i := 1; i <= 1000; i++ {
			sl.Insert(2 * i)
		}

		for i := 1; i <= 1000; i++ {
			x = sl.Search(2 * i)
			assert.Equal(t, x.key, 2*i)
		}
	})

	t.Run("test skip list reader", func(t *testing.T) {
		sl := NewSkipLists()
		sl.Insert(3)
		sl.Insert(9)
		sl.Insert(7)
		sl.Insert(6)
		sl.Insert(12)
		sl.Insert(17)
		sl.Insert(19)
		sl.Insert(21)
		sl.Insert(25)
		sl.Insert(26)

		buf := sl.Serialize()

		slReader := NewSkipListsReader(buf)
		x := slReader.Search(9)
		assert.Equal(t, x, 9)

		x = slReader.Search(17)
		assert.Equal(t, x, 17)

		x = slReader.Search(18)
		assert.Equal(t, x, -1)

		x = slReader.Search(3)
		assert.Equal(t, x, 3)

		x = slReader.Search(7)
		assert.Equal(t, x, 7)

		x = slReader.Search(25) 
		assert.Equal(t, x, 25)

		x = slReader.Search(26)
		assert.Equal(t, x, 26)

	})

	t.Run("test skip list reader Intersection", func(t *testing.T) {
		sl := NewSkipLists()
		sl.Insert(3)
		sl.Insert(9)
		sl.Insert(7)
		sl.Insert(6)
		sl.Insert(12)
		sl.Insert(17)
		sl.Insert(19)
		sl.Insert(21)
		sl.Insert(25)
		sl.Insert(26)

		buf := sl.Serialize()

		slReader1 := NewSkipListsReader(buf)

		// sl 2
		sl2 := NewSkipLists()
		sl2.Insert(3)
		sl2.Insert(9)
		sl2.Insert(7)
		sl2.Insert(6)
		sl2.Insert(12)
		sl2.Insert(17)
		sl2.Insert(19)
		sl2.Insert(21)
		sl2.Insert(25)
		sl2.Insert(26)

		buf2 := sl2.Serialize()

		slReader2 := NewSkipListsReader(buf2)

		intersection := FastPostingListsIntersection(slReader1, slReader2)
		assert.Equal(t, []int{3, 6, 7, 9, 12, 17, 19, 21, 25, 26}, intersection)

	})

}

func InitSkipList() SkipListsReader {
	sl := NewSkipLists()

	for i := 0; i <= 100000; i++ {
		sl.Insert(i)
	}

	buf := sl.Serialize()
	slReader := NewSkipListsReader(buf)

	return slReader
}

func TestSkipListsReaderSearch(t *testing.T) {
	slReader := InitSkipList()

	testCases := []struct {
		expected int
		input    int
	}{}

	for i := 1; i <= 1000; i++ {
		expect := i * 3
		testCases = append(testCases, struct {
			expected int
			input    int
		}{expected: expect, input: expect})
	}

	for idx, tc := range testCases {
		tc := tc
		res := slReader.Search(tc.input)
		t.Run(fmt.Sprintf("test search %d", idx), func(t *testing.T) {
			assert.Equal(t, tc.expected, res)
		})
	}
}

func TestPostingListIntersection(t *testing.T) {
	t.Run("intersection random large", func(t *testing.T) {

		sl1 := NewSkipLists()
		for i := 1; i <= 1000; i++ { //10,20,....,10000
			sl1.Insert(10 * i)
		}

		sl2 := NewSkipLists()
		for i := 1; i <= 1000; i++ { // 3,6,9....3000
			sl2.Insert(3 * i)
		}
		buf1 := sl1.Serialize()
		buf2 := sl2.Serialize()

		reader1 := NewSkipListsReader(buf1)
		reader2 := NewSkipListsReader(buf2)

		out := FastPostingListsIntersection(reader1, reader2)
		// kelipatan 30 sampai 3000
		assert.Equal(t, 100, len(out))
		assert.Equal(t, 30, out[0])
		assert.Equal(t, 3000, out[len(out)-1])
	})

}

func InitReader() (SkipListsReader, SkipListsReader) {
	sl1 := NewSkipLists()
	for i := 1; i <= 1000000; i++ {
		sl1.Insert(1 * i)
	}

	sl2 := NewSkipLists()
	for i := 1; i <= 1000000; i++ {
		sl2.Insert(3 * i)
	}
	buf1 := sl1.Serialize()
	buf2 := sl2.Serialize()

	reader1 := NewSkipListsReader(buf1)
	reader2 := NewSkipListsReader(buf2)
	return reader1, reader2
}

// BenchmarkPostingListIntersection-12    	     178	   6561755 ns/op	13317370 B/op	      33 allocs/op
func BenchmarkPostingListIntersection(b *testing.B) {
	reader1, reader2 := InitReader()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {

		FastPostingListsIntersection(reader1, reader2)

	}
}

// BenchmarkPostingListIntersection2-12    	      50	  22463468 ns/op	96673566 B/op	     109 allocs/op
func BenchmarkPostingListIntersection2(b *testing.B) {
	list1 := []int{}
	for i := 1; i <= 1000000; i++ {

		list1 = append(list1, 1*i)
	}

	list2 := []int{}
	for i := 1; i <= 1000000; i++ {

		list2 = append(list2, 3*i)
	}

	lBuf1 := EncodePostingList(list1)
	lBuf2 := EncodePostingList(list2)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		PostingListIntersection(lBuf1, lBuf2)

	}
}
