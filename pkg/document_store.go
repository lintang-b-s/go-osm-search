package pkg

// ref: https://tangdh.life/posts/lucene/how-lucene-store-storedfields/   , https://www.youtube.com/watch?v=T5RmMNDR5XI&t=3261s
// idk gakpaham
// lucene & elasticsearch simpan per field dokumennya ke file di buat perblock 16kb isinya 1 field beberapa docs sekaligus
// flush ke disknya pas memory buffer buat simpan docs penuh (16kb). flush nya di background..
// di punyaku semua doc ukurannya statis, so simpan langsung beberapa dokumen perblock 16kb?

const (
	MAX_BUFFER_SIZE = 16 * 1024
)

type DiskWriterReaderI interface {
	WriteUVarint(n uint64) int
	WriteFloat64(n float64)
	Write32Bytes(data [32]byte)
	Write64Bytes(data [64]byte)
	Write128Bytes(data [128]byte)
	ReadBytes(offset int, size int) []byte
	Flush() (int, error)
	ReadAt(offset int, fieldBytesSize int) error
	ReadUVarint(bytesOffset int) (uint64, int)
	ReadUint64(bytesOffset int) (uint64, int)
	ReadFloat64(bytesOffset int) (float64, int)
	Read() (int, error)
	ResetFileSeek()
	Paddingblock()
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

func (d *DocumentStore) Flush(n int) error {
	d.DiskWriterReader.Paddingblock()
	_, err := d.DiskWriterReader.Flush()
	return err
}

func (d *DocumentStore) ReadDoc(offset int) (Node, int) {
	id, bytesWritten := d.DiskWriterReader.ReadUVarint(offset)
	offset += bytesWritten
	name := d.DiskWriterReader.ReadBytes(offset, 64)
	offset += 64
	lat, bytesWritten := d.DiskWriterReader.ReadFloat64(offset)
	offset += bytesWritten
	lon, bytesWritten := d.DiskWriterReader.ReadFloat64(offset)
	offset += bytesWritten
	address := d.DiskWriterReader.ReadBytes(offset, 128)
	offset += 128
	tipe := d.DiskWriterReader.ReadBytes(offset, 64)
	offset += 64
	city := d.DiskWriterReader.ReadBytes(offset, 32)
	offset += 32
	newNode := NewNode(int(id), string(name), lat, lon,
		string(address), string(tipe), string(city))
	return newNode, offset
}

func (d *DocumentStore) WriteDocs(docs []Node) {
	d.BlockFirstDocID = append(d.BlockFirstDocID, docs[0].ID)
	for _, doc := range docs {
		if d.IsBufferFull() {
			d.BackgroundWorker.TiggerProcessing(0)
			d.BlockFirstDocID = append(d.BlockFirstDocID, doc.ID)
			lastOffset := d.BlockOffsets[len(d.BlockOffsets)-1]
			d.BlockOffsets = append(d.BlockOffsets, lastOffset+d.DiskWriterReader.BufferSize())
		}
		d.DocOffsetInBlock[doc.ID] = d.DiskWriterReader.BufferSize()
		d.WriteDoc(doc)
	}
}

func (d *DocumentStore) GetDoc(docID int) Node {
	compare := func(a, b int) int {
		return a - b
	}

	blockPos := BinarySearch(d.BlockFirstDocID, docID, compare)
	if blockPos > 0 {
		blockPos-- // return posisi offset block dari docID
	}

	blockOffset := d.BlockOffsets[blockPos]

	node, _ := d.ReadDoc(blockOffset + d.DocOffsetInBlock[docID])
	d.DiskWriterReader.ResetFileSeek()
	return node
}
