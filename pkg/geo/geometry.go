package geo

import (
	"math"

	"github.com/golang/geo/s2"
)

const (
	earthRadiusKM = 6371.0
)

type BoundingBox struct {
	min, max []float64 // lat, lon
}

func (bb *BoundingBox) GetMin() []float64 {
	return bb.min
}

func (bb *BoundingBox) GetMax() []float64 {
	return bb.max
}

func NewBoundingBox(lats, lons []float64) BoundingBox {
	min, max := []float64{lats[0], lons[0]}, []float64{lats[0], lons[0]}
	for i := 1; i < len(lats); i++ {
		if lats[i] < min[0] {
			min[0] = lats[i]
		}
		if lats[i] > max[0] {
			max[0] = lats[i]
		}
		if lons[i] < min[1] {
			min[1] = lons[i]
		}
		if lons[i] > max[1] {
			max[1] = lons[i]
		}
	}
	return BoundingBox{
		min: min,
		max: max,
	}
}

func (bb *BoundingBox) Contains(lat, lon float64) bool {
	if lat < bb.min[0] || lat > bb.max[0] {
		return false
	}
	if lon < bb.min[1] || lon > bb.max[1] {
		return false
	}
	return true
}

func (bb *BoundingBox) PointsContains(lats, lons []float64) bool {
	for i := 0; i < len(lats); i++ {
		if !bb.Contains(lats[i], lons[i]) {
			return false
		}
	}
	return true
}

// https://www.movable-type.co.uk/scripts/latlong.html
func MidPoint(lat1, lon1 float64, lat2, lon2 float64) (float64, float64) {
	p1LatRad := degToRad(lat1)
	p2LatRad := degToRad(lat2)

	diffLon := degToRad(lon2 - lon1)

	bx := math.Cos(p2LatRad) * math.Cos(diffLon)
	by := math.Cos(p2LatRad) * math.Sin(diffLon)

	newLon := degToRad(lon1) + math.Atan2(by, math.Cos(p1LatRad)+bx)
	newLat := math.Atan2(math.Sin(p1LatRad)+math.Sin(p2LatRad), math.Sqrt((math.Cos(p1LatRad)+bx)*(math.Cos(p1LatRad)+bx)+by*by))

	return radToDeg(newLat), radToDeg(newLon)
}

func degToRad(d float64) float64 {
	return d * math.Pi / 180.0
}

func radToDeg(r float64) float64 {
	return 180.0 * r / math.Pi
}

func crossProduct(hLat, hLon, tLat, tLon, qLat, qLon float64) float64 {
	return ((tLon - hLon) * (qLat - hLat)) - ((qLon - hLon) * (tLat - hLat))
}

func isPointOnSegment(pLat, pLon, aLat, aLon, bLat, bLon float64) bool {
	if pLon >= math.Min(aLon, bLon) && pLon <= math.Max(aLon, bLon) &&
		pLat >= math.Min(aLat, bLat) &&
		pLat <= math.Max(aLat, bLat) {
		return true
	} else {
		return false
	}
}

func windingNumber(pLat, pLon float64, polygonLat, polygonLon []float64) (wn int) {

	for i := range polygonLat[:len(polygonLon)-1] {
		if isPointOnSegment(pLat, pLon, polygonLat[i], polygonLon[i], polygonLat[i+1], polygonLon[i+1]) {
			wn = 1
			return
		}
		if polygonLat[i] <= pLat {
			if polygonLat[i+1] > pLat &&
				crossProduct(polygonLat[i], polygonLon[i], polygonLat[i+1], polygonLon[i+1], pLat, pLon) > 0 {
				wn++
			}
		} else if polygonLat[i+1] <= pLat &&
			crossProduct(polygonLat[i], polygonLon[i], polygonLat[i+1], polygonLon[i+1], pLat, pLon) < 0 {
			wn--
		}
	}
	return
}

func IsPointInPolygon(pLat, pLon float64, polygonLat, polygonLon []float64) bool {
	return windingNumber(pLat, pLon, polygonLat, polygonLon) != 0
}

// Given a start point, initial bearing, and distance, this will calculate the destina­tion point and final bearing travelling along a (shortest distance) great circle arc.
// GetDestinationPoint returns the destination point given the starting point, bearing and distance
// dist in km
func GetDestinationPoint(lat1, lon1 float64, bearing float64, dist float64) (float64, float64) {

	dr := dist / earthRadiusKM

	bearing = (bearing * (math.Pi / 180.0))

	lat1 = (lat1 * (math.Pi / 180.0))
	lon1 = (lon1 * (math.Pi / 180.0))

	lat2Part1 := math.Sin(lat1) * math.Cos(dr)
	lat2Part2 := math.Cos(lat1) * math.Sin(dr) * math.Cos(bearing)

	lat2 := math.Asin(lat2Part1 + lat2Part2)

	lon2Part1 := math.Sin(bearing) * math.Sin(dr) * math.Cos(lat1)
	lon2Part2 := math.Cos(dr) - (math.Sin(lat1) * math.Sin(lat2))

	lon2 := lon1 + math.Atan2(lon2Part1, lon2Part2)
	lon2 = math.Mod((lon2+3*math.Pi), (2*math.Pi)) - math.Pi

	lat2 = lat2 * (180.0 / math.Pi)
	lon2 = lon2 * (180.0 / math.Pi)

	return lat2, lon2
}

// TODO: Geofence pake circle

type Coordinate struct {
	Lat float64
	Lon float64
}

func NewCoordinate(lat, lon float64) Coordinate {
	return Coordinate{
		Lat: lat,
		Lon: lon,
	}
}

func ProjectPointToLineCoord(nearestStPoint Coordinate, secondNearestStPoint Coordinate,
	snap Coordinate) Coordinate {
	nearestStPoint = nearestStPoint
	secondNearestStPoint = secondNearestStPoint
	snapLat := snap.Lat
	snapLon := snap.Lon

	nearestStS2 := s2.PointFromLatLng(s2.LatLngFromDegrees(nearestStPoint.Lat, nearestStPoint.Lon))
	secondNearestStS2 := s2.PointFromLatLng(s2.LatLngFromDegrees(secondNearestStPoint.Lat, secondNearestStPoint.Lon))
	snapS2 := s2.PointFromLatLng(s2.LatLngFromDegrees(snapLat, snapLon))
	projection := s2.Project(snapS2, nearestStS2, secondNearestStS2)
	projectLatLng := s2.LatLngFromPoint(projection)
	return NewCoordinate(projectLatLng.Lat.Degrees(), projectLatLng.Lng.Degrees())
}
