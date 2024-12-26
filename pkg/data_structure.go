package pkg

// Node ya tempat/jalan yang ada di osm. yang di index = nama + alamat + building
type Node struct {
	ID       int      `json:"id"`
	Name     [64]byte `json:"name"` // dari tag name osm // 8byte * 64 = 64 character
	Lat      float64  `json:"lat"`  // dari center node polygon way atau langsung coordinate dari osm node
	Lon      float64  `json:"lon"`
	Address  [128]byte  `json:"address"`  // dari tag addr:city/addr:street/addr:place/dll osm, digabungin pakai koma // 128 karakter
	Building [64]byte   `json:"building"` // dari value tag amenity / building osm atau historic kalau node
} // aproksimasi buffer size = 64 + 64 + 64 + 64 + 128 + 64 = 448 bit

func NewNode(id int, name string, lat float64, lon float64, address string, building string) Node {
	var nameB [64]byte
	copy(nameB[:], name)
	var addressB [128]byte
	copy(addressB[:], address)
	var buildingB [64]byte
	copy(buildingB[:], building)
	return Node{
		ID:       id,
		Name:     nameB,
		Lat:      lat,
		Lon:      lon,
		Address:  addressB,
		Building: buildingB,
	}
}
