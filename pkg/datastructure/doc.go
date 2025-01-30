package datastructure

// Node model info 
// @Description OSM Objects indexed in search engines. taken from object way, nodes from osm that have certain tags.
// Node ya tempat/jalan yang ada di osm. yang di index = nama + alamat + building
type Node struct {
	ID      int     `json:"id"`   // ID of osm object
	Name    string  `json:"name"` // osm object name. from osm tag name // 8bit * 64 = 64 character
	Lat     float64 `json:"lat"`  // latitude of center point polygon osm way
	Lon     float64 `json:"lon"` // longitude of center point polygon osm way
	Address string  `json:"address"` // from tag addr:city/addr:street/addr:place/dll osm, digabungin pakai koma // 128 karakter
	City    string  `json:"city"`    // from tag addr:city osm
	Tipe    string  `json:"type"`    // from value tag amenity / building osm or historic
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
