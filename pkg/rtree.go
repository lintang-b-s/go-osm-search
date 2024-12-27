package pkg
// TODO: add reverse geocoding pakai r-tree.
// var (
// 	RtreeMinChildren = 25
// 	RtreeMaxChildren = 50
// 	pointLen         = 0.0001
// )

// type SpatialIndex struct {
// 	rtree *rtreego.Rtree
// }

// func NewSpatialIndex() *SpatialIndex {
// 	return &SpatialIndex{
// 		rtree: rtreego.NewTree(2, RtreeMinChildren, RtreeMaxChildren),
// 	}
// }

// func (s *SpatialIndex) Insert(lat, lon float64, value interface{}) {
// 	s.rtree.Insert(value, rtreego.Point{lat, lon})
// }

// type RtreeNode struct {
// 	where *rtreego.Rect
// 	data  interface{}
// }

// func NewRtreeNode(lat, lon float64, data interface{}) *RtreeNode {
// 	point := rtreego.Point{lat, lon}
// 	where , _:= rtreego.NewRect(point, []float64{pointLen, pointLen})
// 	return &RtreeNode{
// 		where: where,
// 		data:  data,
// 	}
// }
