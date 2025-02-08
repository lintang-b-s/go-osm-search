package geofence

import (
	"errors"
	"fmt"

	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/kvdb"
)

type GeofenceDB interface {
	PutFencePoint(point datastructure.FencePoint) error
	GetFencePoint(id uint32) (point datastructure.FencePoint, err error)
}

type FenceIndex struct {
	fences map[string]GeoFence
	db     GeofenceDB
}

func NewFenceIndex(db GeofenceDB) *FenceIndex {
	return &FenceIndex{
		fences: make(map[string]GeoFence),
		db:     db,
	}
}

func (f *FenceIndex) AddFence(name string, fence GeoFence) {
	f.fences[name] = fence
}

func (f *FenceIndex) DeleteFence(name string) {
	delete(f.fences, name)
}

func (f *FenceIndex) GetFence(name string) (GeoFence, bool) {
	fence, ok := f.fences[name]
	return fence, ok
}

func (f *FenceIndex) Search(name string, lat, lon float64, fencePointID uint32) ([]FenceStatusObj, error) {
	fence, ok := f.fences[name]
	if !ok {
		return []FenceStatusObj{}, fmt.Errorf("FenceIndex does not contain fence %q", name)
	}

	fencePoint, err := f.db.GetFencePoint(fencePointID)
	if err != nil && !errors.Is(err, kvdb.ErrorsKeyNotExists) {
		return []FenceStatusObj{}, fmt.Errorf("FenceIndex does not contain fence %q", name)
	}

	if errors.Is(err, kvdb.ErrorsKeyNotExists) {
		fencePoint.Lat = -999
		fencePoint.Lon = -999
	}

	return fence.Get(lat, lon, fencePoint), nil
}
