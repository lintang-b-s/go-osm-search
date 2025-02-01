package geo

import "math"

const earthRadiusKM = 6371.0

func havFunction(angle_rad float64) float64 {
	return (1 - math.Cos(angle_rad)) / 2.0
}

func havFormula(latOne, longOne, latTwo, longTwo float64) float64 {
	var latitude_diff float64 = latOne - latTwo
	var longitude_diff float64 = longOne - longTwo

	var hav_latitude float64 = havFunction(latitude_diff)
	var hav_longitude float64 = havFunction(longitude_diff)

	return hav_latitude + math.Cos(latOne)*math.Cos(latTwo)*hav_longitude
}

func archaversine(hav_angle float64) float64 {
	var sqrt_hav_angle float64 = math.Sqrt(hav_angle)
	return 2.0 * math.Asin(sqrt_hav_angle)
}

func HaversineDistance(latOne, longOne, latTwo, longTwo float64) float64 {
	var hav_central_angle float64 = havFormula(latOne, longOne, latTwo, longTwo)
	var central_angle_rad float64 = archaversine(hav_central_angle)
	return earthRadiusKM * central_angle_rad
}
