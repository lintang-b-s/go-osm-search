package pkg

// Node ya tempat/jalan yang ada di osm. yang di index = nama + alamat + building
type Node struct {
	ID int `json:"id"`
	Name string `json:"name"` // dari tag name osm
	Lat float64 `json:"lat"` // dari center node polygon way atau langsung coordinate dari osm node
	Lon float64 `json:"lon"`
	Address string `json:"address"` // dari tag addr:city/addr:street/addr:place/dll osm, digabungin pakai koma
	Building string `json:"building"` // dari value tag amenity / building osm atau historic kalau node
}

func NewNode(id int, name string, lat float64, lon float64, address string, building string) Node {
	return Node{
		ID: id,
		Name: name,
		Lat: lat,
		Lon: lon,
		Address: address,
		Building: building,
	}
}