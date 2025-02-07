package compress

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	tests := []struct {
		postingLists []int
	}{
		{
			postingLists: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			postingLists: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 100, 1300, 1500},
		},
		{
			postingLists: []int{1300, 1500, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000},
		},
		{
			postingLists: []int{10000, 11000, 12000, 13000, 14000, 15000},
		},
		{
			postingLists: []int{10000, 12000, 15000, 16000, 19000, 23000},
		},
		{
			postingLists: []int{1000000, 1200000, 1300000, 1400000, 1500000, 1600000},
		},
	}
	t.Run("test encode decode", func(t *testing.T) {
		postingLists := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 100, 1300, 1500}
		encoded := EncodePostingsList(postingLists)
		decoded := DecodePostingsList(encoded)
		assert.Equal(t, postingLists, decoded)
	})

	for _, tt := range tests {
		t.Run("test encode decode 2", func(t *testing.T) {
			encoded := EncodePostingsList(tt.postingLists)
			decoded := DecodePostingsList(encoded)
			assert.Equal(t, tt.postingLists, decoded)
		})
	}

	t.Run("random", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			arr := make([]int, 90)
			for i := 0; i < 90; i++ {
				num := rand.Intn(250000)
				arr[i] = num
			}
			sort.Ints(arr)

			encoded := EncodePostingsList(arr)
			decoded := DecodePostingsList(encoded)
			assert.Equal(t, arr, decoded)
		}
	})

	t.Run("example", func(t *testing.T) {
		arr := []int{824,829,215406}
		encoded := EncodePostingsList(arr)
		decoded := DecodePostingsList(encoded)
		assert.Equal(t, arr, decoded)
	})
}

func TestRunLengthEncoding(t *testing.T) {

	postingsList := []int{1, 1, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4, 4, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 8, 9, 9, 10, 10}
	encoded := RunLengthEncoding(postingsList)
	expect := []int{1, 4, 2, 3, 3, 3, 4, 5, 5, 2, 6, 2, 7, 2, 8, 3, 9, 2, 10, 2}
	assert.Equal(t, expect, encoded)

}
