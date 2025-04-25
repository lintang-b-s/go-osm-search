package searcher

import (
	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
)

type NgramLM interface {
	GetQueryNgramProbability(queries [][]int, n int) []float64
	PreProcessData(tokenizedDocs [][]string, countThresold int) [][]int
	MakeCountMatrix(data [][]int)
	SaveNGramData() error
	LoadNGramData() error
	SetTermIDMap(termIDMap *pkg.IDMap)
}

type DynamicIndexer interface {
	GetOutputDir() string
	GetWorkingDir() string
	GetDocWordCount() map[int]int
	GetDocsCount() int
	GetTermIDMap() *pkg.IDMap
	GetAverageDocLength() float64
	BuildVocabulary()
	GetOSMFeatureMap() *pkg.IDMap
	IsWikiData(nodeID int) bool
}

type SearcherDocStore interface {
	GetDoc(docID int) (datastructure.Node, error)
}

type InvertedIndexI interface {
	Close() error
	GetPostingList(termID int) ([]int, error)
	GetLenFieldInDoc() map[int]int
	GetAverageFieldLength() float64
}

type RtreeI interface {
	ImprovedNearestNeighbor(p datastructure.Point) datastructure.OSMObject
	Search(bound datastructure.RtreeBoundingBox) []datastructure.RtreeNode
	NearestNeighboursRadiusFilterOSM(k int, offfset int, p datastructure.Point, maxRadius float64, osmFeature int) []datastructure.OSMObject
}
