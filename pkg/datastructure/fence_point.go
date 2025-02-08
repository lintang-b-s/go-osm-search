package datastructure

type FencePoint struct {
	ID  uint32
	Lat float64
	Lon float64
}

func NewFencePoint(id uint32, lat, lon float64) *FencePoint {
	return &FencePoint{
		ID:  id,
		Lat: lat,
		Lon: lon,
	}
}
