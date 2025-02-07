package compress

import (
	"bytes"
	"encoding/binary"
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

func encodeUVarint(x uint64) []byte {
	var i int = 0
	var buf [11]byte
	for i = 0; i < len(bitShifts); i++ {
		buf[i] = getLSB(byte(x), bitShifts[i]) | 0b10000000
		x = x >> bitShifts[i]
		if x == 0 {
			break
		}
	}

	buf[i] = buf[i] & 0b01111111

	return append(make([]byte, 0, i+1), buf[:i+1]...)

}

func decodeUVarint(buf []byte) (uint64, int) {
	v, n := binary.Uvarint(buf)
	return v, n
}

func RunLengthEncoding(arr []int) []int {
	encoded := make([]int, 0)
	s := 0
	count := 0

	for i := 0; i < len(arr); i++ {
		if arr[i] != arr[s] {
			// save the element

			encoded = append(encoded, arr[s])

			// save count of the previous element
			encoded = append(encoded, count)

			s = i
			count = 0
		}
		count++
	}

	encoded = append(encoded, arr[s])
	encoded = append(encoded, count)
	return encoded
}
func vbEncodeNum(n int) []byte {
	var buf = []byte{}
	for {
		bb := make([]byte, 1)
		bb[0] = byte(n & 0x7f) // n mod 128
		buf = append(bb, buf...)
		if n < 128 { // n < 128
			break
		}
		n >>= 7 // n div 128
	}
	buf[len(buf)-1] |= 0x80 // buf[len(buf)] += 128
	return buf
}

func vbDecode(bs []byte) []int {
	numbers := make([]int, 0)
	var n int = 0
	for i := 0; i < len(bs); i++ {
		if int(bs[i]) < 128 { // bs[i] < 128
			n = n<<7 + int(bs[i]) // n*128 + bs[i]
		} else {
			n = n<<7 + (int(bs[i]) - 128) // n*128 + (bs[i] - 128)
			numbers = append(numbers, n)
			n = 0
		}
	}
	return numbers
}

func vebDecode2(buf []byte) []int {
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

func EncodePostingsList(postingsList []int) []byte {

	var buf bytes.Buffer
	for _, v := range postingsList {
		buf.Write(encodeUVarint(uint64(v)))
	}
	return buf.Bytes()
}

func DecodePostingsList(bs []byte) []int {
	newBB := make([]byte, len(bs))
	copy(newBB, bs)
	numbers := vebDecode2(newBB)

	return numbers
}

func EncodePostingsList2(postingsList []int) []byte {

	var buf bytes.Buffer
	for _, v := range postingsList {
		buf.Write(vbEncodeNum(v))
	}
	return buf.Bytes()
}

func DecodePostingsList2(bs []byte) []int {
	newBB := make([]byte, len(bs))
	copy(newBB, bs)
	numbers := vbDecode(newBB)

	return numbers
}



/*
func EncodePostingsListDeltaError(postingsList []int) []byte {
	postingsListCopy := make([]int, len(postingsList))
	copy(postingsListCopy, postingsList)
	prev := postingsListCopy[0]
	for i := 1; i < len(postingsListCopy); i++ {
		curr := postingsListCopy[i]
		postingsListCopy[i] = curr - prev
		prev = curr
	}

	var buf bytes.Buffer
	for _, v := range postingsListCopy {
		buf.Write(encodeUVarint(uint64(v)))
	}
	return buf.Bytes()
}

// delta decoding error pas di load test pake k6 wkwkwkwk.
func DecodePostingsListDeltaError(bs []byte) []int {
	newBB := make([]byte, len(bs))
	copy(newBB, bs)
	numbers := vebDecode2(newBB)
	for i := 1; i < len(numbers); i++ {
		numbers[i] = numbers[i-1] + numbers[i]
	}
	return numbers
}
*/
