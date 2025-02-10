package geofence_di

import (
	"github.com/lintang-b-s/osm-search/pkg/geofence"
	"github.com/lintang-b-s/osm-search/pkg/http/usecases"
	"github.com/lintang-b-s/osm-search/pkg/kvdb"
)

func New(db *kvdb.KVDB) usecases.GeofenceIndex {
	return geofence.NewFenceIndex(db)
}

