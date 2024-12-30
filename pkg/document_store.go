package pkg

import (
	"os"
	"sync"
)

// ref: https://tangdh.life/posts/lucene/how-lucene-store-storedfields/   , https://www.youtube.com/watch?v=T5RmMNDR5XI&t=3261s
// idk gakpaham

const (
	MAX_BUFFER_SIZE            = 16 * 1024
	DOCUMENT_METADATA_FILENAME = "docs_store.fdm"
)

type DiskWriterReaderI interface {
	WriteUVarint(n uint64) int
	CheckUVarintSize(n uint64) int
	WriteFloat64(n float64)
	Write32Bytes(data [32]byte)
	Write64Bytes(data [64]byte)
	Write128Bytes(data [128]byte)
	ReadBytes(offset int, size int) ([]byte, error)
	Flush(bufferSize int) (int, error)
	ReadUVarint(int) (uint64, int, error)
	GetIsBlockReseted() bool
	ReadUint64(int) (uint64, int, error)
	ReadFloat64(bytesOffset int) (float64, int, error)
	SkipToBlock(blockPos int) error
	GetCurrBlockPos() int
	Paddingblock()
	LockBuffer()
	UnlockBuffer()
	BufferSize() int
	Close() error
}

type DocumentStore struct {
	DiskWriterReader DiskWriterReaderI
	OutputDir        string
	BlockFirstDocID  []int
	BlockOffsets     []int
	DocOffsetInBlock map[int]int // docID -> offset in block (bukan offset di file/seluruh block)
	BackgroundWorker *BackgroundWorker[int, error]
}

func NewDocumentStore(diskIO DiskWriterReaderI, out string) *DocumentStore {
	ds := &DocumentStore{
		DiskWriterReader: diskIO,
		OutputDir:        out,
		BlockFirstDocID:  make([]int, 0),
		BlockOffsets:     []int{0},
		DocOffsetInBlock: make(map[int]int),
	}
	ds.BackgroundWorker = NewBackgroundWorker[int, error](1, 1, ds.Flush)
	ds.BackgroundWorker.Start()
	return ds
}

func (d *DocumentStore) WriteDoc(node Node) {

	d.DiskWriterReader.WriteUVarint(uint64(node.ID))
	d.DiskWriterReader.Write64Bytes(node.Name)
	d.DiskWriterReader.WriteFloat64(node.Lat)
	d.DiskWriterReader.WriteFloat64(node.Lon)
	d.DiskWriterReader.Write128Bytes(node.Address)
	d.DiskWriterReader.Write64Bytes(node.Tipe)
	d.DiskWriterReader.Write32Bytes(node.City)
}

func (d *DocumentStore) IsBufferFull() bool {
	bufferMaxSize := (float64(MAX_BUFFER_SIZE) * 0.8)
	return d.DiskWriterReader.BufferSize() >= int(bufferMaxSize)
}

var mu sync.Mutex

func (d *DocumentStore) Flush(n int) error {
	// d.DiskWriterReader.Paddingblock()
	// mu.Lock()
	lastOffset := d.BlockOffsets[len(d.BlockOffsets)-1]
	d.BlockOffsets = append(d.BlockOffsets, lastOffset+MAX_BUFFER_SIZE)

	_, err := d.DiskWriterReader.Flush(MAX_BUFFER_SIZE)
	// mu.Unlock()
	return err
}

func (d *DocumentStore) ReadDoc(offset int) (Node, int, error) {
	id, bytesWritten, err := d.DiskWriterReader.ReadUVarint(offset)
	if err != nil {
		return Node{}, offset, err
	}
	offset += bytesWritten
	name, err := d.DiskWriterReader.ReadBytes(offset, 64)
	if err != nil {
		return Node{}, offset, err
	}
	offset += 64
	lat, bytesWritten, err := d.DiskWriterReader.ReadFloat64(offset)
	if err != nil {
		return Node{}, offset, err
	}
	offset += bytesWritten
	lon, bytesWritten, err := d.DiskWriterReader.ReadFloat64(offset)
	if err != nil {
		return Node{}, offset, err
	}
	offset += bytesWritten
	address, err := d.DiskWriterReader.ReadBytes(offset, 128)
	if err != nil {
		return Node{}, offset, err
	}
	offset += 128
	tipe, err := d.DiskWriterReader.ReadBytes(offset, 64)
	if err != nil {
		return Node{}, offset, err
	}
	offset += 64
	city, err := d.DiskWriterReader.ReadBytes(offset, 32)
	if err != nil {
		return Node{}, offset, err
	}
	offset += 32
	newNode := NewNode(int(id), string(name), lat, lon,
		string(address), string(tipe), string(city))
	return newNode, offset, nil
}

func (d *DocumentStore) WriteDocs(docs []Node) {
	d.BlockFirstDocID = append(d.BlockFirstDocID, docs[0].ID)
	lastDocID := 0
	for _, doc := range docs {
		if d.IsBufferFull() {
			// d.BackgroundWorker.TiggerProcessing(0)
			d.Flush(MAX_BUFFER_SIZE)
			d.BlockFirstDocID = append(d.BlockFirstDocID, doc.ID)
		}
		// d.DiskWriterReader.LockBuffer()
		d.DocOffsetInBlock[doc.ID] = d.DiskWriterReader.BufferSize()
		d.WriteDoc(doc)
		// d.DiskWriterReader.UnlockBuffer()
		lastDocID = doc.ID
	}
	d.Flush(MAX_BUFFER_SIZE)
	d.BlockFirstDocID = append(d.BlockFirstDocID, lastDocID)
}

func (d *DocumentStore) GetDoc(docID int) (Node, error) {
	compare := func(a, b int) int {
		return a - b
	}

	blockPos := BinarySearch[int](d.BlockFirstDocID, docID, compare)
	if blockPos > 0 {
		blockPos-- // return posisi offset block dari docID
	}

	// blockOffset := d.BlockOffsets[blockPos]

	// node, _, err := d.ReadDoc(blockOffset + d.DocOffsetInBlock[docID])
	// if err != nil {
	// 	return Node{}, err
	// }

	if d.DiskWriterReader.GetCurrBlockPos() != blockPos {
		if err := d.DiskWriterReader.SkipToBlock(blockPos); err != nil {
			return Node{}, err
		}

	}

	node, _, err := d.ReadDoc(d.DocOffsetInBlock[docID])
	if err != nil {
		return Node{}, err
	}

	return node, nil
}

func (d *DocumentStore) SaveMeta() error {
	metaFile, err := os.OpenFile(d.OutputDir+"/"+DOCUMENT_METADATA_FILENAME, os.O_CREATE|os.O_RDWR, 0666)

	if err != nil {
		return err
	}
	defer metaFile.Close()

	metaDiskIO := NewDiskWriterReader(make([]byte, 0), metaFile)
	defer metaDiskIO.Close()

	metaDiskIO.WriteUVarint(uint64(len(d.BlockFirstDocID)))
	for _, docID := range d.BlockFirstDocID {
		// gakbisa kalo dibuat perblock  16kb metadatanya
		metaDiskIO.WriteUVarint(uint64(docID))
	}
	metaDiskIO.WriteUVarint(uint64(len(d.BlockOffsets)))
	for _, offset := range d.BlockOffsets {
		metaDiskIO.WriteUVarint(uint64(offset))
	}
	metaDiskIO.WriteUVarint(uint64(len(d.DocOffsetInBlock)))
	for docID, offset := range d.DocOffsetInBlock {
		metaDiskIO.WriteUVarint(uint64(docID))
		metaDiskIO.WriteUVarint(uint64(offset))
	}
	_, err = metaDiskIO.Flush(metaDiskIO.BufferSize())
	return err
}

func (d *DocumentStore) LoadMeta() error {
	metaFile, err := os.OpenFile(d.OutputDir+"/"+DOCUMENT_METADATA_FILENAME, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer metaFile.Close()

	metaDiskIO := NewDiskWriterReader(make([]byte, 0), metaFile)
	err = metaDiskIO.PreloadFile()
	if err != nil {
		return err
	}
	defer metaDiskIO.Close()

	offset := 0
	blockFirstDocIDLen, bytesWritten, err := metaDiskIO.ReadUVarint(0)
	if err != nil {
		return err
	}
	offset += bytesWritten
	d.BlockFirstDocID = make([]int, blockFirstDocIDLen)
	for i := 0; i < int(blockFirstDocIDLen); i++ {
		blockIFirstDocID, bytesWritten, err := metaDiskIO.ReadUVarint(offset)
		if err != nil {
			return err
		}
		if metaDiskIO.GetIsBlockReseted() {
			offset = 0
		}
		offset += bytesWritten
		d.BlockFirstDocID[i] = int(blockIFirstDocID)
	}

	blockOffsetsLen, bytesWritten, err := metaDiskIO.ReadUVarint(offset)
	if err != nil {
		return err
	}
	if metaDiskIO.GetIsBlockReseted() {
		offset = 0
	}
	offset += bytesWritten
	d.BlockOffsets = make([]int, blockOffsetsLen)
	for i := 0; i < int(blockOffsetsLen); i++ {
		blockOffset, bytesWritten, err := metaDiskIO.ReadUVarint(offset)
		if err != nil {
			return err
		}
		if metaDiskIO.GetIsBlockReseted() {
			offset = 0
		}
		offset += bytesWritten
		d.BlockOffsets[i] = int(blockOffset)
	}

	docOffsetInBlockLen, bytesWritten, err := metaDiskIO.ReadUVarint(offset)

	if err != nil {
		return err
	}
	if metaDiskIO.GetIsBlockReseted() {
		offset = 0
	}
	offset += bytesWritten
	d.DocOffsetInBlock = make(map[int]int)
	for i := 0; i < int(docOffsetInBlockLen); i++ {
		docID, bytesWritten, err := metaDiskIO.ReadUVarint(offset)
		if err != nil {
			return err
		}
		if metaDiskIO.GetIsBlockReseted() {
			offset = 0
		}
		offset += bytesWritten
		docOffset, bytesWritten, err := metaDiskIO.ReadUVarint(offset)
		if err != nil {
			return err
		}
		if metaDiskIO.GetIsBlockReseted() {
			offset = 0
		}
		offset += bytesWritten
		d.DocOffsetInBlock[int(docID)] = int(docOffset)
	}
	return nil
}

func (d *DocumentStore) Close() error {
	err := d.SaveMeta()
	if err != nil {
		return err
	}
	err = d.DiskWriterReader.Close()
	return err
}
