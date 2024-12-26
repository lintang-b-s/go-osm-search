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
}

func NewDynamicIndex(outputDir string, maxPostingListSize int) *DynamicIndex {
	return &DynamicIndex{
		TermIDMap:                 NewIDMap(),
		IntermediateIndices:       []string{},
		InMemoryIndices:           make(map[int][]int),
		MaxDynamicPostingListSize: maxPostingListSize,
		DocWordCount:              make(map[int]int),
		OutputDir:                 outputDir,
		DocsCount:                 0,
	}
}

var dictionary = sastrawi.DefaultDictionary()
var stemmer = sastrawi.NewStemmer(dictionary)

func (Idx *DynamicIndex) SipmiBatchIndex(ways []OSMWay, onlyOsmNodes []OSMNode, ctr nodeMapContainer) error {
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
			tagStringMap[TagIDMap.GetStr(k)] = TagIDMap.GetStr(v)
		}

		name, address, building := GetNameAddressBuildingFromOSMWay(tagStringMap)
		searchNodes = append(searchNodes, NewNode(nodeIDX, name, centerLat,
			centerLon, address, building))
		nodeIDX++

		if nodeIDX%150000 == 0 {
			err := Idx.SipmiInvert(searchNodes, &block)
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
			tagStringMap[TagIDMap.GetStr(k)] = TagIDMap.GetStr(v)
		}
		name, address, building := GetNameAddressBuildingFromOSNode(tagStringMap)
		searchNodes = append(searchNodes, NewNode(nodeIDX, name, node.Lat,
			node.Lon, address, building))
		nodeIDX++
		if nodeIDX%150000 == 0 {
			err := Idx.SipmiInvert(searchNodes, &block)
			if err != nil {
				return err
			}
			searchNodes = []Node{}
		}
	}

	Idx.DocsCount = nodeIDX // harus di simpan di disk

	bar.Add(1)
	err := Idx.SipmiInvert(searchNodes, &block)
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

	err = Idx.Merge(indices, *mergedIndex)
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

func (Idx *DynamicIndex) SipmiIndex(nodes []Node) error {
	block := 0
	Idx.SipmiInvert(nodes, &block)

	mergedIndex := NewInvertedIndex("merged_index", Idx.OutputDir)
	indices := []InvertedIndex{}
	for _, indexID := range Idx.IntermediateIndices {
		index := NewInvertedIndex(indexID, Idx.OutputDir)
		index.OpenReader()
		indices = append(indices, *index)
	}
	mergedIndex.OpenWriter()

	err := Idx.Merge(indices, *mergedIndex)
	if err != nil {
		return err
	}
	for _, index := range indices {
		index.Close()
	}
	return nil
}

func (Idx *DynamicIndex) Merge(indices []InvertedIndex, mergedIndex InvertedIndex) error {
	lastTerm, lastPosting := -1, []int{}
	for output := range heapMergeKArray(indices) {
		currTerm, currPostings := output.TermID, output.Postings

		if currTerm != lastTerm {
			if lastTerm > 0 {
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

	if lastTerm > 0 {
		sort.Ints(lastPosting)
		err := mergedIndex.AppendPostingList(lastTerm, lastPosting)
		if err != nil {
			return err
		}
	}
	return nil
}

func (Idx *DynamicIndex) SipmiInvert(nodes []Node, block *int) error {
	postingSize := 0

	termToPostingMap := make(map[int][]int)
	for _, node := range nodes {
		termDocPairs := Idx.SipmiParseOSMNode(node)
		if len(termDocPairs) == 0 {
			continue
		}
		// termID, nodeID := termDocPair[0], termDocPair[1]
		for _, termDocPair := range termDocPairs {
			termID, nodeID := termDocPair[0], termDocPair[1]
			var postingList []int
			if _, ok := termToPostingMap[termID]; ok {
				postingList = termToPostingMap[termID]
			} else {
				postingList = []int{}
				termToPostingMap[termID] = postingList
			}
			postingList = append(postingList, nodeID)
			termToPostingMap[termID] = postingList
			postingSize += 1
		}

		if postingSize >= 1e8 {
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
	index.Close()
	return nil
}

func (Idx *DynamicIndex) SipmiParseOSMNode(node Node) [][]int {
	termDocPairs := [][]int{}
	soup := node.Name + " " + node.Address + " " + node.Building
	for _, word := range sastrawi.Tokenize(soup) {
		tokenizedWord := stemmer.Stem(word)
		termID := Idx.TermIDMap.GetID(tokenizedWord)
		pair := []int{termID, node.ID}
		termDocPairs = append(termDocPairs, pair)
	}
	return termDocPairs
}

type SipmiIndexMetadata struct {
	TermIDMap    IDMap
	DocWordCount map[int]int
	DocsCount    int
}

func NewSipmiIndexMetadata(termIDMap IDMap, docWordCount map[int]int, docsCount int) SipmiIndexMetadata {
	return SipmiIndexMetadata{
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
	sipmiMeta := NewSipmiIndexMetadata(Idx.TermIDMap, Idx.DocWordCount, Idx.DocsCount)
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(sipmiMeta)
	if err != nil {
		return err
	}

	metadataFile, err := os.OpenFile(Idx.OutputDir+"/"+"meta.metadata", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
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
	buf := make([]byte, 1024*1024*2) // 2mb
	metadataFile.Read(buf)
	save := SipmiIndexMetadata{}
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
