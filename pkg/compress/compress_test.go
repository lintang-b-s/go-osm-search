package compress

import (
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
		encoded := EncodePostingList(postingLists)
		decoded := DecodePostingList(encoded)
		assert.Equal(t, postingLists, decoded)
	})

	for _, tt := range tests {
		t.Run("test encode decode 2", func(t *testing.T) {
			encoded := EncodePostingList(tt.postingLists)
			decoded := DecodePostingList(encoded)
			assert.Equal(t, tt.postingLists, decoded)
		})
	}
}
