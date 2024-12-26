package pkg

import (
	"bytes"
	"encoding/gob"
	"strconv"

	"github.com/cockroachdb/pebble"
)

type KVDB struct {
	db *pebble.DB
}

func NewKVDB(db *pebble.DB) *KVDB {
	return &KVDB{db}
}

func (db *KVDB) SaveNodes(nodes []Node) error {
	for _, node := range nodes {
		nodeBytes, err := SerializeNode(node)
		if err != nil {
			return err
		}
		err = db.db.Set([]byte(strconv.Itoa(node.ID)), nodeBytes, pebble.Sync)
		if err != nil {
			return err
		}
	}
	return nil 
}

func (db *KVDB) GetNode(id int) (Node, error) {
	nodeBytes, closer, err := db.db.Get([]byte(strconv.Itoa(id)))
	if err != nil {
		return Node{}, err
	}
	defer closer.Close()
	node, err := DeserializeNode(nodeBytes)
	if err != nil {
		return Node{}, err
	}
	return node, nil
}

func SerializeNode(node Node) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(node)
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func DeserializeNode(buf []byte) (Node, error) {
	node := Node{}
	dec := gob.NewDecoder(bytes.NewReader(buf))
	err := dec.Decode(&node)
	if err != nil {
		return Node{}, err
	}
	return node, nil
}
