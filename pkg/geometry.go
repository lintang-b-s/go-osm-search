package pkg

import "github.com/im7mortal/UTM"

func CoordinateToUTM(lat float64, lon float64) (float64, float64, int, string, error) {
	easting, northing, zoneNumber, zoneLetter, err := UTM.FromLatLon(40.71435, -74.00597, false)
	return easting, northing, zoneNumber, zoneLetter, err
}

func UTMToCoordinate(easting float64, northing float64, zoneNumber int, zoneLetter string) (float64, float64, error) {
	lat, lon, err := UTM.ToLatLon(easting, northing, zoneNumber, zoneLetter)
	return lat, lon, err
}

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
	// x, y := make([]float64, len(lat)), make([]float64, len(lon))
	// zoneNumber := 0
	// zoneLetter := ""
	// for i := 0; i < len(lat); i++ {
	// 	var easting, northing float64
	// 	var err error
	// 	easting, northing, zoneNumber, zoneLetter, err = CoordinateToUTM(lat[i], lon[i])
	// 	if err != nil {
	// 		return 0, 0, err
	// 	}
	// 	x[i], y[i] = easting, northing
	// }

	// centerX, centerY := getCenterOfPolygon(x, y)
	// centerLat, centerLon, err := UTMToCoordinate(centerX, centerY, zoneNumber, zoneLetter)
	// return centerLat, centerLon, err
	
	x, y := lat, lon
	centerLat, centerLon := getCenterOfPolygon(x, y)
	return centerLat, centerLon, nil
}
