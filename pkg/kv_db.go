package pkg

import (
	"encoding/binary"
	"math"
	"strconv"

	"github.com/dgraph-io/badger/v4"
)

type KVDB struct {
	db *badger.DB
}

func NewKVDB(db *badger.DB) *KVDB {
	return &KVDB{db}
}

func (db *KVDB) SaveNodes(nodes []Node) error {
	batch := db.db.NewWriteBatch()

	for _, node := range nodes {
		soup := string(node.Name[:]) + " " + string(node.Tipe[:]) + " " + string(node.Address[:])
		if soup == "" {
			continue
		}
		nodeBytes, err := SerializeNode(node)
		if err != nil {
			return err
		}
		err = batch.Set([]byte(strconv.Itoa(node.ID)), nodeBytes)
		if err != nil {
			return err
		}
	}
	err := batch.Flush()
	return err
}

func (db *KVDB) GetNode(id int) (Node, error) {
	txn := db.db.NewTransaction(false)

	badgerItem, err := txn.Get([]byte(strconv.Itoa(id)))
	if err != nil {
		return Node{}, err
	}
	var nodeBytes []byte
	err = badgerItem.Value(func(val []byte) error {
		nodeBytes = append([]byte{}, val...)
		return nil
	})

	if err != nil {
		return Node{}, err
	}
	node, err := DeserializeNode(nodeBytes)
	if err != nil {
		return Node{}, err
	}
	return node, nil
}

func SerializeNode(node Node) ([]byte, error) {
	buf := make([]byte, 350) // aproksimasi 308 byte buat per node, jadi 350 byte aja
	leftPos := 0

	binary.LittleEndian.PutUint32(buf[leftPos:], uint32(node.ID))
	leftPos += 4

	copy(buf[leftPos:leftPos+64], node.Name[:])
	leftPos += 64

	binary.LittleEndian.PutUint64(buf[leftPos:], math.Float64bits(node.Lat))
	leftPos += 8

	binary.LittleEndian.PutUint64(buf[leftPos:], math.Float64bits(node.Lon))
	leftPos += 8

	copy(buf[leftPos:leftPos+128], node.Address[:])
	leftPos += 128

	copy(buf[leftPos:leftPos+64], node.Tipe[:])
	leftPos += 64

	copy(buf[leftPos:leftPos+32], node.City[:])
	leftPos += 32

	return buf[:leftPos], nil
}

func DeserializeNode(buf []byte) (Node, error) {
	node := Node{}
	leftPos := 0

	node.ID = int(binary.LittleEndian.Uint32(buf[leftPos:]))
	leftPos += 4

	copy(node.Name[:], buf[leftPos:leftPos+64])
	leftPos += 64

	node.Lat = math.Float64frombits(binary.LittleEndian.Uint64(buf[leftPos:]))
	leftPos += 8

	node.Lon = math.Float64frombits(binary.LittleEndian.Uint64(buf[leftPos:]))
	leftPos += 8

	copy(node.Address[:], buf[leftPos:leftPos+128])
	leftPos += 128

	copy(node.Tipe[:], buf[leftPos:leftPos+64])
	leftPos += 64

	copy(node.City[:], buf[leftPos:leftPos+32])
	leftPos += 32

	return node, nil
}
