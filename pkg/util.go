package pkg

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/RadhiFadlillah/go-sastrawi"
)

var dictionary = sastrawi.DefaultDictionary()

var Stemmer = sastrawi.NewStemmer(dictionary)

type IDMap struct {
	StrToID    map[string]int
	IDToStr    map[int]string
	Vocabulary map[string]bool
	sync.Mutex
}

func NewIDMap() *IDMap {
	return &IDMap{
		StrToID: make(map[string]int),
		IDToStr: make(map[int]string),
	}
}

func (idMap *IDMap) GetID(str string) int {
	idMap.Lock()
	defer idMap.Unlock()
	if id, ok := idMap.StrToID[str]; ok {
		return id
	}

	id := len(idMap.StrToID)
	idMap.StrToID[str] = id
	idMap.IDToStr[id] = str

	return id
}

func (idMap *IDMap) GetStr(id int) string {
	if str, ok := idMap.IDToStr[id]; ok {
		return str
	}
	return ""
}

func (idMap *IDMap) GetSortedTerms() []string {
	sortedTerms := make([]string, len(idMap.StrToID))
	for term, id := range idMap.StrToID {
		sortedTerms[id] = term
	}
	sort.Strings(sortedTerms)
	return sortedTerms
}

func (idMap *IDMap) BuildVocabulary() {
	idMap.Vocabulary = make(map[string]bool)
	for id := range idMap.StrToID {
		idMap.Vocabulary[id] = true
	}
}

func (idMap *IDMap) GetVocabulary() map[string]bool {
	return idMap.Vocabulary
}

func (idMap *IDMap) IsInVocabulary(term string) bool {
	_, ok := idMap.Vocabulary[term]
	return ok
}



// error

type Error struct {
	orig error
	msg  string
	code error
}

func (e *Error) Error() string {
	if e.orig != nil {
		return fmt.Sprintf("%s", e.msg)
	}

	return e.msg
}

func (e *Error) Unwrap() error {
	return e.orig
}

func WrapErrorf(orig error, code error, format string, a ...interface{}) error {
	return &Error{
		code: code,
		orig: orig,
		msg:  fmt.Sprintf(format, a...),
	}
}

func (e *Error) Code() error {
	return e.code
}

var (
	ErrInternalServerError = errors.New("internal Server Error")
	ErrNotFound            = errors.New("your requested Item is not found")
	ErrConflict            = errors.New("your Item already exist")
	ErrBadParamInput       = errors.New("given Param is not valid")
)

var MessageInternalServerError string = "internal server error"
