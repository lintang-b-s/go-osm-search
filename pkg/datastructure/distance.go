package datastructure

import "math"

const (
	earthRadiusKM = 6371.0
)

func havFunction(angleRad float64) float64 {
	return (1 - math.Cos(angleRad)) / 2.0
}

func degreeToRadians(angle float64) float64 {
	return angle * (math.Pi / 180.0)
}

// very slow 
func haversineDistance(latOne, longOne, latTwo, longTwo float64) float64 {
	latOne = degreeToRadians(latOne)
	longOne = degreeToRadians(longOne)
	latTwo = degreeToRadians(latTwo)
	longTwo = degreeToRadians(longTwo)

	dist := 2.0 * math.Asin(math.Sqrt(havFunction(latOne-latTwo)+math.Cos(latOne)*math.Cos(latTwo)*havFunction(longOne-longTwo)))
	return earthRadiusKM * dist
}

func euclideanDistance(latOne, longOne, latTwo, longTwo float64) float64 {
	return math.Sqrt((latOne-latTwo)*(latOne-latTwo) + (longOne-longTwo)*(longOne-longTwo))
}
