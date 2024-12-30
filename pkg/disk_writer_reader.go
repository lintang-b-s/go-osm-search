package pkg

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"sync"
)

type DiskWriterReader struct {
	Buf       []byte
	File      *os.File
	Offset    int
	BufReader *bufio.Reader
	mu        sync.Mutex
}

func NewDiskWriterReader(buf []byte, f *os.File) *DiskWriterReader {
	dw := &DiskWriterReader{
		Buf:  buf,
		File: f,
	}
	dw.BufReader = bufio.NewReader(dw.File)
	return dw
}

/*
WriteUVarint. encode unsigned uint64 ke varint format. then write ke buffer.

varint encode 64 bit integer ke byte slice 1-9 bytes, dengan angka yang lebih kecil cenderung take less space.
ref: https://sqlite.org/src4/doc/trunk/www/varint.wiki
*/
func (d *DiskWriterReader) WriteUVarint(n uint64) int {
	newBuf := binary.AppendUvarint(d.Buf, n)
	d.Offset = len(newBuf)

	d.Buf = newBuf
	return len(newBuf)
}

func (d *DiskWriterReader) WriteFloat64(n float64) {
	ui := math.Float64bits(n)
	d.Buf = append(d.Buf, make([]byte, 8)...)
	binary.LittleEndian.PutUint64(d.Buf[d.Offset:], ui)
	d.Offset = len(d.Buf)

}

func (d *DiskWriterReader) Write32Bytes(data [32]byte) {
	d.Buf = append(d.Buf, data[:]...)
	d.Offset = len(d.Buf)
}

func (d *DiskWriterReader) Write64Bytes(data [64]byte) {
	d.Buf = append(d.Buf, data[:]...)
	d.Offset = len(d.Buf)
}

func (d *DiskWriterReader) Write128Bytes(data [128]byte) {
	d.Buf = append(d.Buf, data[:]...)
	d.Offset = len(d.Buf)
}

func (d *DiskWriterReader) ReadBytes(offset int, size int) ([]byte, error) {
	buf, err := d.ReadAt(offset, size)
	return buf, err
}

func (d *DiskWriterReader) LockBuffer() {
	d.mu.Lock()
}

func (d *DiskWriterReader) UnlockBuffer() {
	d.mu.Unlock()
}

func (d *DiskWriterReader) Flush(bufferSize int) (int, error) {
	// d.LockBuffer()

	buf := make([]byte, bufferSize)
	copy(buf, d.Buf)

	d.Buf = make([]byte, 0, bufferSize)
	d.Offset = 0

	// d.UnlockBuffer()

	paddingSize := bufferSize - len(buf)
	padding := bytes.Repeat([]byte{0}, paddingSize)
	buf = append(buf, padding...)

	writer := bufio.NewWriter(d.File)
	bytesWritten, err := writer.Write(buf)
	if err != nil {
		return 0, err
	}
	err = writer.Flush()

	return bytesWritten, err
}

// ReadAt. read specific field pada specific offset dengan ukuran field nya fieldBytesSize bytes
func (d *DiskWriterReader) ReadAt(offset int, fieldBytesSize int) ([]byte, error) {
	_, err := d.File.Seek(int64(offset), 0)
	if err != nil {
		return []byte{}, err
	}

	d.BufReader.Reset(d.File)
	buf := make([]byte, fieldBytesSize)
	_, err = d.BufReader.Read(buf)
	// d.BufReader.Peek() yang ini salah
	// d.BufReader.
	if err != nil {
		return []byte{}, err
	}

	return buf, nil
}

func (d *DiskWriterReader) ReadUVarint(bytesOffset int) (uint64, int, error) {
	_, err := d.File.Seek(int64(bytesOffset), 0)

	if err != nil {
		return 0, 0, err
	}

	d.BufReader.Reset(d.File)
	buf := make([]byte, 9)
	_, err = d.BufReader.Read(buf) 
	if err != nil {
		return 0, 0, err
	}
	n, bytesWritten := binary.Uvarint(buf)

	return n, bytesWritten, nil
}

func (d *DiskWriterReader) ReadUint64(bytesOffset int) (uint64, int, error) {
	_, err := d.File.Seek(int64(bytesOffset), 0)
	if err != nil {
		return 0, 0, err
	}

	d.BufReader.Reset(d.File)
	buf := make([]byte, 8)
	_, err = d.BufReader.Read(buf)
	if err != nil {
		return 0, 0, err
	}

	return binary.LittleEndian.Uint64(buf), 8, nil
}

func (d *DiskWriterReader) ReadFloat64(bytesOffset int) (float64, int, error) {
	ui, bytesWritten, err := d.ReadUint64(bytesOffset)
	return math.Float64frombits(ui), bytesWritten, err
}

func (d *DiskWriterReader) Close() error {
	return d.File.Close()
}

func (d *DiskWriterReader) BufferSize() int {
	return len(d.Buf)
}

func (d *DiskWriterReader) Paddingblock() {
	paddingSize := MAX_BUFFER_SIZE - len(d.Buf)
	padding := bytes.Repeat([]byte{0}, paddingSize)
	d.Buf = append(d.Buf, padding...)
	d.Offset = len(d.Buf)

}
