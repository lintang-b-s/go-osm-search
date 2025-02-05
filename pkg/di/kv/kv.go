package kv_di

import (
	"context"

	"github.com/lintang-b-s/osm-search/pkg/kvdb"

	bolt "go.etcd.io/bbolt"
)

func New(ctx context.Context) (*kvdb.KVDB, error) {
	db, err := bolt.Open("docs_store.db", 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(kvdb.BBOLTDB_BUCKET))
		return err
	})
	if err != nil {
		return nil, err
	}

	bboltKV := kvdb.NewKVDB(db)

	cleanup := func() {
		_ = db.Close()
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		cleanup()
	}()

	return bboltKV, nil
}
