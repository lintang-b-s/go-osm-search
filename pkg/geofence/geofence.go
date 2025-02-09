package geofence

import (
	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
)

type FenceStatus int

const (
	INSIDE FenceStatus = iota
	OUTSIDE
	ENTER
	EXIT
	CROSS
)

type FenceStatusObj struct {
	Status FenceStatus
	Fence  datastructure.Circle
}

type GeoFence interface {
	Add(f datastructure.Circle)
	Get(lat, lon float64, oldFencePoint datastructure.FencePoint) []FenceStatusObj
}

type RtreeFence struct {
	rtree      *datastructure.Rtree
	fenceIDMap *pkg.IDMap
	fence      map[string]datastructure.Circle
}

func NewRtreeFence() *RtreeFence {
	return &RtreeFence{
		rtree:      datastructure.NewRtree(2, 25, 50),
		fenceIDMap: pkg.NewIDMap(),
	}
}

func NewGeoFence() GeoFence {
	return NewRtreeFence()
}

func (r *RtreeFence) Add(f datastructure.Circle) {
	circleFenceObj := datastructure.NewOSMObject(r.fenceIDMap.GetID(f.GetKey()), f.GetCenterLat(), f.GetCenterLon(), nil, f.GetBound())
	r.rtree.InsertLeaf(f.GetBound(), circleFenceObj)
	r.fence[f.GetKey()] = f
}

func (r *RtreeFence) Get(lat, lon float64, oldFencePoint datastructure.FencePoint) []FenceStatusObj {
	pBound := datastructure.NewRtreeBoundingBox(2, []float64{lat - 0.0001, lon - 0.0001}, []float64{lat + 0.0001, lon + 0.0001})
	nodes := r.rtree.Search(pBound)

	var results = []FenceStatusObj{}
	for _, node := range nodes {
		fence := r.fence[r.fenceIDMap.GetStr(node.Leaf.ID)]

		var oldFencePointStatus FenceStatus

		if oldFencePoint.Lat != -999 {
			if fence.Contains(oldFencePoint.Lat, oldFencePoint.Lon) {
				oldFencePointStatus = INSIDE
			} else {
				oldFencePointStatus = OUTSIDE
			}
		} else {
			oldFencePointStatus = OUTSIDE
		}

		var currentFencePointStatus FenceStatus

		if fence.Contains(lat, lon) {
			currentFencePointStatus = INSIDE
		} else {
			currentFencePointStatus = OUTSIDE
		}
		results = r.AppendNewFenceStatus(results, oldFencePointStatus, oldFencePoint,
			currentFencePointStatus, fence, lat, lon)
	}

	return results
}

func (r *RtreeFence) AppendNewFenceStatus(results []FenceStatusObj, oldFencePointStatus FenceStatus,
	oldFencePoint datastructure.FencePoint, currentFenceStatus FenceStatus, fence datastructure.Circle,
	currFenceLat, currFenceLon float64) []FenceStatusObj {
	if oldFencePointStatus == INSIDE && currentFenceStatus == INSIDE {
		results = append(results, FenceStatusObj{Status: INSIDE, Fence: fence})
	} else if oldFencePointStatus == INSIDE && currentFenceStatus == OUTSIDE {
		results = append(results, FenceStatusObj{Status: EXIT, Fence: fence})
		results = append(results, FenceStatusObj{Status: OUTSIDE, Fence: fence})
	} else if oldFencePointStatus == OUTSIDE && currentFenceStatus == INSIDE {
		results = append(results, FenceStatusObj{Status: ENTER, Fence: fence})
		results = append(results, FenceStatusObj{Status: INSIDE, Fence: fence})
	} else if oldFencePointStatus == OUTSIDE && currentFenceStatus == OUTSIDE {
		if fence.IsLineCircleIntersect(oldFencePoint.Lat, oldFencePoint.Lon, currFenceLat, currFenceLon) {
			results = append(results, FenceStatusObj{Status: CROSS, Fence: fence})
		} else {
			results = append(results, FenceStatusObj{Status: OUTSIDE, Fence: fence})
		}
	}
	return results
}
