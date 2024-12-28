package pkg

import (
	"encoding/binary"
	"errors"
	"iter"
	"math"
	"os"
)

type InvertedIndex struct {
	IndexName        string
	DirName          string
	PostingMetadata  map[int][]int // termID -> [startPositionInIndexFile, len(postingList), lengthInBytesOfPostingLists]
	IndexFilePath    string
	MetadataFilePath string
	Terms            []int
	IndexFile        *os.File
	DocTermCountDict map[int]int // docID -> termCount (jumlah term di dalam document)

	CurrTermPosition int
}

func NewInvertedIndex(index_name, directoryName string) *InvertedIndex {
	return &InvertedIndex{
		IndexName:        index_name,
		DirName:          directoryName,
		PostingMetadata:  make(map[int][]int),
		IndexFilePath:    directoryName + "/" + index_name + ".index",
		MetadataFilePath: directoryName + "/" + index_name + ".metadata",
		Terms:            []int{},
		DocTermCountDict: make(map[int]int),
		CurrTermPosition: 0,
	}
}

func (Idx *InvertedIndex) OpenWriter() error {
	file, err := os.OpenFile(Idx.IndexFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	Idx.IndexFile = file
	return nil
}

func (Idx *InvertedIndex) Close() error {
	if Idx.IndexFile != nil {
		err := Idx.IndexFile.Close()
		if err != nil {
			return err
		}

		metadataFile, err := os.OpenFile(Idx.MetadataFilePath, os.O_RDWR|os.O_CREATE, 0666)
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

		bufferSizeFile, err := os.OpenFile(Idx.DirName+"/"+Idx.IndexName+"_size.metadata", os.O_RDWR|os.O_CREATE, 0666)
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
	file, err := os.OpenFile(Idx.IndexFilePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	Idx.IndexFile = file

	metadataFile, err := os.OpenFile(Idx.MetadataFilePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer metadataFile.Close()

	bufferSizeFile, err := os.OpenFile(Idx.DirName+"/"+Idx.IndexName+"_size.metadata", os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer bufferSizeFile.Close()

	buf := make([]byte, 100)
	_, err = bufferSizeFile.Read(buf)
	if err != nil {
		return err
	}
	approxBufferSize := int(math.Float64frombits((binary.LittleEndian.Uint64(buf))))

	buf = make([]byte, approxBufferSize)
	_, err = metadataFile.Read(buf)
	if err != nil {
		return err
	}
	Idx.DeserializeMetadata(buf)

	return nil
}

func (Idx *InvertedIndex) GetPostingList(termID int) ([]int, error) {
	postingMetadata, ok := Idx.PostingMetadata[termID]
	if !ok {
		return []int{}, errors.New("termID not found")
	}
	startPositionInIndexFile := int64(postingMetadata[0])
	Idx.IndexFile.Seek(startPositionInIndexFile, 0)
	buf := make([]byte, postingMetadata[2])
	_, err := Idx.IndexFile.Read(buf)
	if err != nil {
		return []int{}, err
	}
	postingList := DecodePostingList(buf)
	return postingList, nil
}

func (Idx *InvertedIndex) AppendPostingList(termID int, postingList []int) error {
	encodedPostingList := EncodePostingList(postingList)
	startPositionInIndexFile, err := Idx.IndexFile.Seek(0, 2)
	if err != nil {
		return err
	}
	lengthInBytesOfPostingList, err := Idx.IndexFile.Write(encodedPostingList)
	if err != nil {
		return err
	}
	Idx.Terms = append(Idx.Terms, termID)
	Idx.PostingMetadata[termID] = []int{int(startPositionInIndexFile), len(postingList), lengthInBytesOfPostingList}
	return nil
}

type IndexIteratorItem struct {
	TermID   int
	TermSize int
}

func NewIndexIteratorItem(termID int, termSize int) IndexIteratorItem {
	return IndexIteratorItem{
		TermID:   termID,
		TermSize: termSize,
	}
}

func (Idx *InvertedIndex) IterateInvertedIndex() iter.Seq2[IndexIteratorItem, []int] {
	return func(yield func(IndexIteratorItem, []int) bool) {
		for Idx.CurrTermPosition < len(Idx.Terms) {
			termID := Idx.Terms[Idx.CurrTermPosition]
			Idx.CurrTermPosition += 1
			startPosition, _, lengthInBytes := Idx.PostingMetadata[termID][0], Idx.PostingMetadata[termID][1], Idx.PostingMetadata[termID][2]
			Idx.IndexFile.Seek(int64(startPosition), 0)
			buf := make([]byte, lengthInBytes)
			_, err := Idx.IndexFile.Read(buf)
			if err != nil {
				return
			}
			postingList := DecodePostingList(buf)
			item := NewIndexIteratorItem(termID, len(Idx.Terms)+1)

			if !yield(item, postingList) {
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
	err = os.Remove(Idx.IndexFilePath)
	if err != nil {
		return err
	}
	err = os.Remove(Idx.MetadataFilePath)
	if err != nil {
		return err
	}
	return nil
}

func (Idx *InvertedIndex) GetAproximateMetadataBufferSize() int {
	allLen := 4 * 3 // 4 byte* 3
	termsSize := 4 * len(Idx.Terms)
	postingMetadata := 4 * 4 * len(Idx.PostingMetadata)
	docTermCountDict := 4 * 3 * len(Idx.DocTermCountDict)
	return allLen + termsSize + postingMetadata + docTermCountDict + 2
}

func (Idx *InvertedIndex) SerializeMetadata() []byte {
	approxBufferSize := Idx.GetAproximateMetadataBufferSize()
	buf := make([]byte, approxBufferSize)
	leftPos := 0

	binary.LittleEndian.PutUint32(buf[leftPos:], uint32(len(Idx.Terms)))
	leftPos += 4 // 32 bit

	binary.LittleEndian.PutUint32(buf[leftPos:], uint32(len(Idx.PostingMetadata)))
	leftPos += 4 // 32 bit

	binary.LittleEndian.PutUint32(buf[leftPos:], uint32(len(Idx.DocTermCountDict)))
	leftPos += 4 // 32 bit

	for _, term := range Idx.Terms {
		// kita pakai uint32bit untuk menyimpan term

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(term))
		leftPos += 4 // 32 bit
	}

	for term, val := range Idx.PostingMetadata {

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

	for docID, termCount := range Idx.DocTermCountDict {
		// docID = 4 byte, termCount = 4 byte

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(docID))
		leftPos += 4 // 32 bit

		binary.LittleEndian.PutUint32(buf[leftPos:], uint32(termCount))
		leftPos += 4 // 32 bit
	}

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

	Idx.Terms = make([]int, termCount)
	Idx.PostingMetadata = make(map[int][]int)
	Idx.DocTermCountDict = make(map[int]int)

	for i := 0; i < termCount; i++ {

		term := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4
		Idx.Terms[i] = term
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

		Idx.PostingMetadata[term] = []int{startPositionInIndexFile, lenPostingList, lengthInBytesOfPostingLists}
	}

	for i := 0; i < docTermCountDictCount; i++ {

		docID := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4

		termCount := int(binary.LittleEndian.Uint32(buf[leftPos:]))
		leftPos += 4

		Idx.DocTermCountDict[docID] = termCount
	}
}

type SaveMetadata struct {
	PostingMeta  map[int][]int
	Terms        []int
	DocWordCount map[int]int
}

func NewSaveMetadata(postings map[int][]int, docWordCount map[int]int, terms []int) SaveMetadata {
	return SaveMetadata{
		postings, terms, docWordCount,
	}
}
