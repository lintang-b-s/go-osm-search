package pkg

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
