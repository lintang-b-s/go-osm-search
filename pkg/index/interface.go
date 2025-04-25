package index

import "github.com/lintang-b-s/osm-search/pkg/datastructure"

type SpellCorrectorI interface {
	Preprocessdata(tokenizedDocs [][]string)
	GetWordCandidates(mispelledWord string, editDistance int) ([]int, []string, error)
	GetCorrectQueryCandidates(allPossibleQueryTerms [][]datastructure.WordCandidate) [][]datastructure.WordCandidate
	GetCorrectSpellingSuggestion(allCorrectQueryCandidates [][]datastructure.WordCandidate) ([]int, error)
	GetMatchedWordBasedOnPrefix(prefixWord string) ([]int, error)
	GetMatchedWordsAutocomplete(allQueryCandidates [][]datastructure.WordCandidate, originalQueryTerms []int) ([][]int, error)
}

type DocumentStoreI interface {
	WriteDocs(docs []datastructure.Node)
}

type BboltDBI interface {
	SaveDocs(nodes []datastructure.Node) error
}
