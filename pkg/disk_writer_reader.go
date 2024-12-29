package pkg

import (
	"bufio"
	"encoding/binary"
	"math"
	"os"
)

type DiskWriterReader struct {
	Buf    []byte
	File   *os.File
	Offset int
}

func NewDiskWriterReader(buf []byte, f *os.File) *DiskWriterReader {
	return &DiskWriterReader{
		Buf:  buf,
		File: f,
	}
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

func (d *DiskWriterReader) Write128Bytes(data [128]byte, size int) {
	d.Buf = append(d.Buf, data[:]...)
	d.Offset = len(d.Buf)
}

func (d *DiskWriterReader) ReadBytes(offset int, size int) []byte {
	d.ReadAt(offset, size)
	return d.Buf[:size]
}

func (d *DiskWriterReader) Flush() (int, error) {
	writer := bufio.NewWriter(d.File)
	bytesWritten, err := writer.Write(d.Buf)
	if err != nil {
		return 0, err
	}
	err = writer.Flush()
	return bytesWritten, err
}

// ReadAt. read specific field pada specific offset dengan ukuran field nya fieldBytesSize bytes
func (d *DiskWriterReader) ReadAt(offset int, fieldBytesSize int) error {
	_, err := d.File.Seek(int64(offset), 0)
	reader := bufio.NewReader(d.File)

	if err != nil {
		return err
	}

	for i := 0; i < fieldBytesSize; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return err
		}
		d.Buf[i] = b
	}

	return nil
}

func (d *DiskWriterReader) ReadUVarint(bytesOffset int) (uint64, int) {
	_, err := d.File.Seek(int64(bytesOffset), 0)

	buf := make([]byte, 9)

	_, err = d.File.Read(buf)

	if err != nil {
		return 0, 0
	}

	n, bytesWritten := binary.Uvarint(buf)

	return n, bytesWritten
}

func (d *DiskWriterReader) ReadUint64(bytesOffset int) (uint64, int) {
	_, err := d.File.Seek(int64(bytesOffset), 0)

	buf := make([]byte, 8)

	_, err = d.File.Read(buf)

	if err != nil {
		return 0, 0
	}

	n := binary.LittleEndian.Uint64(buf)

	return uint64(n), 8
}

func (d *DiskWriterReader) ReadFloat64(bytesOffset int) (float64, int) {
	ui, bytesWritten := d.ReadUint64(bytesOffset)
	return math.Float64frombits(ui), bytesWritten
}

func (d *DiskWriterReader) Read() (int, error) {
	reader := bufio.NewReader(d.File)
	bytesRead, err := reader.Read(d.Buf)
	return bytesRead, err
}

func (d *DiskWriterReader) Close() error {
	return d.File.Close()
}

func (d *DiskWriterReader) BufferSize() int {
	return len(d.Buf)
}
