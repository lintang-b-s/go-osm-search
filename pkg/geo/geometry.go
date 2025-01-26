package geo

func shoelaceFormula(x []float64, y []float64) float64 {
	var sum float64
	for i := 0; i < len(x)-1; i++ {
		sum += (x[i] * y[i+1]) - (x[i+1] * y[i])
	}
	return 0.5 * sum
}

func getCenterOfPolygon(x []float64, y []float64) (float64, float64) {
	var sumX, sumY float64
	for i := 0; i < len(x)-1; i++ {
		sumX += (x[i] + x[i+1]) * ((x[i] * y[i+1]) - (x[i+1] * y[i]))
		sumY += (y[i] + y[i+1]) * ((x[i] * y[i+1]) - (x[i+1] * y[i]))
	}
	area := shoelaceFormula(x, y)
	return sumX / (6 * area), sumY / (6 * area)
}

func CenterOfPolygonLatLon(lat, lon []float64) (float64, float64, error) {

	x, y := lat, lon
	centerLat, centerLon := getCenterOfPolygon(x, y)
	return centerLat, centerLon, nil
}

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
