package searcher

const (
	COUNT_THRESOLD_NGRAM = 2
	EDIT_DISTANCE        = 2
	START_CHAR           = '-'
	ALPHA_NO_EDIT_PROB   = 0.9
	ALPHA_EDIT_PROB      = 0.1
)

type EditConst int

const (
	Insertion EditConst = iota
	Deletion
	Substitution
	Transposition
)

type SimiliarityScoring int

const (
	TF_IDF_COSINE SimiliarityScoring = iota
	BM25_PLUS
	BM25_FIELD
)

// BM25+ parameter
const (
	DELTA = 1.0
	K1    = 1.2
	B     = 0.98
	// param BM25F
	K1_BM25F       = 10
	NAME_WEIGHT    = 20
	ADDRESS_WEIGHT = 1
	NAME_B         = 0.95
	ADDRESS_B      = 0.3
)

const (
	osmObjContainWikiDataWeight = 10
)
