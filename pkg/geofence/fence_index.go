package geofence

import (
	"errors"
	"fmt"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/kvdb"
)

var (
	ErrFenceNotExists = errors.New("fence not exists")
)

type GeofenceDB interface {
	PutQueryPoint(point datastructure.QueryPoint) error
	GetQueryPoint(id string) (point datastructure.QueryPoint, err error)
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

func (f *FenceIndex) AddFence(name string)error {
	if _, ok := f.fences[name];ok  {
		return pkg.WrapErrorf(errors.New("already exixts"), pkg.ErrBadParamInput, "fence already exists")
	}
	f.fences[name] = NewRtreeFence()
	return nil 
}

func (f *FenceIndex) DeleteFence(name string) {
	delete(f.fences, name)
}

func (f *FenceIndex) GetFence(name string) (GeoFence, bool) {
	fence, ok := f.fences[name]
	return fence, ok
}

func (f *FenceIndex) Search(name string, lat, lon float64, queryPointID string) ([]FenceStatusObj, error) {
	fence, ok := f.fences[name]
	if !ok {
		return []FenceStatusObj{}, pkg.WrapErrorf(ErrFenceNotExists, pkg.ErrBadParamInput, fmt.Sprintf("FenceIndex does not contain fence %s", name))
	}

	fencePoint, err := f.db.GetQueryPoint(queryPointID)
	if err != nil && !errors.Is(err, kvdb.ErrorsKeyNotExists) {
		return []FenceStatusObj{}, pkg.WrapErrorf(ErrFenceNotExists, pkg.ErrBadParamInput, fmt.Sprintf("FenceIndex does not contain queryPoint %s", queryPointID))
	}

	if errors.Is(err, kvdb.ErrorsKeyNotExists) {
		fencePoint.Lat = -999
		fencePoint.Lon = -999
	}

	newQueryPoint := datastructure.NewQueryPoint(queryPointID, lat, lon)
	err = f.db.PutQueryPoint(newQueryPoint)
	if err != nil {
		return []FenceStatusObj{}, err
	}
	newFenceStatus := fence.Get(lat, lon, fencePoint)
	return newFenceStatus, nil
}

func (f *FenceIndex) UpdateFencePoint(name string, lat, lon float64, queryPointID string) error {
	if _, ok := f.fences[name]; !ok {
		return pkg.WrapErrorf(ErrFenceNotExists, pkg.ErrBadParamInput, fmt.Sprintf("FenceIndex does not contain fence %s", name))
	}
	newQueryPoint := datastructure.NewQueryPoint(queryPointID, lat, lon)
	err := f.db.PutQueryPoint(newQueryPoint)
	if err != nil {
		return err
	}
	return nil
}

func (f *FenceIndex) AddFencePoint(name, fencePointName string, lat, lon, radius float64) error {
	if _, ok := f.fences[name]; !ok {
		return pkg.WrapErrorf(ErrFenceNotExists, pkg.ErrBadParamInput, fmt.Sprintf("FenceIndex does not contain fence %s", name))
	}

	circleFence := datastructure.NewCircle(fencePointName, lat, lon, radius)
	f.fences[name].Add(name, circleFence)
	return nil
}
