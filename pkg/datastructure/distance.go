package datastructure

import "math"

const (
	earthRadiusKM = 6371.0
)

// https://scikit-learn.org/stable/modules/generated/sklearn.metrics.pairwise.haversine_distances.html
// sin^2(a/2)
func havFunction(angleRad float64) float64 {
	return math.Pow(math.Sin(angleRad/2.0), 2)
}

func degreeToRadians(angle float64) float64 {
	return angle * (math.Pi / 180.0)
}

func haversineDistance(latOne, longOne, latTwo, longTwo float64) float64 {
	latOne = degreeToRadians(latOne)
	longOne = degreeToRadians(longOne)
	latTwo = degreeToRadians(latTwo)
	longTwo = degreeToRadians(longTwo)

	dist := 2.0 * math.Asin(math.Sqrt(havFunction(latOne-latTwo)+math.Cos(latOne)*math.Cos(latTwo)*havFunction(longOne-longTwo)))
	return earthRadiusKM * dist
}
