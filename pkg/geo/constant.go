package geo

var ValidSearchTags = map[string]bool{
	"amenity":          true,
	"building":         true,
	"sport":            true,
	"tourism":          true,
	"leisure":          true,
	"boundary":         true,
	"landuse":          true,
	"craft":            true,
	"aeroway":          true,
	"historic":         true,
	"residential":      true,
	"railway":          true,
	"shop":             true,
	"junction":         true,
	"route":            true,
	"ferry":            true,
	"highway":          true,
	"motorcar":         true,
	"motor_vehicle":    true,
	"access":           true,
	"industrial":       true,
	"service":          true,
	"healthcare":       true,
	"office":           true,
	"public_transport": true,
	"waterway":         true,
	"water":            true,
	"telecom":          true,
	"power":            true,
	"place":            true,
	"geological":       true,
	"emergency":        true,
	"bulding":          true,
	"aerialway":        true,
	"barrier":          true,
}

var ValidNodeSearchTag = map[string]bool{
	"historic": true,
	"name":     true,
}

var roadTypeMaxSpeed = map[string]int{
	"motorway":       100,
	"motorroad":      90,
	"trunk":          70,
	"motorway_link":  70,
	"trunk_link":     65,
	"primary":        65,
	"primary_link":   60,
	"secondary":      60,
	"secondary_link": 50,
	"tertiary":       50,
	"tertiary_link":  40,
	"unclassified":   40,
	"residential":    30,
	"road":           20,
	"service":        20,
	"track":          15,
	"living_street":  5,
}

const (
	ROAD_PRIORITY_KEY = 1
)
