package datastructure

// Node model info
// @Description OSM Objects indexed in search engines. taken from object way, nodes from osm that have certain tags.
type Node struct {
	ID              int     `json:"id"`               // ID of osm object
	Name            string  `json:"name"`             // osm object name. from osm tag name
	Lat             float64 `json:"lat"`              // latitude of center point polygon osm way
	Lon             float64 `json:"lon"`              // longitude of center point polygon osm way
	Address         string  `json:"address"`          // from tag addr:city/addr:street/addr:place/dll osm, digabungin pakai koma
	Tipe            string  `json:"type"`             // from value tag amenity / building osm or historic
	ContainWikiData bool    `json:"contain_wikidata"` // true if this node contain wikidata
}

func NewNode(id int, name string, lat float64, lon float64, address string, tipe string, city string, wikiData bool) Node {

	return Node{
		ID:      id,
		Name:    name,
		Lat:     lat,
		Lon:     lon,
		Address: address,
		Tipe:    tipe,
		ContainWikiData: wikiData,
	}
}

type WordCandidate struct {
	CandiateWordID int
	TypoWord       string
	CorrectedWord  string
}

func NewWordCandidate(wordID int, typoWord, correctedWord string) WordCandidate {
	return WordCandidate{
		CandiateWordID: wordID,
		TypoWord:       typoWord,
		CorrectedWord:  correctedWord,
	}
}
