package datastructure

type QueryPoint struct {
	ID  string
	Lat float64
	Lon float64
}

func NewQueryPoint(id string, lat, lon float64) QueryPoint {
	return QueryPoint{
		ID:  id,
		Lat: lat,
		Lon: lon,
	}
}
