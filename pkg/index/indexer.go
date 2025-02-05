package index

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/lintang-b-s/osm-search/pkg"
	"github.com/lintang-b-s/osm-search/pkg/datastructure"
	"github.com/lintang-b-s/osm-search/pkg/geo"

	"github.com/RadhiFadlillah/go-sastrawi"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	BATCH_SIZE = 100000
)

type SpellCorrectorI interface {
	Preprocessdata(tokenizedDocs [][]string)
	GetWordCandidates(mispelledWord string, editDistance int) ([]int, error)
	GetCorrectQueryCandidates(allPossibleQueryTerms [][]int) [][]int
	GetCorrectSpellingSuggestion(allCorrectQueryCandidates [][]int, originalQueryTermIDs []int) ([]int, error)
	GetMatchedWordBasedOnPrefix(prefixWord string) ([]int, error)
	GetMatchedWordsAutocomplete(allQueryCandidates [][]int, originalQueryTerms []int) ([][]int, error)
}

type DocumentStoreI interface {
	WriteDocs(docs []datastructure.Node)
}

type BboltDBI interface {
	SaveDocs(nodes []datastructure.Node) error
}

// https://nlp.stanford.edu/IR-book/pdf/04const.pdf (4.3 Single-pass in-memory indexing)
type DynamicIndex struct {
	TermIDMap                 *pkg.IDMap
	workingDir                string
	intermediateIndices       []string
	maxDynamicPostingListSize int
	docWordCount              map[int]int
	averageDocLength          float64
	outputDir                 string
	docsCount                 int
	spellCorrectorBuilder     SpellCorrectorI
	IndexedData               IndexedData
	documentStore             BboltDBI //DocumentStoreI
}

type IndexedData struct {
	Ways     []geo.OSMWay
	Nodes    []geo.OSMNode
	Ctr      geo.NodeMapContainer
	TagIDMap *pkg.IDMap
}

func NewIndexedData(ways []geo.OSMWay, nodes []geo.OSMNode, ctr geo.NodeMapContainer, tagIDMap *pkg.IDMap) IndexedData {
	return IndexedData{
		Ways:     ways,
		Nodes:    nodes,
		Ctr:      ctr,
		TagIDMap: tagIDMap,
	}
}

type InvertedIDXDB interface {
	SaveDocs(nodes []datastructure.Node) error
}

func NewDynamicIndex(outputDir string, maxPostingListSize int,
	server bool, spell SpellCorrectorI, indexedData IndexedData, boltDB BboltDBI) (*DynamicIndex, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return &DynamicIndex{}, err
	}
	idx := &DynamicIndex{
		TermIDMap:                 pkg.NewIDMap(),
		intermediateIndices:       []string{},
		workingDir:                pwd,
		maxDynamicPostingListSize: maxPostingListSize,
		docWordCount:              make(map[int]int),
		outputDir:                 outputDir,
		docsCount:                 0,
		spellCorrectorBuilder:     spell,
		IndexedData:               indexedData,
		documentStore:             boltDB,
	}
	if server {
		err := idx.LoadMeta()
		if err != nil {
			return nil, err
		}
	}

	return idx, nil
}

// SpimiBatchIndex a function to create multiple inverted index segments from osm objects and
// then merge all of those segments into one merged inverted index using a single-pass-in-memory indexing algorithm
func (Idx *DynamicIndex) SpimiBatchIndex(ctx context.Context, osmSpatialIdx geo.OSMSpatialIndex,
	osmRelations []geo.OsmRelation) ([]datastructure.Node, error) {
	var batchingLock sync.Mutex // buat lock block & nodeIDx.

	osmData := make([]datastructure.OSMObject, 0, len(Idx.IndexedData.Ways)+len(Idx.IndexedData.Nodes))

	fmt.Println("")
	bar := progressbar.NewOptions(4,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][2/3]Indexing osm objects..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	fmt.Println("")
	bar.Add(1)
	fmt.Println("")

	block := 0
	nodeIDX := 0

	nodeBoundingBox := make(map[string]geo.BoundingBox)

	type IndexingResults struct {
		Error error
	}

	allSearchNodes := make([]datastructure.Node, 0, len(Idx.IndexedData.Ways)+len(Idx.IndexedData.Nodes))

	processOSMWaysBatch := func(ways []geo.OSMWay, wg *sync.WaitGroup, ctx context.Context, lock *sync.Mutex, indexingRes chan<- IndexingResults) {
		defer wg.Done()
		searchNodes := []datastructure.Node{}

		for _, way := range ways {

			select {
			case <-ctx.Done():
				indexingRes <- IndexingResults{Error: fmt.Errorf("context cancelled")}
				return
			default:
			}

			lat := make([]float64, len(way.NodeIDs))
			lon := make([]float64, len(way.NodeIDs))
			for i := 0; i < len(way.NodeIDs); i++ {
				node := way.NodeIDs[i]
				nodeLat := Idx.IndexedData.Ctr.GetNode(node).Lat
				nodeLon := Idx.IndexedData.Ctr.GetNode(node).Lon
				lat[i] = nodeLat
				lon[i] = nodeLon
			}

			sort.Float64s(lat)
			sort.Float64s(lon)

			centerLat, centerLon := lat[len(lat)/2], lon[len(lon)/2]

			name, street, tipe, postalCode, houseNumber := geo.GetNameAddressTypeFromOSMWay(way.TagMap)

			if name == "" || tipe == "chalet" {
				continue
			}

			if IsWayDuplicateCheck(strings.ToLower(name), lat, lon, nodeBoundingBox, lock) {
				// cek duplikat kalo sebelumnya ada way dengan nama sama dan posisi sama dengan way ini.
				continue
			}

			address, city := Idx.GetFullAdress(street, postalCode, houseNumber, centerLat, centerLon, osmSpatialIdx, osmRelations)

			lock.Lock()
			nodeBoundingBox[strings.ToLower(name)] = geo.NewBoundingBox(lat, lon)

			searchNodes = append(searchNodes, datastructure.NewNode(nodeIDX, name, centerLat,
				centerLon, address, tipe, city))

			rtreeItem := datastructure.OSMObject{
				ID:  nodeIDX,
				Lat: centerLat,
				Lon: centerLon,
			}
			osmData = append(osmData, rtreeItem)

			nodeIDX++
			lock.Unlock()

			if len(searchNodes) == BATCH_SIZE {
				errChan := make(chan error)

				go func() {
					errChan <- Idx.SpimiInvert(searchNodes, &block, lock, "name", ctx)
				}()

				go func() {
					errChan <- Idx.SpimiInvert(searchNodes, &block, lock, "address", ctx)
				}()

				if err := <-errChan; err != nil {
					indexingRes <- IndexingResults{Error: err}
					return
				}

				if err := <-errChan; err != nil {
					indexingRes <- IndexingResults{Error: err}
					return
				}

				err := Idx.documentStore.SaveDocs(searchNodes)
				if err != nil {
					indexingRes <- IndexingResults{Error: err}
					return
				}

				lock.Lock()
				allSearchNodes = append(allSearchNodes, searchNodes...)
				lock.Unlock()

				searchNodes = []datastructure.Node{}
			}
		}

		if len(searchNodes) != 0 {
			errChan := make(chan error)
			go func() {
				errChan <- Idx.SpimiInvert(searchNodes, &block, lock, "name", ctx)
			}()

			go func() {
				errChan <- Idx.SpimiInvert(searchNodes, &block, lock, "address", ctx)
			}()

			if err := <-errChan; err != nil {
				indexingRes <- IndexingResults{Error: err}
				return
			}

			if err := <-errChan; err != nil {
				indexingRes <- IndexingResults{Error: err}
				return
			}

			err := Idx.documentStore.SaveDocs(searchNodes)
			if err != nil {
				indexingRes <- IndexingResults{Error: err}
				return
			}

			lock.Lock()
			allSearchNodes = append(allSearchNodes, searchNodes...)
			lock.Unlock()
		}

	}

	indexingRes := make(chan IndexingResults)

	batchingOSMWays := func(ways []geo.OSMWay, ctx context.Context) error {
		var wg sync.WaitGroup
		for start, end := 0, 0; start < len(ways); start = end {
			wg.Add(1)
			end = start + BATCH_SIZE
			if end > len(ways) {
				end = len(ways)
			}
			batch := ways[start:end]
			go processOSMWaysBatch(batch, &wg, ctx, &batchingLock, indexingRes)
		}

		go func() {
			// semua goroutine done jika context cancelled atau semua goroutine selesai atau ada salah satu goroutine batchProcessing yang error.
			// kalau done semua goroutine, close indexingRes channel.
			wg.Wait()
			close(indexingRes)
		}()

		for res := range indexingRes {
			// iterate indexing res channel, jika ada yang error, return error & keluar dari SpimiBatchIndexing ->
			// cancel context, semua goroutine batchProcessing akan Done & close indexingRes Channel.

			if res.Error != nil {
				return res.Error
			}
		}
		return nil
	}

	processOSMNodesBatch := func(nodes []geo.OSMNode, wg *sync.WaitGroup, ctx context.Context, lock *sync.Mutex, indexingRes chan<- IndexingResults) {
		defer wg.Done()

		searchNodes := []datastructure.Node{}

		for _, node := range nodes {
			select {
			case <-ctx.Done():
				indexingRes <- IndexingResults{Error: fmt.Errorf("context cancelled")}
				return
			default:
			}

			name, street, tipe, postalCode, houseNumber := geo.GetNameAddressTypeFromOSMWay(node.TagMap)
			if name == "" || tipe == "chalet" {
				continue
			}

			if IsNodeDuplicateCheck(strings.ToLower(name), node.Lat, node.Lon, nodeBoundingBox, lock) {
				// cek duplikat kalo sebelumnya ada way dengan nama sama dan posisi sama dengan node ini. gak usah set bounding box buat node.
				continue
			}

			address, city := Idx.GetFullAdress(street, postalCode, houseNumber, node.Lat, node.Lon, osmSpatialIdx, osmRelations)

			lock.Lock()

			searchNodes = append(searchNodes, datastructure.NewNode(nodeIDX, name, node.Lat,
				node.Lon, address, tipe, city))

			rtreeItem := datastructure.OSMObject{
				ID:  nodeIDX,
				Lat: node.Lat,
				Lon: node.Lon,
			}
			osmData = append(osmData, rtreeItem)

			nodeIDX++
			lock.Unlock()

			if len(searchNodes) == BATCH_SIZE {
				errChan := make(chan error)
				go func() {
					errChan <- Idx.SpimiInvert(searchNodes, &block, lock, "name", ctx)
				}()

				go func() {
					errChan <- Idx.SpimiInvert(searchNodes, &block, lock, "address", ctx)
				}()

				if err := <-errChan; err != nil {
					indexingRes <- IndexingResults{Error: err}
					return
				}

				if err := <-errChan; err != nil {
					indexingRes <- IndexingResults{Error: err}
					return
				}

				err := Idx.documentStore.SaveDocs(searchNodes)
				if err != nil {
					indexingRes <- IndexingResults{Error: err}
					return
				}

				lock.Lock()
				allSearchNodes = append(allSearchNodes, searchNodes...)
				lock.Unlock()

				searchNodes = []datastructure.Node{}
			}
		}

		if len(searchNodes) != 0 {
			errChan := make(chan error)
			go func() {
				errChan <- Idx.SpimiInvert(searchNodes, &block, lock, "name", ctx)
			}()

			go func() {
				errChan <- Idx.SpimiInvert(searchNodes, &block, lock, "address", ctx)
			}()

			if err := <-errChan; err != nil {
				indexingRes <- IndexingResults{Error: err}
				return
			}

			if err := <-errChan; err != nil {
				indexingRes <- IndexingResults{Error: err}
				return
			}

			err := Idx.documentStore.SaveDocs(searchNodes)
			if err != nil {
				indexingRes <- IndexingResults{Error: err}
				return
			}

			lock.Lock()
			allSearchNodes = append(allSearchNodes, searchNodes...)
			lock.Unlock()
		}
	}

	indexingNodesRes := make(chan IndexingResults)

	batchingOSMNodes := func(nodes []geo.OSMNode, ctx context.Context) error {
		var wg sync.WaitGroup
		for start, end := 0, 0; start < len(nodes); start = end {
			wg.Add(1)
			end = start + BATCH_SIZE
			if end > len(nodes) {
				end = len(nodes)
			}
			batch := nodes[start:end]
			go processOSMNodesBatch(batch, &wg, ctx, &batchingLock, indexingNodesRes)
		}

		go func() {
			// semua goroutine done jika context cancelled atau semua goroutine selesai atau ada salah satu goroutine batchProcessing yang error.
			// kalau done semua goroutine, close indexingNodesRes channel.
			wg.Wait()
			close(indexingNodesRes)
		}()

		for res := range indexingNodesRes {
			// iterate indexing res channel, jika ada yang error, return error & keluar dari SpimiBatchIndexing ->
			// cancel context, semua goroutine batchProcessing akan Done & close indexingRes Channel.
			// if tidak error -> tunggu sampai wait.Done() unblock -> close indexingNodesRes channel & exit loop ini.
			if res.Error != nil {
				return res.Error
			}
		}
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var errChan = make(chan error, 2)
	go func() {
		defer wg.Done()

		err := batchingOSMWays(Idx.IndexedData.Ways, ctx)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	go func() {
		defer wg.Done()
		err := batchingOSMNodes(Idx.IndexedData.Nodes, ctx)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		// kalau error -> return err , cancel context, wg.Wait() unblock & close errChan
		// kalau error == nil semua -> wg.Wait() unblock & close errChan. keluar dari loop ini.
		if err != nil {
			bar.Add(3)
			fmt.Println("")
			return nil, err
		}
		bar.Add(1)
		fmt.Println("")
	}

	err := datastructure.SerializeRtreeData(Idx.workingDir, Idx.outputDir, osmData)
	if err != nil {
		return nil, err
	}

	Idx.docsCount = nodeIDX

	// merge semua inverted indexes di intermediateIndices ke merged_index.

	// merged untuk field name
	mergedIndex := NewInvertedIndex("merged_name_index", Idx.outputDir, Idx.workingDir)
	indices := []*InvertedIndex{}
	for _, indexID := range Idx.intermediateIndices {
		if strings.Contains(indexID, "name") {
			index := NewInvertedIndex(indexID, Idx.outputDir, Idx.workingDir)
			err := index.OpenReader()
			if err != nil {
				return nil, err
			}
			indices = append(indices, index)
		}
	}
	mergedIndex.OpenWriter()

	err = Idx.Merge(indices, mergedIndex)
	if err != nil {
		return nil, err
	}

	lenDF := Idx.MergeFieldLengths(indices)
	mergedIndex.SetLenFieldInDoc(lenDF)

	for _, index := range indices {
		err := index.Close()
		if err != nil {
			return nil, err
		}
	}
	err = mergedIndex.Close()
	if err != nil {
		return nil, err
	}

	// merged untuk field address
	mergedIndex = NewInvertedIndex("merged_address_index", Idx.outputDir, Idx.workingDir)
	indices = []*InvertedIndex{}
	for _, indexID := range Idx.intermediateIndices {
		if strings.Contains(indexID, "address") {
			index := NewInvertedIndex(indexID, Idx.outputDir, Idx.workingDir)
			err := index.OpenReader()
			if err != nil {
				return nil, err
			}
			indices = append(indices, index)
		}
	}
	mergedIndex.OpenWriter()

	err = Idx.Merge(indices, mergedIndex)
	if err != nil {
		return nil, err
	}

	lenDF = Idx.MergeFieldLengths(indices)
	mergedIndex.SetLenFieldInDoc(lenDF)
	for _, index := range indices {
		err := index.Close()
		if err != nil {
			return nil, err
		}
	}
	err = mergedIndex.Close()
	if err != nil {
		return nil, err
	}

	bar.Add(1)
	fmt.Println("")
	return allSearchNodes, nil
}

func IsWayDuplicateCheck(name string, lats, lons []float64, nodeBoundingBox map[string]geo.BoundingBox,
	lock *sync.Mutex) bool {
	lock.Lock()
	prevBB, ok := nodeBoundingBox[name]

	if !ok {
		lock.Unlock()
		return false
	}
	contain := prevBB.PointsContains(lats, lons)

	if !contain {
		// perbesar bounding box nya karena namanya sama tapi mungkin bb sebelumnya lebih kecil & gak contain bb ini.
		nodeBoundingBox[name] = geo.NewBoundingBox(lats, lons)
	}
	lock.Unlock()

	currWayBB := geo.NewBoundingBox(lats, lons)
	inverseContain := currWayBB.PointsContains(prevBB.GetMin(), prevBB.GetMax()) // cek sebaliknya (cuur osm way Bounding Box contain previous same name bounding box)
	return contain || inverseContain
}

func IsNodeDuplicateCheck(name string, lats, lon float64, nodeBoundingBox map[string]geo.BoundingBox,
	lock *sync.Mutex) bool {
	lock.Lock()
	defer lock.Unlock()
	prevBB, ok := nodeBoundingBox[name]
	if !ok {
		return false
	}
	contain := prevBB.Contains(lats, lon)
	return contain
}

func (Idx *DynamicIndex) MergeFieldLengths(indices []*InvertedIndex) map[int]int {
	lenDF := make(map[int]int)
	for _, index := range indices {
		for docID, fieldLength := range index.lenFieldInDoc {
			if _, ok := lenDF[docID]; ok {
				lenDF[docID] += fieldLength
			} else {
				lenDF[docID] = fieldLength
			}
		}
	}
	return lenDF
}

// Merge. merge k inverted indexes into 1 merged index.
func (Idx *DynamicIndex) Merge(indices []*InvertedIndex, mergedIndex *InvertedIndex) error {
	lastTerm, lastPosting := -1, []int{}
	mergeKArrayIterator := NewMergeKArrayIterator(indices)
	for output, err := range mergeKArrayIterator.mergeKSortedArray() {
		if err != nil {
			return fmt.Errorf("error when merge posting lists: %w", err)
		}

		currTerm, currPostings := output.TermID, output.Postings

		if currTerm != lastTerm {

			if lastTerm != -1 {
				sort.Ints(lastPosting)
				err := mergedIndex.AppendPostingList(lastTerm, lastPosting)
				if err != nil {
					return fmt.Errorf("error when merge posting lists: %w", err)
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

// SpimiInvert is a function to invert a batch of nodes into a posting list & write it to inverted index file.
// https://nlp.stanford.edu/IR-book/pdf/04const.pdf (Figure 4.4 Spimi-invert)
func (Idx *DynamicIndex) SpimiInvert(nodes []datastructure.Node, block *int, lock *sync.Mutex, field string,
	ctx context.Context) error {
	postingSize := 0

	termToPostingMap := make(map[int][]int)
	lenDF := make(map[int]int)
	tokenStreams := Idx.SpimiParseOSMNodes(nodes, lock, field, lenDF, ctx) // [pair of termID and nodeID]

	var postingList []int
	for _, termDocPair := range tokenStreams {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		default:
		}

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

		if postingSize >= Idx.maxDynamicPostingListSize {
			postingSize = 0
			terms := []int{}
			for termID, _ := range termToPostingMap {
				terms = append(terms, termID)
			}
			sort.Ints(terms)

			lock.Lock()
			indexID := "index_" + field + "_" + strconv.Itoa(*block)
			*block += 1
			lock.Unlock()

			index := NewInvertedIndex(indexID, Idx.outputDir, Idx.workingDir)
			index.SetLenFieldInDoc(lenDF)
			err := index.OpenWriter()
			if err != nil {
				return err
			}

			lock.Lock()
			Idx.intermediateIndices = append(Idx.intermediateIndices, indexID)
			lock.Unlock()
			for term := range terms {

				sort.Ints(termToPostingMap[term])
				index.AppendPostingList(term, termToPostingMap[term])
			}

			termToPostingMap = make(map[int][]int)
			index.Close()
		}
	}

	terms := []int{}
	for termID, _ := range termToPostingMap {
		terms = append(terms, termID)
	}
	sort.Ints(terms)

	lock.Lock()
	indexID := "index_" + field + "_" + strconv.Itoa(*block)
	*block += 1
	lock.Unlock()

	index := NewInvertedIndex(indexID, Idx.outputDir, Idx.workingDir)
	index.SetLenFieldInDoc(lenDF)
	err := index.OpenWriter()
	if err != nil {
		return err
	}

	lock.Lock()
	Idx.intermediateIndices = append(Idx.intermediateIndices, indexID)
	lock.Unlock()
	for _, term := range terms {
		sort.Ints(termToPostingMap[term])
		index.AppendPostingList(term, termToPostingMap[term])
	}

	err = index.Close()
	if err != nil {
		return err
	}
	return nil
}

// SpimiParseOSMNode is a function to parse an OSM node into a token stream (termID-docID pairs).
func (Idx *DynamicIndex) SpimiParseOSMNode(node datastructure.Node, lenDF map[int]int,
	lock *sync.Mutex, field string) [][]int {
	termDocPairs := [][]int{}

	soup := ""
	switch field {
	case "name":
		soup = node.Name
	case "address":
		soup = node.Address
	}

	if soup == "" {
		return termDocPairs
	}

	words := sastrawi.Tokenize(soup)
	lock.Lock()
	Idx.docWordCount[node.ID] += len(words)
	lock.Unlock()

	lenDF[node.ID] = len(words)

	for _, word := range words {
		tokenizedWord := pkg.Stemmer.Stem(word)
		lock.Lock()
		termID := Idx.TermIDMap.GetID(tokenizedWord)
		lock.Unlock()
		pair := []int{termID, node.ID}
		termDocPairs = append(termDocPairs, pair)
	}
	return termDocPairs
}

// SpimiParseOSMNodes is a function to parse a batch of OSM nodes into a token stream (termID-docID pairs).
func (Idx *DynamicIndex) SpimiParseOSMNodes(nodes []datastructure.Node, lock *sync.Mutex, field string,
	lenDF map[int]int, ctx context.Context) [][]int {
	termDocPairs := [][]int{}
	for _, node := range nodes {
		select {
		case <-ctx.Done():
			return termDocPairs
		default:

		}
		termDocPairs = append(termDocPairs, Idx.SpimiParseOSMNode(node, lenDF, lock, field)...)
	}
	return termDocPairs
}

func (Idx *DynamicIndex) BuildSpellCorrectorAndNgram(ctx context.Context, allSearchNodes []datastructure.Node, osmSpatialIdx geo.OSMSpatialIndex,
	osmRelations []geo.OsmRelation) error {
	fmt.Println("")
	bar := progressbar.NewOptions(5,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][3/3]Building Ngram..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	fmt.Println("")
	bar.Add(1)

	Idx.docsCount = len(allSearchNodes)

	tokenizedDocs := [][]string{}
	for _, node := range allSearchNodes {

		soup := node.Name + " " + node.Address

		tokenized := sastrawi.Tokenize(soup)
		stemmedTokens := []string{}
		for _, token := range tokenized {
			stemmedToken := pkg.Stemmer.Stem(token)
			stemmedTokens = append(stemmedTokens, stemmedToken)
		}
		tokenizedDocs = append(tokenizedDocs, stemmedTokens)
	}

	bar.Add(1)
	Idx.spellCorrectorBuilder.Preprocessdata(tokenizedDocs)
	bar.Add(1)
	fmt.Println("")
	return nil
}

type SpimiIndexMetadata struct {
	TermIDMap    *pkg.IDMap
	DocWordCount map[int]int
	DocsCount    int
}

func NewSpimiIndexMetadata(termIDMap *pkg.IDMap, docWordCount map[int]int, docsCount int) SpimiIndexMetadata {
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

// SaveMeta is a function to save the metadata of the main inverted index to disk.
func (Idx *DynamicIndex) SaveMeta() error {
	// save to disk
	SpimiMeta := NewSpimiIndexMetadata(Idx.TermIDMap, Idx.docWordCount, Idx.docsCount)

	buf, err := msgpack.Marshal(&SpimiMeta)
	if err != nil {
		return fmt.Errorf("error when marshalling metadata: %w", err)
	}

	var metadataFile *os.File
	if Idx.workingDir != "/" {
		metadataFile, err = os.OpenFile(Idx.workingDir+"/"+Idx.outputDir+"/"+"meta.metadata", os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
	} else {
		metadataFile, err = os.OpenFile(Idx.outputDir+"/"+"meta.metadata", os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
	}

	defer metadataFile.Close()
	err = metadataFile.Truncate(0)
	if err != nil {
		return err
	}

	_, err = metadataFile.Write(buf)

	return err
}

// LoadMeta is a function to load the metadata of the main inverted index from disk.
func (Idx *DynamicIndex) LoadMeta() error {
	var metadataFile *os.File
	var err error
	if Idx.workingDir != "/" {
		metadataFile, err = os.OpenFile(Idx.workingDir+"/"+Idx.outputDir+"/"+"meta.metadata", os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
	} else {
		metadataFile, err = os.OpenFile(Idx.outputDir+"/"+"meta.metadata", os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
	}

	defer metadataFile.Close()

	stat, err := os.Stat(metadataFile.Name())
	if err != nil {
		return fmt.Errorf("error when getting metadata file stat: %w", err)
	}

	buf := make([]byte, stat.Size()*2)
	_, err = metadataFile.Read(buf)
	if err != nil {
		return fmt.Errorf("error when reading metadata file: %w", err)
	}

	save := SpimiIndexMetadata{}

	err = msgpack.Unmarshal(buf, &save)
	if err != nil {
		return fmt.Errorf("error when unmarshalling metadata merged_index: %w", err)
	}

	Idx.TermIDMap = save.TermIDMap
	Idx.docWordCount = save.DocWordCount
	Idx.docsCount = save.DocsCount

	for i := 0; i < Idx.docsCount; i++ {
		Idx.averageDocLength += float64(Idx.docWordCount[i])
	}
	Idx.averageDocLength /= float64(Idx.docsCount)
	return nil
}

func (Idx *DynamicIndex) GetAverageDocLength() float64 {
	return Idx.averageDocLength
}

func (Idx *DynamicIndex) GetOutputDir() string {
	return Idx.outputDir
}

func (Idx *DynamicIndex) GetWorkingDir() string {
	return Idx.workingDir
}

func (Idx *DynamicIndex) GetDocWordCount() map[int]int {
	return Idx.docWordCount
}

func (Idx *DynamicIndex) GetDocsCount() int {
	return Idx.docsCount
}

func (Idx *DynamicIndex) GetTermIDMap() *pkg.IDMap {
	return Idx.TermIDMap
}

func (Idx *DynamicIndex) BuildVocabulary() {
	Idx.TermIDMap.BuildVocabulary()
}

func (Idx *DynamicIndex) GetFullAdress(street, postalCode, houseNumber string, centerLat, centerLon float64,
	osmSpatialIdx geo.OSMSpatialIndex, osmRelations []geo.OsmRelation) (string, string) {
	centerItem := datastructure.Point{
		Lat: centerLat,
		Lon: centerLon,
	}

	boundingBox := datastructure.NewRtreeBoundingBox(2, []float64{centerLat - 0.0001, centerLon - 0.0001},
		[]float64{centerLat + 0.0001, centerLon + 0.0001})

	// membuat address
	address := ""

	if street != "" {
		address += street
	} else if osmSpatialIdx.StreetRtree.Size > 0 {
		// pick nearest street
		street := osmSpatialIdx.StreetRtree.ImprovedNearestNeighbor(centerItem)

		streetName, _, _, _, _ := geo.GetNameAddressTypeFromOSMWay(Idx.IndexedData.Ways[street.Leaf.ID].TagMap)
		address += streetName
	}

	if houseNumber != "" {
		address += ", " + houseNumber
	}

	// kelurahan
	kelurahans := []datastructure.RtreeNode{}
	if osmSpatialIdx.KelurahanRtree.Size > 0 {
		kelurahans = osmSpatialIdx.KelurahanRtree.Search(boundingBox)
	}
	kelurahan := ""

	for _, currKelurahan := range kelurahans {
		kelurahanRel := osmRelations[currKelurahan.Leaf.ID]
		if postalCode == "" {
			postalCode = kelurahanRel.PostalCode
		}
		if len(kelurahanRel.BoundaryLat) == 0 {
			continue
		}

		inside := geo.IsPointInPolygon(centerLat, centerLon, kelurahanRel.BoundaryLat, kelurahanRel.BoundaryLon)
		if inside {
			kelurahan = kelurahanRel.Name
			break
		}
	}

	if kelurahan != "" {
		address += ", " + kelurahan
	}

	// // kecamatan
	kecamatans := []datastructure.RtreeNode{}
	if osmSpatialIdx.KecamatanRtree.Size > 0 {
		kecamatans = osmSpatialIdx.KecamatanRtree.Search(boundingBox)
	}
	kecamatan := ""

	for _, currKecamatan := range kecamatans {
		kecamatanRel := osmRelations[currKecamatan.Leaf.ID]
		if len(kecamatanRel.BoundaryLat) == 0 {
			continue
		}

		inside := geo.IsPointInPolygon(centerLat, centerLon, kecamatanRel.BoundaryLat, kecamatanRel.BoundaryLon)
		if inside {
			kecamatan = kecamatanRel.Name
			break
		}

	}

	if kecamatan != "" {
		address += ", " + kecamatan
	}

	// // city
	cities := []datastructure.RtreeNode{}
	if osmSpatialIdx.KotaKabupatenRtree.Size > 0 {
		cities = osmSpatialIdx.KotaKabupatenRtree.Search(boundingBox)
	}

	city := ""

	for _, currCity := range cities {
		cityRel := osmRelations[currCity.Leaf.ID]
		if len(cityRel.BoundaryLat) == 0 {
			continue
		}
		inside := geo.IsPointInPolygon(centerLat, centerLon, cityRel.BoundaryLat, cityRel.BoundaryLon)
		if inside {
			city = cityRel.Name
			break
		}
	}

	if city != "" {
		address += ", " + city
	}

	// // provinsi
	provinsis := []datastructure.RtreeNode{}
	if osmSpatialIdx.ProvinsiRtree.Size > 0 {
		provinsis = osmSpatialIdx.ProvinsiRtree.Search(boundingBox)

	}
	provinsi := ""

	for i := 0; i < len(provinsis); i++ {
		currProvinsi := provinsis[i]
		provinsiRel := osmRelations[currProvinsi.Leaf.ID]
		if len(provinsiRel.BoundaryLat) == 0 {
			continue
		}

		inside := geo.IsPointInPolygon(centerLat, centerLon, provinsiRel.BoundaryLat, provinsiRel.BoundaryLon)
		if inside {
			provinsi = provinsiRel.Name
			// yang inii harus tanpa break
		}
	}

	if provinsi != "" {
		address += ", " + provinsi
	}

	// // postalCode
	if postalCode != "" {
		address += ", " + postalCode
	} else if osmSpatialIdx.KelurahanRtree.Size > 0 {
		nearestWayKelurahan := osmSpatialIdx.KelurahanRtree.Search(boundingBox)
		for _, currKelurahan := range nearestWayKelurahan {
			kelurahanRel := osmRelations[currKelurahan.Leaf.ID]
			inside := geo.IsPointInPolygon(centerLat, centerLon, kelurahanRel.BoundaryLat, kelurahanRel.BoundaryLon)
			if inside {
				address += ", " + kelurahanRel.PostalCode
				break
			}
		}

	}
	// country
	if osmSpatialIdx.CountryRtree.Size != 0 {
		country := osmSpatialIdx.CountryRtree.Search(boundingBox)

		countryRel := osmRelations[country[0].Leaf.ID]
		address += ", " + countryRel.Name
	}

	return address, city
}
