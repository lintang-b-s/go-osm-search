package compress

import (
	"encoding/binary"
	"sync"
)

var BITMASK = []byte{
	0b00000001,
	0b00000011,
	0b00000111,
	0b00001111,
	0b00011111,
	0b00111111,
	0b01111111,
	0b11111111,
}

func getLSB(x byte, n uint8) byte {
	if n > 8 {
		panic("can extract at max 8 bits from the number")
	}
	return x & BITMASK[n-1]
}

var bitShifts = [10]uint8{7, 7, 7, 7, 7, 7, 7, 7, 7, 1}

var bufPool = sync.Pool{
	New: func() any {
		return new([11]byte)
	},
}

func encodeUVarint(x uint64) []byte {
	var i int = 0
	buf := bufPool.Get().(*[11]byte)
	for i = 0; i < len(bitShifts); i++ {
		buf[i] = getLSB(byte(x), bitShifts[i]) | 0b10000000
		x = x >> bitShifts[i]
		if x == 0 {
			break
		}
	}

	buf[i] = buf[i] & 0b01111111
	bufPool.Put(buf)
	return append(make([]byte, 0, i+1), buf[:i+1]...)

}

func decodeUVarint(buf []byte) (uint64, int) {
	v, n := binary.Uvarint(buf)
	return v, n
}

func DecodePostingList(buf []byte) []int {
	var results []int
	for len(buf) > 0 {
		v, n := decodeUVarint(buf)
		if n == 0 {
			break
		}

		results = append(results, int(v))
		buf = buf[n:]
	}
	return results
}

func EncodePostingList(arr []int) []byte {

	buf := make([]byte, 0)
	for i := 0; i < len(arr); i++ {
		buf = append(buf, encodeUVarint(uint64(arr[i]))...)
	}
	return buf
}

// func DecodePostingList(buf []byte) []int {
// 	var results []int
// 	for len(buf) > 0 {
// 		v, n := decodeUVarint(buf)
// 		if n == 0 {
// 			break
// 		}

// 		results = append(results, int(v))
// 		buf = buf[n:]

// 	}

// 	for i := 1; i < len(results); i++ {
// 		results[i] += results[i-1]
// 	}
// 	return results
// }
//  error
// func EncodePostingList(arr []int) []byte {
// 	var res = make([]int, len(arr))
// 	copy(res, arr)
// 	for i := len(arr) - 1; i >= 1; i-- {
// 		res[i] -= res[i-1]
// 	}
// 	buf := make([]byte, 0)
// 	for i := 0; i < len(arr); i++ {
// 		buf = append(buf, encodeUVarint(uint64(res[i]))...)
// 	}
// 	return buf
// }
