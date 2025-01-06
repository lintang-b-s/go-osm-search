package pkg

import (
	"encoding/binary"
	"math/rand"
)

const (
	HEADER_KEY = 1<<31 - 1
	MAX_LEVEL  = 20
)

type SkipListsnode struct {
	key     int
	forward []*SkipListsnode
}

func NewSkipListsNode(key int, level int) *SkipListsnode {
	return &SkipListsnode{
		key:     key,
		forward: make([]*SkipListsnode, level),
	}
}

type SkipLists struct {
	header   *SkipListsnode
	level    int
	maxLevel int
}

func NewSkipLists() SkipLists {
	sl := SkipLists{
		header:   NewSkipListsNode(HEADER_KEY, MAX_LEVEL),
		level:    0,
		maxLevel: MAX_LEVEL,
	}

	for i := 0; i < sl.maxLevel; i++ {
		// sl.header.forward[i] = sl.header
		sl.header.forward[i] = nil ///sl.header//nil
	}

	return sl
}

func (sl *SkipLists) Search(target int) *SkipListsnode {
	x := sl.header
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < target {
			x = x.forward[i]
		}
	}

	x = x.forward[0]
	if x != nil && x.key == target {
		return x
	}
	return nil
}

func (sl *SkipLists) Insert(num int) {
	update := make([]*SkipListsnode, sl.maxLevel)
	x := sl.header
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < num {
			x = x.forward[i]
		}
		update[i] = x
	}

	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		sl.level = sl.level + 1
		newLevel = sl.level
		update[newLevel] = sl.header
	}

	x = NewSkipListsNode(num, newLevel+1)

	for i := 0; i <= newLevel; i++ {
		x.forward[i] = update[i].forward[i]
		update[i].forward[i] = x
	}
}

func (sl *SkipLists) Erase(num int) *SkipListsnode {
	update := make([]*SkipListsnode, sl.maxLevel)
	x := sl.header
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < num {
			x = x.forward[i]
		}
		update[i] = x
	}

	x = x.forward[0]

	if x != nil && x.key == num {
		for i := 0; i <= sl.level; i++ {
			if update[i].forward[i] != x {
				break
			}
			update[i].forward[i] = x.forward[i]
		}

		for sl.level > 0 {
			if sl.header.forward[sl.level] == sl.header {
				sl.level--
			} else {
				break
			}
		}

		return x
	}
	return nil
}

func (sl *SkipLists) randomLevel() int {
	newLevel := 0
	for rand.Float64() < 0.25 {
		newLevel++
	}
	return min(newLevel, sl.maxLevel)
}

func (sl *SkipLists) Serialize() []byte {
	bb := []byte{}

	intBuf := make([]byte, 4)

	binary.LittleEndian.PutUint32(intBuf, uint32(sl.level))
	bb = append(bb, intBuf...)

	levelBuf := make([][]byte, sl.level+1)
	prevLevelOffset := 4 + 4*(sl.level+1) // 4 for level, 4*sl.level buat offset setiap level // offset lists level sl.level

	itemOffsetLevelMap := make(map[int]map[int]int, sl.level+1)

	insideLevelOffsetMap := make(map[int]map[int]int, sl.level+1)

	for i := sl.level; i >= 0; i-- {

		binary.LittleEndian.PutUint32(intBuf, uint32(prevLevelOffset))
		bb = append(bb, intBuf...)

		x := sl.header.forward[i]
		buf := []byte{}

		insideOffset := 0

		if _, ok := itemOffsetLevelMap[i]; !ok {
			itemOffsetLevelMap[i] = make(map[int]int)
		}
		itemOffsetLevelMap[i][HEADER_KEY] = prevLevelOffset + insideOffset

		if _, ok := insideLevelOffsetMap[i]; !ok {
			insideLevelOffsetMap[i] = make(map[int]int)
		}
		insideLevelOffsetMap[i][HEADER_KEY] = insideOffset

		binary.LittleEndian.PutUint32(intBuf, uint32(HEADER_KEY))
		buf = append(buf, intBuf...)
		insideOffset += 4

		binary.LittleEndian.PutUint32(intBuf, 0) // down level offset
		buf = append(buf, intBuf...)
		insideOffset += 4

		binary.LittleEndian.PutUint32(intBuf, 0) // up level offset
		buf = append(buf, intBuf...)
		insideOffset += 4

		// iterate item lists level i.
		for x != nil {

			itemOffsetLevelMap[i][x.key] = prevLevelOffset + insideOffset

			insideLevelOffsetMap[i][x.key] = insideOffset

			binary.LittleEndian.PutUint32(intBuf, uint32(x.key))
			buf = append(buf, intBuf...)
			insideOffset += 4

			binary.LittleEndian.PutUint32(intBuf, 0) // down level offset
			buf = append(buf, intBuf...)
			insideOffset += 4

			binary.LittleEndian.PutUint32(intBuf, 0) // up level offset
			buf = append(buf, intBuf...)
			insideOffset += 4

			x = x.forward[i]
		}

		binary.LittleEndian.PutUint32(intBuf, uint32(HEADER_KEY-1)) // anggap ini nil
		buf = append(buf, intBuf...)
		insideOffset += 4

		levelBuf[i] = buf

		prevLevelOffset += insideOffset

	}

	for i := sl.level; i >= 0; i-- {
		x := sl.header //   header di level i

		for x != nil {
			if _, ok := itemOffsetLevelMap[i-1][x.key]; i > 0 && ok {

				keyPos := insideLevelOffsetMap[i][x.key] // offset x.key di dalam level i
				binary.LittleEndian.PutUint32(intBuf, uint32(itemOffsetLevelMap[i-1][x.key]))
				copy(levelBuf[i][keyPos+4:], intBuf)
			}
			if _, ok := itemOffsetLevelMap[i+1][x.key]; ok && i < sl.level {
				keyPos := insideLevelOffsetMap[i][x.key]                                      // offset x.key di dalam level i                                    // key position di level i
				binary.LittleEndian.PutUint32(intBuf, uint32(itemOffsetLevelMap[i+1][x.key])) // x.key position di level i+1
				copy(levelBuf[i][keyPos+8:], intBuf)
			}
			x = x.forward[i]
		}
	}

	for i := sl.level; i >= 0; i-- {
		buf := levelBuf[i]

		bb = append(bb, buf...)
	}

	return bb
}

type SkipListsReader struct {
	bb []byte

	level int
}

func NewSkipListsReader(bb []byte) SkipListsReader {
	slr := SkipListsReader{
		bb: bb,
	}
	level := int(binary.LittleEndian.Uint32(slr.bb))
	slr.level = int(level)

	return slr
}

/*
level 2= 6		 ->9			   ->25
level 1= 6		 ->9	->17	   ->25
level 0= 3->6->7->9->12->17->19->21->25->26

bytes per item  = [4byte for key, 4 byte for downlevel offset, 4 byte for uplevel offset] = 12 byte per item

bytes
level 2= 8		  ->16				  ->24
level 1= 8	      ->16	  ->24		  ->32
level 0= 8->16->24->32->40->48->56->64->72->80

search(12) bytes offset:
8 (level2) -> 16 (level2) -> 16 level(1) -> 32 kevel(0) -> 40 level(0)

bytes
level 2= 12		   ->24				    ->36
level 1= 12	       ->24	   ->36		    ->48
level 0= 12->24->36->48->60->72->84->96->108->120
*/

// Search. search di serialized skip lists.
func (slr *SkipListsReader) Search(target int) int {

	levelOffset := 4 + 4*(slr.level+1)
	startLevelOffsetOffset := 4

	for i := slr.level; i >= 0; i-- {
		nextLevelOffset := len(slr.bb)
		if i != 0 {
			nextLevelOffset = int(binary.LittleEndian.Uint32(slr.bb[startLevelOffsetOffset+4:]))
		}

		nextInt := int(binary.LittleEndian.Uint32(slr.bb[int(levelOffset)+12 : int(levelOffset)+16]))
		for int(levelOffset+16) <= len(slr.bb[:nextLevelOffset]) && nextInt < target {

			levelOffset += 12
			nextInt = int(binary.LittleEndian.Uint32(slr.bb[int(levelOffset)+12 : int(levelOffset)+16]))

		}

		if i != 0 {
			if levelOffset+4 == nextLevelOffset {
				levelOffset -= 12
			}

			levelOffset = int(binary.LittleEndian.Uint32(slr.bb[levelOffset+4:])) // di HEADER_KEY, downlevel  offset nya salah

		}
		startLevelOffsetOffset += 4

	}

	bbOffset := int(levelOffset)

	x := int(binary.LittleEndian.Uint32(slr.bb[bbOffset+12:]))
	if x == target {
		return x
	}
	return -1
}

func FastPostlingListsIntersection(a, b SkipListsReader) []int {

	answer := []int{}
	zeroLevelOffsetA := int(binary.LittleEndian.Uint32(a.bb[(4 + 4*(a.level)):])) // zero level lists offset dari a
	zeroLevelOffsetB := int(binary.LittleEndian.Uint32(b.bb[(4 + 4*(b.level)):]))

	p1 := int(binary.LittleEndian.Uint32(a.bb[zeroLevelOffsetA:])) // paling bawah (level 0)
	p2 := int(binary.LittleEndian.Uint32(b.bb[zeroLevelOffsetB:])) // paling bawah (level 0)

	for p1 != HEADER_KEY-1 && p2 != HEADER_KEY-1 {

		if p1 == p2 {
			answer = append(answer, p1)
			zeroLevelOffsetA += 12
			zeroLevelOffsetB += 12
			p1 = int(binary.LittleEndian.Uint32(a.bb[zeroLevelOffsetA:]))
			p2 = int(binary.LittleEndian.Uint32(b.bb[zeroLevelOffsetB:]))
		} else if p1 < p2 {

			if zeroLevelOffsetSkipPointer, hasSkip := hasSkipAndItsSkipLessThanB(a, zeroLevelOffsetA, p2); hasSkip {
				// zeroLevelOffsetSkipPointer, hasSkip := hasSkipAndItsSkipLessThanB(a, zeroLevelOffsetA, p2)

				for hasSkip {
					zeroLevelOffsetA = (zeroLevelOffsetSkipPointer)
					p1 = int(binary.LittleEndian.Uint32(a.bb[zeroLevelOffsetA:]))
					zeroLevelOffsetSkipPointer, hasSkip = hasSkipAndItsSkipLessThanB(a, zeroLevelOffsetA, p2)
				}
			} else {
				zeroLevelOffsetA += 12
				p1 = int(binary.LittleEndian.Uint32(a.bb[zeroLevelOffsetA:]))
			}

		} else {

			if zeroLevelOffsetSkipPointer, hasSkip := hasSkipAndItsSkipLessThanB(b, zeroLevelOffsetB, p1); hasSkip {
				// zeroLevelOffsetSkipPointer, hasSkip := hasSkipAndItsSkipLessThanB(b, zeroLevelOffsetB, p1)

				for hasSkip {
					zeroLevelOffsetB = (zeroLevelOffsetSkipPointer)
					p2 = int(binary.LittleEndian.Uint32(b.bb[zeroLevelOffsetB:]))
					zeroLevelOffsetSkipPointer, hasSkip = hasSkipAndItsSkipLessThanB(b, zeroLevelOffsetB, p1)
				}
			} else {
				zeroLevelOffsetB += 12
				p2 = int(binary.LittleEndian.Uint32(b.bb[zeroLevelOffsetB:]))
			}
		}
	}
	return answer[1:]
}

/*
	hasSkipAndItsSkipLessThanB. check jika skip pointer di upper level a[aOffset:] <= b . if yes -> return zeroLevelSkipPointerOffset
	example:

	b = 15

a:
level2  5		  10			 15				20
level1	5		  10			 15				20
level0	5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20

a[aOffset:] = 5 -> return offset level0 dari skip pointer 10
*/

func hasSkipAndItsSkipLessThanB(a SkipListsReader, aOffset int, b int) (int, bool) {

	// zeroLevelOffset := int(binary.LittleEndian.Uint32(a.bb[4+4*(a.level):]))

	aUpLevelOffset := int(binary.LittleEndian.Uint32(a.bb[aOffset+8:])) // +8 buffer offset pada upper level (level 1)
	if aUpLevelOffset == 0 {
		// a[offset:] dont have upper level pointer
		return -1, false
	}

	skipPointerDownLevelOffset := -1
	skipPointerMaxLevel := -1

	for i := 1; i < a.level && aUpLevelOffset != 0; i++ {
		// check di upper skip pointer is less than b. aUpLevelOffset == 0 -> dont have upper level item

		aSkipPointer := int(binary.LittleEndian.Uint32(a.bb[aUpLevelOffset+12 : aUpLevelOffset+16])) // +12 skip pointer dari a di upper level i.
		if int(aUpLevelOffset+12) <= len(a.bb) && aSkipPointer != HEADER_KEY-1 && aSkipPointer <= b {
			// if a[Offset:] has skip pointer & aSkipPointer <= b -> save maxLevelSkipPointer & offset dari skipPointer

			skipPointerDownLevelOffset = int(aUpLevelOffset + 12)
			if i > skipPointerMaxLevel {
				skipPointerMaxLevel = i
			}

		}

		aUpLevelOffset = int(binary.LittleEndian.Uint32(a.bb[aUpLevelOffset+8:])) // +8 buffer =  offset pada upper level i.
	}

	if skipPointerDownLevelOffset != -1 {
		// if skipPointer dari a less or equal than  b -> return level0 offset dari skip pointer
		for j := skipPointerMaxLevel; j > 0; j-- {
			skipPointerDownLevelOffset = int(binary.LittleEndian.Uint32(a.bb[skipPointerDownLevelOffset+4:])) // +4 byte = offset pada downlevel skip pointer a.
		}
		return skipPointerDownLevelOffset, true
	}

	return -1, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func PostingListIntersection(aBuf, bBuf []byte) []int {
	a := DecodePostingList(aBuf)
	b := DecodePostingList(bBuf)
	idx1, idx2 := 0, 0
	result := []int{}

	for idx1 < len(a) && idx2 < len(b) {
		if a[idx1] < b[idx2] {
			idx1++
		} else if b[idx2] < a[idx1] {
			idx2++
		} else {
			result = append(result, a[idx1])
			idx1++
			idx2++
		}
	}
	return result
}

// yang ini bener
// func hasSkipAndItsSkipLessThanB(a SkipListsReader, aOffset int, b int) (int, bool) {

// 	// zeroLevelOffset := int(binary.LittleEndian.Uint32(a.bb[4+4*(a.level):]))

// 	aUpLevelOffset := int(binary.LittleEndian.Uint32(a.bb[aOffset+8:])) // +8 buffer offset pada upper level (level 1)
// 	if aUpLevelOffset == 0 {
// 		// a[offset:] dont have upper level pointer
// 		return -1, false
// 	}
// 	for i := 1; i < a.level && aUpLevelOffset != 0; i++ {
// 		// check di upper skip pointer is less than b. if yes -> return zeroLeveloffset , boolean hasSkip

// 		aSkipPointer := int(binary.LittleEndian.Uint32(a.bb[aUpLevelOffset+12 : aUpLevelOffset+16])) // +12 skip pointer dari a di upper level i.
// 		if int(aUpLevelOffset+12) <= len(a.bb) && aSkipPointer != HEADER_KEY-1 && aSkipPointer <= b {
// 			// if a[Offset:] has skip pointer & aSkipPointer <= b -> return level0 offset dari aSkipPointer

// 			// skipPointerDownLevelOffset := int(aUpLevelOffset + 12) // benar
// 			aUpLevelOffset += 12 // aSkipPointer

// 			for j := i; j > 0; j-- {
// 				aUpLevelOffset = int(binary.LittleEndian.Uint32(a.bb[aUpLevelOffset+4:])) // +4 byte = offset pada downlevel skip pointer a.
// 			}
// 			// bbOffset := int(aUpLevelOffset)
// 			// return bbOffset, true
// 			return aUpLevelOffset, true
// 		}

// 		aUpLevelOffset = int(binary.LittleEndian.Uint32(a.bb[aUpLevelOffset+8:])) // +8 buffer =  offset pada upper level i.
// 	}

// 	return -1, false
// }
