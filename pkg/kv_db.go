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
		soup := string(node.Name[:]) + " " + string(node.Address[:]) + " " + string(node.Building[:])
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
	buf := make([]byte, 460) // aproksimasi 448 bit buat per node, jadi 460 bit aja
	leftPos := 0

	binary.LittleEndian.PutUint16(buf[leftPos:], uint16(node.ID))
	leftPos += 2

	copy(buf[leftPos:leftPos+64], node.Name[:])
	leftPos += 64

	binary.LittleEndian.PutUint64(buf[leftPos:], math.Float64bits(node.Lat))
	leftPos += 8

	binary.LittleEndian.PutUint64(buf[leftPos:], math.Float64bits(node.Lon))
	leftPos += 8

	copy(buf[leftPos:leftPos+128], node.Address[:])
	leftPos += 128

	copy(buf[leftPos:leftPos+64], node.Building[:])
	leftPos += 64

	return buf[:leftPos], nil
}

func DeserializeNode(buf []byte) (Node, error) {
	node := Node{}
	leftPos := 0

	node.ID = int(binary.LittleEndian.Uint16(buf[leftPos:]))
	leftPos += 2

	copy(node.Name[:], buf[leftPos:leftPos+64])
	leftPos += 64

	node.Lat = math.Float64frombits(binary.LittleEndian.Uint64(buf[leftPos:]))
	leftPos += 8

	node.Lon = math.Float64frombits(binary.LittleEndian.Uint64(buf[leftPos:]))
	leftPos += 8

	copy(node.Address[:], buf[leftPos:leftPos+128])
	leftPos += 128

	copy(node.Building[:], buf[leftPos:leftPos+64])
	leftPos += 64

	return node, nil
}

// func (db *KVDB) SaveNodes(nodes []Node) error {
// 	batch := db.db.NewBatch()
// 	defer batch.Close()

// 	for _, node := range nodes {
// 		soup := string(node.Name[:]) + " " + string(node.Address[:]) + " " + string(node.Building[:])
// 		if soup == "" {
// 			continue
// 		}
// 		nodeBytes, err := SerializeNode(node)
// 		if err != nil {
// 			return err
// 		}
// 		// err = db.db.Set([]byte(strconv.Itoa(node.ID)), nodeBytes, pebble.Sync)
// 		// if err != nil {
// 		// 	return err
// 		// } lemot banget
// 		err = batch.Set([]byte(strconv.Itoa(node.ID)), nodeBytes, pebble.Sync)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 	}
// 	err := batch.Commit(pebble.Sync)
// 	return err
// }

// func (db *KVDB) GetNode(id int) (Node, error) {
// 	nodeBytes, closer, err := db.db.Get([]byte(strconv.Itoa(id)))
// 	if err != nil {
// 		return Node{}, err
// 	}
// 	defer closer.Close()
// 	node, err := DeserializeNode(nodeBytes)
// 	if err != nil {
// 		return Node{}, err
// 	}
// 	return node, nil
// }
