package index

import (
	"encoding/binary"
	"fmt"
	"iter"
	"math"
	"os"

	"github.com/lintang-b-s/osm-search/pkg/compress"
)

// inverted index untuk satu field tertentu
type InvertedIndex struct {
	indexName          string
	dirName            string
	postingMetadata    map[int][3]int // termID -> [startPositionInIndexFile, len(postingList), lengthInBytesOfPostingLists]
	indexFilePath      string
	metadataFilePath   string
	terms              []int
	indexFile          *os.File
	lenFieldInDoc      map[int]int // docID -> termCount (jumlah term di dalam document) untuk field tertentu
	averageFieldLength float64
	currTermPosition   int
}

func NewInvertedIndex(index_name, directoryName, workingDir string,
) *InvertedIndex {

	indexFilePath := directoryName + "/" + index_name + ".index"
	metadataFilePath := directoryName + "/" + index_name + ".metadata"
	if workingDir != "/" {
		indexFilePath = workingDir + "/" + directoryName + "/" + index_name + ".index"
		metadataFilePath = workingDir + "/" + directoryName + "/" + index_name + ".metadata"
	}

	return &InvertedIndex{
		indexName:        index_name,
		dirName:          directoryName,
		postingMetadata:  make(map[int][3]int),
		indexFilePath:    indexFilePath,
		metadataFilePath: metadataFilePath,
		terms:            []int{},
		lenFieldInDoc:    make(map[int]int),
		currTermPosition: 0,
	}
}

func (Idx *InvertedIndex) SetLenFieldInDoc(lenFieldInDoc map[int]int) {
	Idx.lenFieldInDoc = lenFieldInDoc
}

func (Idx *InvertedIndex) GetLenFieldInDoc() map[int]int {
	return Idx.lenFieldInDoc
}

func (Idx *InvertedIndex) GetAverageFieldLength() float64 {
	return Idx.averageFieldLength
}

func (Idx *InvertedIndex) OpenWriter() error {
	file, err := os.OpenFile(Idx.indexFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	Idx.indexFile = file
	return nil
}

func (Idx *InvertedIndex) Close() error {
	if Idx.indexFile != nil {
		err := Idx.indexFile.Close()
		if err != nil {
			return err
		}

		metadataFile, err := os.OpenFile(Idx.metadataFilePath, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		defer metadataFile.Close()

		err = metadataFile.Truncate(0)
		if err != nil {
			return err
		}

		metadataBuf := Idx.SerializeMetadata()

		_, err = metadataFile.Write(metadataBuf)
		if err != nil {
			return err
		}

		pwd, err := os.Getwd()
		if err != nil {
			return err
		}

		var bufferSizeFile *os.File
		if pwd != "/" {
			bufferSizeFile, err = os.OpenFile(pwd+"/"+Idx.dirName+"/"+Idx.indexName+"_size.metadata", os.O_RDWR|os.O_CREATE, 0666)
		} else {
			bufferSizeFile, err = os.OpenFile(Idx.dirName+"/"+Idx.indexName+"_size.metadata", os.O_RDWR|os.O_CREATE, 0666)
		}
		if err != nil {
			return err
		}

		defer bufferSizeFile.Close()

		err = bufferSizeFile.Truncate(0)
		if err != nil {
			return err
		}

		metadataBufferSize := len(metadataBuf)

		bufferSizeBuf := make([]byte, 100)
		binary.LittleEndian.PutUint64(bufferSizeBuf[:], math.Float64bits(float64(metadataBufferSize)))

		_, err = bufferSizeFile.Write(bufferSizeBuf)
		if err != nil {
			return err
		}

	}
	return nil
}

func (Idx *InvertedIndex) OpenReader() error {
	file, err := os.OpenFile(Idx.indexFilePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	Idx.indexFile = file

	metadataFile, err := os.OpenFile(Idx.metadataFilePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer metadataFile.Close()

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var bufferSizeFile *os.File
	if pwd != "/" {
		bufferSizeFile, err = os.OpenFile(pwd+"/"+Idx.dirName+"/"+Idx.indexName+"_size.metadata", os.O_RDONLY|os.O_CREATE, 0666)
	} else {
		bufferSizeFile, err = os.OpenFile(Idx.dirName+"/"+Idx.indexName+"_size.metadata", os.O_RDONLY|os.O_CREATE, 0666)
	}

	if err != nil {
		return err
	}
	defer bufferSizeFile.Close()

	buf := make([]byte, 100)
	_, err = bufferSizeFile.Read(buf)
	if err != nil {
		return err
	}
	approxBufferSize := int(math.Float64frombits(binary.LittleEndian.Uint64(buf)))

	buf = make([]byte, approxBufferSize)
	_, err = metadataFile.Read(buf)
	if err != nil {
		return err
	}
	Idx.DeserializeMetadata(buf)

	return nil
}

func (Idx *InvertedIndex) GetPostingList(termID int) ([]int, error) {
	postingMetadata, ok := Idx.postingMetadata[termID]
	if !ok {
		return []int{}, nil // in case termID not found
	}
	startPositionInIndexFile := int64(postingMetadata[0])
	Idx.indexFile.Seek(startPositionInIndexFile, 0)
	buf := make([]byte, postingMetadata[2])
	_, err := Idx.indexFile.Read(buf)
	if err != nil {
		return []int{}, err
	}
	postingList := compress.DecodePostingsList(buf)

	return postingList, nil
}

func (Idx *InvertedIndex) AppendPostingList(termID int, postingList []int) error {
	encodedPostingList := compress.EncodePostingsList(postingList)
	startPositionInIndexFile, err := Idx.indexFile.Seek(0, 2)
	if err != nil {
		return err
	}
	lengthInBytesOfPostingList, err := Idx.indexFile.Write(encodedPostingList)
	if err != nil {
		return err
	}

	Idx.terms = append(Idx.terms, termID)

	Idx.postingMetadata[termID] = [3]int{int(startPositionInIndexFile), len(postingList),
		lengthInBytesOfPostingList}

	return nil
}

type IndexIteratorItem struct {
	termID      int
	termSize    int
	postingList []int
}

func NewIndexIteratorItem(termID int, termSize int, postingList []int) IndexIteratorItem {
	return IndexIteratorItem{
		termID:      termID,
		termSize:    termSize,
		postingList: postingList,
	}
}

func (tem *IndexIteratorItem) GetTermID() int {
	return tem.termID
}

func (tem *IndexIteratorItem) GetTermSize() int {
	return tem.termSize
}

func (tem *IndexIteratorItem) GetPostingList() []int {
	return tem.postingList
}

type InvertedIndexIterator struct {
	invertedIndex *InvertedIndex
}

func NewInvertedIndexIterator(idx *InvertedIndex) *InvertedIndexIterator {
	return &InvertedIndexIterator{invertedIndex: idx}
}

// IterateInvertedIndex. iterate inverted index sorted by termID. yield termID and postinglists. O(N) where N is total number of terms in inverted index.
func (it *InvertedIndexIterator) IterateInvertedIndex() iter.Seq2[IndexIteratorItem, error] {
	return func(yield func(IndexIteratorItem, error) bool) {
		for it.invertedIndex.currTermPosition < len(it.invertedIndex.terms) {
			termID := it.invertedIndex.terms[it.invertedIndex.currTermPosition]
			it.invertedIndex.currTermPosition += 1
			startPosition, _, lengthInBytes := it.invertedIndex.postingMetadata[termID][0], it.invertedIndex.postingMetadata[termID][1], it.invertedIndex.postingMetadata[termID][2]
			_, err := it.invertedIndex.indexFile.Seek(int64(startPosition), 0)
			if err != nil {
				yield(NewIndexIteratorItem(-1, -1, []int{}), fmt.Errorf("error when iterating inverted index: %w", err))
				return
			}
			buf := make([]byte, lengthInBytes)
			_, err = it.invertedIndex.indexFile.Read(buf)
			if err != nil {
				yield(NewIndexIteratorItem(-1, -1, []int{}), fmt.Errorf("error when iterating inverted index: %w", err))
				return
			}

			postingList := compress.DecodePostingsList(buf)
			item := NewIndexIteratorItem(termID, len(it.invertedIndex.terms), postingList)

			if !yield(item, nil) {
				return
			}
		}
	}
}

func (Idx *InvertedIndex) ExitAndRemove() error {
	err := Idx.Close()
	if err != nil {
		return err
	}
	err = os.Remove(Idx.indexFilePath)
	if err != nil {
		return err
	}
	err = os.Remove(Idx.metadataFilePath)
	if err != nil {
		return err
	}
	return nil
}

func (Idx *InvertedIndex) GetAproximateMetadataBufferSize() int {
	allLen := 4 * 3 // 4 byte* 3
	termsSize := 4 * len(Idx.terms)
	postingMetadata := 4 * 4 * len(Idx.postingMetadata)
	docTermCountDict := 4 * 2 * len(Idx.lenFieldInDoc)
	return allLen + termsSize + postingMetadata + docTermCountDict + 8
}

func (Idx *InvertedIndex) SerializeMetadata() []byte {
	approxBufferSize := Idx.GetAproximateMetadataBufferSize()
	buf := make([]byte, approxBufferSize)
	leftPos := 0

	binary.LittleEndian.PutUint32(buf[leftPos:], uint32(len(Idx.terms)))
	leftPos += 4 // 32 bit

	binary.LittleEndian.PutUint32(buf[leftPos:], uint32(len(Idx.postingMetadata)))
	leftPos += 4 // 32 bit

	binary.LittleEndian.PutUint32(buf[leftPos:], uint32(len(Idx.lenFieldInDoc)))
	leftPos += 4 // 32 bit

	for _, term := range Idx.terms {
		// kita pakai uint32bit untuk menyimpan term

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(term))
		leftPos += 4 // 32 bit
	}

	for term, val := range Idx.postingMetadata {

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(term))
		leftPos += 4 // 32 bit

		startPositionInIndexFile := val[0]    // 4 byte
		lenPostingList := val[1]              // 4 byte
		lengthInBytesOfPostingLists := val[2] // 4 byte

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(lengthInBytesOfPostingLists))
		leftPos += 4

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(lenPostingList))
		leftPos += 4

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(startPositionInIndexFile))
		leftPos += 4

	}

	for docID, termCount := range Idx.lenFieldInDoc {
		// docID = 4 byte, termCount = 4 byte

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(docID))
		leftPos += 4 // 32 bit

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(termCount))
		leftPos += 4 // 32 bit

		Idx.averageFieldLength += float64(termCount)
	}

	Idx.averageFieldLength = Idx.averageFieldLength / float64(len(Idx.lenFieldInDoc))

	binary.LittleEndian.PutUint64(buf[leftPos:], math.Float64bits(Idx.averageFieldLength))

	return buf
}

// DeserializeMetadata.
func (Idx *InvertedIndex) DeserializeMetadata(buf []byte) {
	leftPos := 0

	termCount := int(binary.LittleEndian.Uint32(buf[0:4]))
	leftPos += 4

	PostingMetadatacount := int(binary.LittleEndian.Uint32(buf[4:8]))
	leftPos += 4

	docTermCountDictCount := int(binary.LittleEndian.Uint32(buf[8:12]))
	leftPos += 4

	Idx.terms = make([]int, termCount)
	Idx.postingMetadata = make(map[int][3]int)
	Idx.lenFieldInDoc = make(map[int]int)

	for i := 0; i < termCount; i++ {

		term := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4
		Idx.terms[i] = term
	}

	for i := 0; i < PostingMetadatacount; i++ {

		term := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4

		lengthInBytesOfPostingLists := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4

		lenPostingList := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4

		startPositionInIndexFile := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4

		Idx.postingMetadata[term] = [3]int{startPositionInIndexFile, lenPostingList, lengthInBytesOfPostingLists}
	}

	for i := 0; i < docTermCountDictCount; i++ {

		docID := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4

		termCount := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4

		Idx.lenFieldInDoc[docID] = termCount
	}

	Idx.averageFieldLength = math.Float64frombits(binary.LittleEndian.Uint64(buf[leftPos:]))
}
