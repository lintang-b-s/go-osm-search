package pkg

// https://nlp.stanford.edu/IR-book/pdf/05comp.pdf (Variable byte codes) (figure 5.8 function VBEncodeNumber(n), VBEncode(numbers), VBDecode(bytestream))

func encodeNumber(number int) []byte {
	var bytesList = []byte{}
	for {
		b := byte(number % 128)
		bytesList = append([]byte{b}, bytesList...)
		if number < 128 {
			break
		}
		number /= 128
	}
	bytesList[len(bytesList)-1] += 128
	return bytesList
}

func Encode(numbers []int) []byte {
	var result = []byte{}
	for _, n := range numbers {
		result = append(result, encodeNumber(n)...)
	}
	return result
}

func Decode(bytestream []byte) []int {
	var n int
	var numbers = []int{}
	for _, b := range bytestream {
		if b < 128 {
			n = 128*n + int(b)
		} else {
			n = 128*n + int(b-128)
			numbers = append(numbers, n)
			n = 0
		}
	}
	return numbers
}

func EncodePostingList(postingList []int) []byte {
	return Encode(postingList) 
}

func DecodePostingList(bytestream []byte) []int {
	return Decode(bytestream) 
}
