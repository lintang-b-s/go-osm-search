package datastructure

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
		outInt, err := out.GetAllItems()
		if err != nil {
			t.Error(err)
		}
		// kelipatan 30 sampai 3000
		assert.Equal(t, 100, len(outInt))
		assert.Equal(t, 30, outInt[0])
		assert.Equal(t, 3000, outInt[len(outInt)-1])
	})

	tests := []struct {
		name string
		p1   []int
		p2   []int

		wantErr error
	}{
		{
			name: "success 1",
			p1:   []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29, 31, 40, 50, 60, 70, 80, 90, 100},
			p2: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
				21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37,
				38, 39, 40},

			wantErr: nil,
		},
	}

	for _, tt := range tests {
		sl1 := NewSkipLists()
		for _, p := range tt.p1 {
			sl1.Insert(p)
		}

		sl2 := NewSkipLists()
		for _, p := range tt.p2 {
			sl2.Insert(p)
		}
		buf1 := sl1.Serialize()
		buf2 := sl2.Serialize()

		reader1 := NewSkipListsReader(buf1)
		reader2 := NewSkipListsReader(buf2)

		out := FastPostingListsIntersection(reader1, reader2)
		outInt, err := out.GetAllItems()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, 17, len(outInt))
		assert.Equal(t, 1, outInt[0])
		assert.Equal(t, 40, outInt[len(outInt)-1])
	}

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
