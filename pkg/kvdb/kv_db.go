package kvdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/lintang-b-s/osm-search/pkg/datastructure"

	"go.etcd.io/bbolt"
)

var (
	ErrorsKeyNotExists = errors.New("key not exists")
)

const (
	BBOLTDB_BUCKET          = "osmSearch"
	BBOLTDB_GEOFENCE_BUCKET = "geofence"
)

type KVDB struct {
	db *bbolt.DB
	sync.Mutex
}

func NewKVDB(db *bbolt.DB) *KVDB {

	return &KVDB{db,
		sync.Mutex{}}
}

// save osm objects ks boltDB. batching
func (db *KVDB) SaveDocs(nodes []datastructure.Node) error {
	db.Lock()
	defer db.Unlock()
	return db.db.Batch(func(tx *bbolt.Tx) error {
		for _, node := range nodes {
			err := db.Set(node, tx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (db *KVDB) Set(node datastructure.Node, tx *bbolt.Tx) error {

	nodeBytes, err := serializeNode(node)
	if err != nil {
		return err
	}
	b := tx.Bucket([]byte(BBOLTDB_BUCKET))
	err = b.Put([]byte(strconv.Itoa(node.ID)), nodeBytes)
	if err != nil {
		return err
	}
	return nil // harus return nil , kalau return err kena rollback txn-nya
}

func (db *KVDB) SaveDocsNoBatch(nodes []datastructure.Node) error {
	db.Lock()
	defer db.Unlock()
	for _, node := range nodes {
		err := db.setNoBatch(node)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *KVDB) setNoBatch(node datastructure.Node) error {

	return db.db.Update(func(tx *bbolt.Tx) error {
		nodeBytes, err := serializeNode(node)
		if err != nil {
			return err
		}
		b := tx.Bucket([]byte(BBOLTDB_BUCKET))
		err = b.Put([]byte(strconv.Itoa(node.ID)), nodeBytes)
		if err != nil {
			return err
		}
		return nil // harus return nil , kalau return err kena rollback txn-nya
	})
}

func (db *KVDB) GetDoc(id int) (node datastructure.Node, err error) {

	db.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BBOLTDB_BUCKET))
		nodeBytes := b.Get([]byte(strconv.Itoa(id)))
		if nodeBytes == nil {
			err = fmt.Errorf("document with docID: %d not found", id)
			return nil
		}
		node, err = deserializeNode(nodeBytes)
		return nil
	})
	return
}

func (db *KVDB) PutFencePoint(point datastructure.FencePoint) error {
	return db.db.Update(func(tx *bbolt.Tx) error {
		nodeBytes, err := serializePoint(point)
		if err != nil {
			return err
		}
		b := tx.Bucket([]byte(BBOLTDB_GEOFENCE_BUCKET))
		err = b.Put([]byte(strconv.Itoa(int(point.ID))), nodeBytes)
		if err != nil {
			return err
		}
		return nil // harus return nil , kalau return err kena rollback txn-nya
	})
}

func (db *KVDB) GetFencePoint(id uint32) (point datastructure.FencePoint, err error) {

	db.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BBOLTDB_GEOFENCE_BUCKET))
		nodeBytes := b.Get([]byte(strconv.Itoa(int(id))))
		if nodeBytes == nil {
			err = ErrorsKeyNotExists
			return nil
		}
		point, err = deserializePoint(nodeBytes)
		return nil
	})
	return
}

func GetFloat(bb *bytes.Buffer, offset int) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(bb.Bytes()[offset:]))
}

func PutFloat(bb *bytes.Buffer, offset int, val float64) {
	binary.LittleEndian.PutUint64(bb.Bytes()[offset:], math.Float64bits(val))
}

func GetInt(bb *bytes.Buffer, offset int) int {
	return int(binary.LittleEndian.Uint32(bb.Bytes()[offset:]))
}

func GetUint32(bb *bytes.Buffer, offset int) uint32 {
	return binary.LittleEndian.Uint32(bb.Bytes()[offset:])
}

// PutInt. set int ke byte array page di posisi = offset.
func PutInt(bb *bytes.Buffer, offset int, val int) {
	binary.LittleEndian.PutUint32(bb.Bytes()[offset:], uint32(val))
}

func PutUint32(bb *bytes.Buffer, offset int, val uint32) {
	binary.LittleEndian.PutUint32(bb.Bytes()[offset:], val)
}

// GetBytes. return byte array dari byte array page di posisi = offset. di awal ada panjang bytes nya sehingga buat read bytes tinggal baca buffer page[offset+4:offset+4+length]
func GetBytes(bb *bytes.Buffer, offset int) []byte {
	length := GetInt(bb, offset)
	b := make([]byte, GetInt(bb, offset))
	copy(b, bb.Bytes()[offset+4:offset+4+length])
	return b
}

// PutBytes. set byte array ke byte array page di posisi = offset.
func PutBytes(bb *bytes.Buffer, offset int, b []byte) {
	PutInt(bb, offset, len(b))
	copy(bb.Bytes()[offset+4:], b)
}

// GetString. return string dari byte array page di posisi= offset.
func GetString(bb *bytes.Buffer, offset int) string {
	return string(GetBytes(bb, offset))
}

// putString. set string ke byte array page di posisi = offset.
func PutString(bb *bytes.Buffer, offset int, s string) int {
	PutBytes(bb, offset, []byte(s))
	return len([]byte(s))
}

func GetDocSize(doc datastructure.Node) int {
	return 4 + 4 + len([]byte(doc.Name)) + 8 + 8 + 4 + len([]byte(doc.Address)) + 4 + len([]byte(doc.Tipe)) +
		4
}

func serializeNode(node datastructure.Node) ([]byte, error) {

	bb := bytes.NewBuffer(make([]byte, GetDocSize(node)))

	leftPos := 0

	PutInt(bb, leftPos, node.ID)
	leftPos += 4

	stringLen := PutString(bb, leftPos, node.Name)
	leftPos += stringLen + 4

	PutFloat(bb, leftPos, node.Lat)
	leftPos += 8

	PutFloat(bb, leftPos, node.Lon)
	leftPos += 8

	stringLen = PutString(bb, leftPos, node.Address)
	leftPos += stringLen + 4

	stringLen = PutString(bb, leftPos, node.Tipe)
	leftPos += stringLen + 4

	return bb.Bytes(), nil
}

func deserializeNode(buf []byte) (datastructure.Node, error) {
	bb := bytes.NewBuffer(buf)
	node := datastructure.Node{}
	leftPos := 0

	node.ID = GetInt(bb, leftPos)
	leftPos += 4

	node.Name = GetString(bb, leftPos)
	leftPos += len([]byte(node.Name)) + 4 // +4 dari int panjang bytearray dari string

	node.Lat = GetFloat(bb, leftPos)
	leftPos += 8

	node.Lon = GetFloat(bb, leftPos)
	leftPos += 8

	node.Address = GetString(bb, leftPos)
	leftPos += len([]byte(node.Address)) + 4

	node.Tipe = GetString(bb, leftPos)
	leftPos += len([]byte(node.Tipe)) + 4

	return node, nil
}

// serialize deserialize  fence point
func serializePoint(point datastructure.FencePoint) ([]byte, error) {

	bb := bytes.NewBuffer(make([]byte, 20))

	leftPos := 0

	PutUint32(bb, leftPos, point.ID)
	leftPos += 4

	PutFloat(bb, leftPos, point.Lat)
	leftPos += 8

	PutFloat(bb, leftPos, point.Lon)
	leftPos += 8

	return bb.Bytes(), nil
}

func deserializePoint(buf []byte) (datastructure.FencePoint, error) {
	bb := bytes.NewBuffer(buf)
	point := datastructure.FencePoint{}
	leftPos := 0

	point.ID = GetUint32(bb, leftPos)
	leftPos += 4

	point.Lat = GetFloat(bb, leftPos)
	leftPos += 8

	point.Lon = GetFloat(bb, leftPos)
	leftPos += 8

	return point, nil
}
