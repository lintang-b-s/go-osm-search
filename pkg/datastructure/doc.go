package datastructure

// Node ya tempat/jalan yang ada di osm. yang di index = nama + alamat + building
type Node struct {
	ID      int     `json:"id"`   // 32 bit/ 4 byte
	Name    string  `json:"name"` // dari tag name osm // 8bit * 64 = 64 character
	Lat     float64 `json:"lat"`  // dari center  point polygon  osm way
	Lon     float64 `json:"lon"`
	Address string  `json:"address"` // dari tag addr:city/addr:street/addr:place/dll osm, digabungin pakai koma // 128 karakter
	City    string  `json:"city"`    // dari tag addr:city osm
	Tipe    string  `json:"type"`    // dari value tag amenity / building osm atau historic kalau node
}

func NewNode(id int, name string, lat float64, lon float64, address string, tipe string, city string) Node {

	return Node{
		ID:      id,
		Name:    name,
		Lat:     lat,
		Lon:     lon,
		Address: address,
		Tipe:    tipe,
		City:    city,
	}
}
