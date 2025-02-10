package datastructure

import (
	"math"
)

type Circle struct {
	key       string  `json:"key"`
	centerLat float64 `json:"center_lat"`
	centerLon float64 `json:"center_lon"`
	radius    float64 `json:"radius"` // in km
}

func (c *Circle) GetKey() string {
	return c.key
}

func (c *Circle) GetCenterLat() float64 {
	return c.centerLat
}

func (c *Circle) GetCenterLon() float64 {
	return c.centerLon
}

func (c *Circle) GetRadius() float64 {
	return c.radius
}

func NewCircle(key string, centerLat, centerLon, radius float64) Circle {
	return Circle{
		key:       key,
		centerLat: centerLat,
		centerLon: centerLon,
		radius:    radius,
	}
}

// is the point (lat, lon) inside the circle?
func (c *Circle) Contains(lat, lon float64) bool {
	return HaversineDistance(c.centerLat, c.centerLon, lat, lon) <= c.radius
}

func projection(pLat, pLon, centerLat float64) (float64, float64) {
	return pLat * earthRadiusM, pLon * earthRadiusM * math.Cos(centerLat)
}

func radToDegree(rad float64) float64 {
	return rad * (180 / math.Pi)
}

// is the line (lat1, lon1) to (lat2, lon2) crossing the circle?
// https://gis.stackexchange.com/questions/36841/line-intersection-with-circle-on-a-sphere-globe-or-earth
func (c *Circle) IsLineCircleIntersect(lat1, lon1, lat2, lon2 float64) bool {
	cLat := degreeToRadians(c.centerLat)
	cLon := degreeToRadians(c.centerLon)

	cRadius := c.radius * 1000

	lat1, lon1 = degreeToRadians(lat1), degreeToRadians(lon1)
	aLat, aLon := projection(lat1, lon1, cLat)

	lat2, lon2 = degreeToRadians(lat2), degreeToRadians(lon2)
	bLat, bLon := projection(lat2, lon2, cLat)

	ccLat, ccLon := projection(cLat, cLon, cLat)

	vLat := aLat - ccLat
	vLon := aLon - ccLon

	uLat := bLat - aLat
	uLon := bLon - aLon

	alpha := uLat*uLat + uLon*uLon

	beta := uLat*vLat + uLon*vLon

	gamma := vLat*vLat + vLon*vLon - cRadius*cRadius

	discriminant := beta*beta - alpha*gamma
	if discriminant < 0 {
		return false
	}
	sqrtDiscriminant := math.Sqrt(discriminant)
	t1 := (-beta + sqrtDiscriminant) / alpha
	t2 := (-beta - sqrtDiscriminant) / alpha

	if t1 >= 0 && t1 <= 1 {
		return true
	}
	if t2 >= 0 && t2 <= 1 {
		return true
	}

	return false
}

func (c *Circle) GetBound() RtreeBoundingBox {
	return NewRtreeBoundingBox(2, []float64{c.centerLat - c.radius, c.centerLon - c.radius},
		[]float64{c.centerLat + c.radius, c.centerLon + c.radius})
}
