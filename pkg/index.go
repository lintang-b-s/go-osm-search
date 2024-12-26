package pkg

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"iter"
	"os"
)

type InvertedIndex struct {
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
		err = metadataFile.Truncate(0)
		if err != nil {
			return err
		}

		// metadataBuf := Idx.SerializeMetadata() // keknya ini salah soalnya postingmetadata selalu [0,0,0]
		metadataBuf, err := Idx.SerializeMetadataGob()
		if err != nil {
			return err
		}
		_, err = metadataFile.Write(metadataBuf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (Idx *InvertedIndex) OpenReader() error {
	file, err := os.OpenFile(Idx.IndexFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	Idx.IndexFile = file

	metadataFile, err := os.OpenFile(Idx.MetadataFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	buf := make([]byte, 1024*1024*2) // 2mb
	metadataFile.Read(buf)
	// Idx.DeserializeMetadata(buf)
	Idx.DeserializeMetadataGob(buf)
	return nil
}

func (Idx *InvertedIndex) GetPostingList(termID int) ([]int, error) {
	postingMetadata := Idx.PostingMetadata[termID]
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
			item := NewIndexIteratorItem(termID, len(Idx.Terms))
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

/*
	 SerializeMetadata serializes the metadata of the inverted index. serialize PostingMetadata, Terms, and DocTermCountDict.

		struktur buffer = dari kiri ke kanan=offset dari rightPos
		dari kanan ke kiri = list term, key & value PostingMetadata, key & value DocTermCountDict
		 (salah)
*/
func (Idx *InvertedIndex) SerializeMetadata() []byte {
	buf := make([]byte, 1024*1024*2) // 2mb
	leftPos := 0
	rightPos := len(buf) - 1

	binary.LittleEndian.PutUint16(buf[leftPos:], uint16(len(Idx.Terms)))
	leftPos += 2 // 16 bit

	binary.LittleEndian.PutUint16(buf[leftPos:], uint16(len(Idx.PostingMetadata)))
	leftPos += 2 // 16 bit

	binary.LittleEndian.PutUint16(buf[leftPos:], uint16(len(Idx.DocTermCountDict)))
	leftPos += 2 // 16 bit

	for _, term := range Idx.Terms {
		// kita pakai uint16bit untuk menyimpan term
		offset := rightPos - 2

		binary.LittleEndian.PutUint16(buf[leftPos:], uint16(offset))
		leftPos += 2 // 16 bit

		rightPos -= 2 // 16 bit / 2 byte
		binary.LittleEndian.PutUint16(buf[rightPos:], uint16(term))
	}

	for term, val := range Idx.PostingMetadata {
		// term = 2 byte, setiap posting = 2byte
		offset := rightPos - 2 - 2*len(val) // 2*len(postingList) karena setiap value dari PostingMetadata = 2 byte

		binary.LittleEndian.PutUint16(buf[leftPos:], uint16(offset))
		leftPos += 2 // 16 bit

		rightPos -= 2 // 16 bit / 2 byte
		binary.LittleEndian.PutUint16(buf[rightPos:], uint16(term))

		startPositionInIndexFile := val[0]    // 2 byte
		lenPostingList := val[1]              // 2 byte
		lengthInBytesOfPostingLists := val[2] // 2 byte

		rightPos -= 2
		binary.LittleEndian.PutUint16(buf[rightPos:], uint16(lengthInBytesOfPostingLists))

		rightPos -= 2
		binary.LittleEndian.PutUint16(buf[rightPos:], uint16(lenPostingList))

		rightPos -= 2
		binary.LittleEndian.PutUint16(buf[rightPos:], uint16(startPositionInIndexFile))
	}

	for docID, termCount := range Idx.DocTermCountDict {
		// docID = 2 byte, termCount = 2 byte
		offset := rightPos - 2 - 2

		binary.LittleEndian.PutUint16(buf[leftPos:], uint16(offset))
		leftPos += 2 // 16 bit

		rightPos -= 2 // 16 bit / 2 byte
		binary.LittleEndian.PutUint16(buf[rightPos:], uint16(docID))

		rightPos -= 2
		binary.LittleEndian.PutUint16(buf[rightPos:], uint16(termCount))
	}

	return buf
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
func (Idx *InvertedIndex) SerializeMetadataGob() ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	save := NewSaveMetadata(Idx.PostingMetadata, Idx.DocTermCountDict, Idx.Terms)
	err := enc.Encode(save)
	return buf.Bytes(), err
}

func (Idx *InvertedIndex) DeserializeMetadataGob(data []byte) error {
	// fileSize := fileInfo.Size()

	// data := make([]byte, fileSize)
	// _, err = io.ReadFull(f, data)
	// if err != nil {
	// 	fmt.Println("Error reading file:", err)
	// 	return err
	// }
	save := SaveMetadata{}
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&save)
	Idx.PostingMetadata = save.PostingMeta
	Idx.DocTermCountDict = save.DocWordCount
	Idx.Terms = save.Terms
	return err
}

// DeserializeMetadata. deserialize metadata inverted index (salah)
func (Idx *InvertedIndex) DeserializeMetadata(buf []byte) {
	leftPos := 0

	termCount := int(binary.LittleEndian.Uint16(buf[0:2]))
	leftPos += 2

	PostingMetadatacount := int(binary.LittleEndian.Uint16(buf[2:4]))
	leftPos += 2

	docTermCountDictCount := int(binary.LittleEndian.Uint16(buf[4:6]))
	leftPos += 2

	Idx.Terms = make([]int, termCount)
	Idx.PostingMetadata = make(map[int][]int)
	Idx.DocTermCountDict = make(map[int]int)

	for i := termCount - 1; i >= 0; i-- {
		// dibalik karena urutan terms di buffer dari kanan ke kiri
		offset := int(binary.LittleEndian.Uint16(buf[leftPos : leftPos+2]))
		leftPos += 2

		term := int(binary.LittleEndian.Uint16(buf[offset : offset+2]))
		offset += 2
		Idx.Terms[i] = term
	}

	for i := 0; i < PostingMetadatacount; i++ {
		offset := int(binary.LittleEndian.Uint16(buf[leftPos : leftPos+2]))
		leftPos += 2

		term := int(binary.LittleEndian.Uint16(buf[offset : offset+2]))
		offset += 2

		lengthInBytesOfPostingLists := int(binary.LittleEndian.Uint16(buf[offset : offset+2]))
		offset += 2

		lenPostingList := int(binary.LittleEndian.Uint16(buf[offset : offset+2]))
		offset += 2

		startPositionInIndexFile := int(binary.LittleEndian.Uint16(buf[offset : offset+2]))
		offset += 2

		Idx.PostingMetadata[term] = []int{startPositionInIndexFile, lenPostingList, lengthInBytesOfPostingLists}
	}

	for i := 0; i < docTermCountDictCount; i++ {

		offset := int(binary.LittleEndian.Uint16(buf[leftPos : leftPos+2]))
		leftPos += 2

		docID := int(binary.LittleEndian.Uint16(buf[offset : offset+2]))
		offset += 2

		termCount := int(binary.LittleEndian.Uint16(buf[offset : offset+2]))
		offset += 2

		Idx.DocTermCountDict[docID] = termCount
	}
}
