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

// FenceStatusObj. model info
//
//	@Description	all fences containing query points.
type FenceStatusObj struct {
	Status FenceStatus          `json:"fence_status"` // fence - querypoint status
	Fence  datastructure.Circle `json:"fence"`        // fence object
}

type GeoFence interface {
	Add(fenceName string, f datastructure.Circle)
	Get(lat, lon float64, oldQueryPoint datastructure.QueryPoint) []FenceStatusObj
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
		fence:      make(map[string]datastructure.Circle),
	}
}

func NewGeoFence() GeoFence {
	return NewRtreeFence()
}

func (r *RtreeFence) Add(fenceName string, f datastructure.Circle) {
	circleFenceObj := datastructure.NewOSMObject(r.fenceIDMap.GetID(f.GetKey()), f.GetCenterLat(), f.GetCenterLon(), nil, f.GetBound())

	oldFenceNode, _ := r.rtree.FindLeaf(circleFenceObj, r.rtree.Root, 1)
	if oldFenceNode == nil {
		r.rtree.InsertLeaf(f.GetBound(), circleFenceObj, false)
	} else {
		r.rtree.Delete(circleFenceObj)
		r.rtree.InsertLeaf(f.GetBound(), circleFenceObj, false)
	}

	r.fence[f.GetKey()] = f
}

func (r *RtreeFence) Get(lat, lon float64, oldQueryPoint datastructure.QueryPoint) []FenceStatusObj {

	nearbyFences := r.rtree.NearestNeighboursPQ(3, datastructure.NewPoint(lat, lon))

	var results = []FenceStatusObj{}
	for _, node := range nearbyFences {
		fence := r.fence[r.fenceIDMap.GetStr(node.ID)]

		var oldQueryPointStatus FenceStatus

		if oldQueryPoint.Lat != -999 {
			if fence.Contains(oldQueryPoint.Lat, oldQueryPoint.Lon) {
				oldQueryPointStatus = INSIDE
			} else {
				oldQueryPointStatus = OUTSIDE
			}
		} else {
			oldQueryPointStatus = OUTSIDE
		}

		var currentFencePointStatus FenceStatus

		if fence.Contains(lat, lon) {
			currentFencePointStatus = INSIDE
		} else {
			currentFencePointStatus = OUTSIDE
		}
		results = r.appendNewFenceStatus(results, oldQueryPointStatus, oldQueryPoint,
			currentFencePointStatus, fence, lat, lon)
	}

	return results
}

func (r *RtreeFence) appendNewFenceStatus(results []FenceStatusObj, oldQueryPointStatus FenceStatus,
	oldQueryPoint datastructure.QueryPoint, currentFenceStatus FenceStatus, fence datastructure.Circle,
	currFenceLat, currFenceLon float64) []FenceStatusObj {
	if oldQueryPointStatus == INSIDE && currentFenceStatus == INSIDE {
		results = append(results, FenceStatusObj{Status: INSIDE, Fence: fence})
	} else if oldQueryPointStatus == INSIDE && currentFenceStatus == OUTSIDE {
		results = append(results, FenceStatusObj{Status: EXIT, Fence: fence})
		results = append(results, FenceStatusObj{Status: OUTSIDE, Fence: fence})
	} else if oldQueryPointStatus == OUTSIDE && currentFenceStatus == INSIDE {
		results = append(results, FenceStatusObj{Status: ENTER, Fence: fence})
		results = append(results, FenceStatusObj{Status: INSIDE, Fence: fence})
	} else if oldQueryPointStatus == OUTSIDE && currentFenceStatus == OUTSIDE {
		if fence.IsLineCircleIntersect(oldQueryPoint.Lat, oldQueryPoint.Lon, currFenceLat, currFenceLon) {
			results = append(results, FenceStatusObj{Status: CROSS, Fence: fence})
		} else {
			results = append(results, FenceStatusObj{Status: OUTSIDE, Fence: fence})
		}
	}
	return results
}
