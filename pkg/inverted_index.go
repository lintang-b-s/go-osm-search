package pkg

import (
	"bytes"
	"encoding/gob"
	"os"
	"sort"
	"strconv"

	"github.com/RadhiFadlillah/go-sastrawi"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

// https://nlp.stanford.edu/IR-book/pdf/04const.pdf (4.3 Single-pass in-memory indexing)
type DynamicIndex struct {
	TermIDMap                 IDMap
	IntermediateIndices       []string
	InMemoryIndices           map[int][]int
	MaxDynamicPostingListSize int
	DocWordCount              map[int]int
	OutputDir                 string
	DocsCount                 int
	KV                        InvertedIDXDB
}
type InvertedIDXDB interface {
	SaveNodes(nodes []Node) error
	GetNode(id int) (Node, error)
}

func NewDynamicIndex(outputDir string, maxPostingListSize int, kv InvertedIDXDB,
	server bool) (*DynamicIndex, error) {
	idx := &DynamicIndex{
		TermIDMap:                 NewIDMap(),
		IntermediateIndices:       []string{},
		InMemoryIndices:           make(map[int][]int),
		MaxDynamicPostingListSize: maxPostingListSize,
		DocWordCount:              make(map[int]int),
		OutputDir:                 outputDir,
		DocsCount:                 0,
		KV:                        kv,
	}
	if server {
		err := idx.LoadMeta()
		if err != nil {
			return nil, err
		}
	}

	return idx, nil
}

var dictionary = sastrawi.DefaultDictionary()
var stemmer = sastrawi.NewStemmer(dictionary)

func (Idx *DynamicIndex) SpimiBatchIndex(ways []OSMWay, onlyOsmNodes []OSMNode, ctr nodeMapContainer,
	tagIDMap IDMap) error {
	searchNodes := []Node{}
	nodeIDX := 0
	bar := progressbar.NewOptions(5,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][2/2]Indexing osm objects..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	bar.Add(1)
	block := 0

	nodeBoundingBox := make(map[string]BoundingBox)

	for _, way := range ways { // ways yang makan banyak memory
		lat := make([]float64, len(way.NodeIDs))
		lon := make([]float64, len(way.NodeIDs))
		for i := 0; i < len(way.NodeIDs); i++ {
			node := way.NodeIDs[i]
			nodeLat := ctr.nodeMap[node].Lat
			nodeLon := ctr.nodeMap[node].Lon
			lat[i] = nodeLat
			lon[i] = nodeLon
		}

		centerLat, centerLon, err := CenterOfPolygonLatLon(lat, lon)
		if err != nil {
			return err
		}
		tagStringMap := make(map[string]string)
		for k, v := range way.TagMap {
			tagStringMap[tagIDMap.GetStr(k)] = tagIDMap.GetStr(v)

		}

		name, address, building, city := GetNameAddressBuildingFromOSMWay(tagStringMap)

		if IsWayDuplicateCheck(name, lat, lon, nodeBoundingBox) {
			// cek duplikat kalo sebelumnya ada way dengan nama sama dan posisi sama dengan way ini.
			continue
		}

		nodeBoundingBox[name] = NewBoundingBox(lat, lon)

		searchNodes = append(searchNodes, NewNode(nodeIDX, name, centerLat,
			centerLon, address, building, city))
		nodeIDX++

		if len(searchNodes) == 240000 {
			err := Idx.SpimiInvert(searchNodes, &block)
			if err != nil {
				return err
			}
			err = Idx.KV.SaveNodes(searchNodes)
			if err != nil {
				return err
			}
			searchNodes = []Node{}
		}
	}
	bar.Add(1)

	for _, node := range onlyOsmNodes {
		tagStringMap := make(map[string]string)
		for k, v := range node.TagMap {
			tagStringMap[tagIDMap.GetStr(k)] = tagIDMap.GetStr(v)
		}
		name, address, building, city := GetNameAddressBuildingFromOSNode(tagStringMap)
		if name == "" {
			continue
		}

		if IsNodeDuplicateCheck(name, node.Lat, node.Lon, nodeBoundingBox) {
			// cek duplikat kalo sebelumnya ada way dengan nama sama dan posisi sama dengan node ini. gak usah set bounding box buat node.
			continue
		}

		searchNodes = append(searchNodes, NewNode(nodeIDX, name, node.Lat,
			node.Lon, address, building, city))
		nodeIDX++
		if len(searchNodes) == 240000 {
			err := Idx.SpimiInvert(searchNodes, &block)
			if err != nil {
				return err
			}
			err = Idx.KV.SaveNodes(searchNodes)
			if err != nil {
				return err
			}
			searchNodes = []Node{}
		}
	}

	Idx.DocsCount = nodeIDX

	bar.Add(1)
	err := Idx.SpimiInvert(searchNodes, &block)
	if err != nil {
		return err
	}
	err = Idx.KV.SaveNodes(searchNodes)
	if err != nil {
		return err
	}
	bar.Add(1)

	mergedIndex := NewInvertedIndex("merged_index", Idx.OutputDir)
	indices := []InvertedIndex{}
	for _, indexID := range Idx.IntermediateIndices {
		index := NewInvertedIndex(indexID, Idx.OutputDir)
		err := index.OpenReader()
		if err != nil {
			return err
		}
		indices = append(indices, *index)
	}
	mergedIndex.OpenWriter()

	err = Idx.Merge(indices, mergedIndex)
	if err != nil {
		return err
	}
	for _, index := range indices {
		err := index.Close()
		if err != nil {
			return err
		}
	}
	err = mergedIndex.Close()
	if err != nil {
		return err
	}
	bar.Add(1)
	return nil
}

func IsWayDuplicateCheck(name string, lats, lon []float64, nodeBoundingBox map[string]BoundingBox) bool {
	prevBB, ok := nodeBoundingBox[name]
	if !ok {
		return false
	}
	contain := prevBB.PointsContains(lats, lon)
	return contain
}

func IsNodeDuplicateCheck(name string, lats, lon float64, nodeBoundingBox map[string]BoundingBox) bool {
	prevBB, ok := nodeBoundingBox[name]
	if !ok {
		return false
	}
	contain := prevBB.Contains(lats, lon)
	return contain
}

func (Idx *DynamicIndex) SpimiIndex(nodes []Node) error {
	block := 0
	Idx.SpimiInvert(nodes, &block)

	mergedIndex := NewInvertedIndex("merged_index", Idx.OutputDir)
	indices := []InvertedIndex{}
	for _, indexID := range Idx.IntermediateIndices {
		index := NewInvertedIndex(indexID, Idx.OutputDir)
		index.OpenReader()
		indices = append(indices, *index)
	}
	mergedIndex.OpenWriter()

	err := Idx.Merge(indices, mergedIndex)
	if err != nil {
		return err
	}
	for _, index := range indices {
		index.Close()
	}
	return nil
}

func (Idx *DynamicIndex) Merge(indices []InvertedIndex, mergedIndex *InvertedIndex) error {
	lastTerm, lastPosting := -1, []int{}
	for output := range heapMergeKArray(indices) {
		currTerm, currPostings := output.TermID, output.Postings

		if currTerm != lastTerm {
			if lastTerm != -1 {
				sort.Ints(lastPosting)
				err := mergedIndex.AppendPostingList(lastTerm, lastPosting)
				if err != nil {
					return err
				}
			}
			lastTerm, lastPosting = currTerm, currPostings
		} else {
			lastPosting = append(lastPosting, currPostings...)
		}

	}

	if lastTerm != -1 {
		sort.Ints(lastPosting)
		err := mergedIndex.AppendPostingList(lastTerm, lastPosting)
		if err != nil {
			return err
		}
	}
	return nil
}

func (Idx *DynamicIndex) GetNode(nodeID int) (Node, error) {
	return Idx.KV.GetNode(nodeID)
}

// https://nlp.stanford.edu/IR-book/pdf/04const.pdf (Figure 4.4 Spimi-invert)
func (Idx *DynamicIndex) SpimiInvert(nodes []Node, block *int) error {
	postingSize := 0

	termToPostingMap := make(map[int][]int)
	tokenStreams := Idx.SpimiParseOSMNodes(nodes) // [pair of termID and nodeID]

	var postingList []int
	for _, termDocPair := range tokenStreams {

		if len(tokenStreams) == 0 {
			continue
		}
		termID, nodeID := termDocPair[0], termDocPair[1]
		if _, ok := termToPostingMap[termID]; ok {
			postingList = termToPostingMap[termID]
		} else {
			postingList = []int{}
			termToPostingMap[termID] = postingList
		}
		postingList = append(postingList, nodeID)
		termToPostingMap[termID] = postingList
		postingSize += 1

		if postingSize >= Idx.MaxDynamicPostingListSize {
			postingSize = 0
			terms := []int{}
			for termID, _ := range termToPostingMap {
				terms = append(terms, termID)
			}
			sort.Ints(terms)
			indexID := "index_" + strconv.Itoa(*block)
			index := NewInvertedIndex(indexID, Idx.OutputDir)
			err := index.OpenWriter()
			if err != nil {
				return err
			}
			Idx.IntermediateIndices = append(Idx.IntermediateIndices, indexID)
			for term := range terms {

				sort.Ints(termToPostingMap[term])
				index.AppendPostingList(term, termToPostingMap[term])
			}
			*block += 1
			termToPostingMap = make(map[int][]int)
			index.Close()
		}
	}

	terms := []int{}
	for termID, _ := range termToPostingMap {
		terms = append(terms, termID)
	}
	sort.Ints(terms)
	indexID := "index_" + strconv.Itoa(*block)
	index := NewInvertedIndex(indexID, Idx.OutputDir)
	err := index.OpenWriter()
	if err != nil {
		return err
	}
	Idx.IntermediateIndices = append(Idx.IntermediateIndices, indexID)
	for _, term := range terms {
		sort.Ints(termToPostingMap[term])
		index.AppendPostingList(term, termToPostingMap[term])
	}
	*block += 1
	err = index.Close()
	if err != nil {
		return err
	}
	return nil
}

func (Idx *DynamicIndex) SpimiParseOSMNode(node Node) [][]int {
	termDocPairs := [][]int{}
	soup := string(node.Name[:]) + " " + string(node.Building[:]) + " " + string(node.Address[:])
	if soup == "" {
		return termDocPairs
	}

	words := sastrawi.Tokenize(soup)
	Idx.DocWordCount[node.ID] = len(words)
	for _, word := range words {
		tokenizedWord := stemmer.Stem(word)
		termID := Idx.TermIDMap.GetID(tokenizedWord)
		pair := []int{termID, node.ID}
		termDocPairs = append(termDocPairs, pair)
	}
	return termDocPairs
}

func (Idx *DynamicIndex) SpimiParseOSMNodes(nodes []Node) [][]int {
	termDocPairs := [][]int{}
	for _, node := range nodes {
		termDocPairs = append(termDocPairs, Idx.SpimiParseOSMNode(node)...)
	}
	return termDocPairs
}

type SpimiIndexMetadata struct {
	TermIDMap    IDMap
	DocWordCount map[int]int
	DocsCount    int
}

func NewSpimiIndexMetadata(termIDMap IDMap, docWordCount map[int]int, docsCount int) SpimiIndexMetadata {
	return SpimiIndexMetadata{
		TermIDMap:    termIDMap,
		DocWordCount: docWordCount,
		DocsCount:    docsCount,
	}
}
func (Idx *DynamicIndex) Close() error {
	err := Idx.SaveMeta()
	return err
}

func (Idx *DynamicIndex) SaveMeta() error {
	// save to disk
	SpimiMeta := NewSpimiIndexMetadata(Idx.TermIDMap, Idx.DocWordCount, Idx.DocsCount)
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(SpimiMeta)
	if err != nil {
		return err
	}

	metadataFile, err := os.OpenFile(Idx.OutputDir+"/"+"meta.metadata", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer metadataFile.Close()
	err = metadataFile.Truncate(0)
	if err != nil {
		return err
	}

	_, err = metadataFile.Write(buf.Bytes())

	return err
}

func (Idx *DynamicIndex) LoadMeta() error {
	metadataFile, err := os.OpenFile(Idx.OutputDir+"/"+"meta.metadata", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer metadataFile.Close()
	buf := make([]byte, 1024*1024*40)
	metadataFile.Read(buf)
	save := SpimiIndexMetadata{}
	dec := gob.NewDecoder(bytes.NewReader(buf))
	err = dec.Decode(&save)
	if err != nil {
		return err
	}
	Idx.TermIDMap = save.TermIDMap
	Idx.DocWordCount = save.DocWordCount
	Idx.DocsCount = save.DocsCount
	return nil
}

func (Idx *DynamicIndex) GetOutputDir() string {
	return Idx.OutputDir
}

func (Idx *DynamicIndex) GetDocWordCount() map[int]int {
	return Idx.DocWordCount
}

func (Idx *DynamicIndex) GetDocsCount() int {
	return Idx.DocsCount
}

func (Idx *DynamicIndex) GetTermIDMap() IDMap {
	return Idx.TermIDMap
}
