package geo

import "math"

const (
	earthRadiusKM = 6371.0
	kRad          = math.Pi / 180.0
)

func havFunction(angle_rad float64) float64 {
	return (1 - math.Cos(angle_rad)) / 2.0
}

func HaversineDistance(latOne, longOne, latTwo, longTwo float64) float64 {

	sqrt_hav_angle := math.Sqrt(havFunction(latOne-latTwo) + math.Cos(latOne)*math.Cos(latTwo)*havFunction(longOne-longTwo))
	central_angle_rad := 2.0 * math.Asin(sqrt_hav_angle)
	return earthRadiusKM * central_angle_rad
}

