package searcher

import (
	"math"
	"sort"
)

// https://trec.nist.gov/pubs/trec13/papers/microsoft-cambridge.web.hard.pdf
func (se *Searcher) scoreBM25Field(allPostingsNameField map[int][]int,
	allPostingsAddressField map[int][]int, allQueryTermIDs []int) []int {

	documentScore := make(map[int]float64)

	docCount := float64(se.Idx.GetDocsCount())

	nameLenDF := se.MainIndexNameField.GetLenFieldInDoc()
	addressLenDF := se.MainIndexAddressField.GetLenFieldInDoc()
	averageNameLenDF := se.MainIndexNameField.GetAverageFieldLength()
	averageAddressLenDF := se.MainIndexAddressField.GetAverageFieldLength()

	for _, qTermID := range allQueryTermIDs {

		namePostingsList, ok := allPostingsNameField[qTermID]
		addressPostingsList, ok := allPostingsAddressField[qTermID]

		uniqueDocContainingTerm := make(map[int]struct{}, len(namePostingsList)+len(addressPostingsList))

		// name field
		tfTermDocNameField := make(map[int]float64, len(namePostingsList))

		if ok {
			for _, docID := range namePostingsList {
				tfTermDocNameField[docID]++ // conunt(t,d)
				uniqueDocContainingTerm[docID] = struct{}{}
			}
		}

		// address field

		tfTermDocAddressField := make(map[int]float64, len(addressPostingsList))

		if ok {
			for _, docID := range addressPostingsList {
				tfTermDocAddressField[docID]++ // conunt(t,d)
				uniqueDocContainingTerm[docID] = struct{}{}
			}
		}

		// score untuk doc yang include term di name field

		idf := math.Log10(docCount-float64(len(uniqueDocContainingTerm))+0.5) - math.Log10(float64(len(uniqueDocContainingTerm))+0.5) // log(N-df_t+0.5/df_t+0.5)

		for docID, tftd := range tfTermDocNameField {
			weightTD := NAME_WEIGHT * (tftd / (1 + NAME_B*((float64(nameLenDF[docID])/averageNameLenDF)-1)))
			documentScore[docID] += (weightTD / (K1_BM25F + weightTD)) * idf
		}

		for docID, tftd := range tfTermDocAddressField {
			weightTD := ADDRESS_WEIGHT * (tftd / (1 + NAME_B*((float64(addressLenDF[docID])/averageAddressLenDF)-1)))
			documentScore[docID] += (weightTD / (K1_BM25F + weightTD)) * idf
		}

	}

	documentIDs := make([]int, 0, len(documentScore))
	for k := range documentScore {
		if se.Idx.IsWikiData(k) {
			documentScore[k] += osmObjContainWikiDataWeight
		}
		documentIDs = append(documentIDs, k)
	}

	sort.SliceStable(documentIDs, func(i, j int) bool {
		return documentScore[documentIDs[i]] > documentScore[documentIDs[j]]
	})

	return documentIDs
}

func (se *Searcher) scoreBM25FieldWithScores(allPostingsNameField map[int][]int,
	allPostingsAddressField map[int][]int, allQueryTermIDs []int) []docWithScore {

	documentScore := make(map[int]float64)

	docCount := float64(se.Idx.GetDocsCount())

	nameLenDF := se.MainIndexNameField.GetLenFieldInDoc()
	addressLenDF := se.MainIndexAddressField.GetLenFieldInDoc()
	averageNameLenDF := se.MainIndexNameField.GetAverageFieldLength()
	averageAddressLenDF := se.MainIndexAddressField.GetAverageFieldLength()

	for _, qTermID := range allQueryTermIDs {

		namePostingsList, ok := allPostingsNameField[qTermID]
		addressPostingsList, ok := allPostingsAddressField[qTermID]

		uniqueDocContainingTerm := make(map[int]struct{}, len(namePostingsList)+len(addressPostingsList))

		// name field
		tfTermDocNameField := make(map[int]float64, len(namePostingsList))

		if ok {
			for _, docID := range namePostingsList {
				tfTermDocNameField[docID]++ // conunt(t,d)
				uniqueDocContainingTerm[docID] = struct{}{}
			}
		}

		// address field

		tfTermDocAddressField := make(map[int]float64, len(addressPostingsList))

		if ok {
			for _, docID := range addressPostingsList {
				tfTermDocAddressField[docID]++ // conunt(t,d)
				uniqueDocContainingTerm[docID] = struct{}{}
			}
		}

		// score untuk doc yang include term di name field

		idf := math.Log10(docCount-float64(len(uniqueDocContainingTerm))+0.5) - math.Log10(float64(len(uniqueDocContainingTerm))+0.5) // log(N-df_t+0.5/df_t+0.5)

		for docID, tftd := range tfTermDocNameField {
			weightTD := NAME_WEIGHT * (tftd / (1 + NAME_B*((float64(nameLenDF[docID])/averageNameLenDF)-1)))
			documentScore[docID] += (weightTD / (K1_BM25F + weightTD)) * idf
		}

		for docID, tftd := range tfTermDocAddressField {
			weightTD := ADDRESS_WEIGHT * (tftd / (1 + NAME_B*((float64(addressLenDF[docID])/averageAddressLenDF)-1)))
			documentScore[docID] += (weightTD / (K1_BM25F + weightTD)) * idf
		}

	}

	documentIDs := make([]docWithScore, 0, len(documentScore))
	for k := range documentScore {
		if se.Idx.IsWikiData(k) {
			documentScore[k] += osmObjContainWikiDataWeight
		}
		documentIDs = append(documentIDs, newDocWithScore(k, documentScore[k]))
	}

	return documentIDs
}

func (se *Searcher) scoreBM25Plus(allPostingsField map[int][]int) []int {
	// param bm25+

	documentScore := make(map[int]float64)

	docsCount := float64(se.Idx.GetDocsCount())
	docWordCount := se.Idx.GetDocWordCount()

	avgDocLength := se.Idx.GetAverageDocLength()

	for _, postings := range allPostingsField {

		tfTermDoc := make(map[int]float64)
		for _, docID := range postings {
			tfTermDoc[docID]++ // conunt(t,d)
		}

		idf := math.Log10(docsCount+1) - math.Log10(float64(len(tfTermDoc))) // log(N/df_t)

		for docID, tftd := range tfTermDoc {
			// https://www.cs.otago.ac.nz/homepages/andrew/papers/2014-2.pdf

			documentScore[docID] += idf * (DELTA +
				((K1+1)+tftd)/(K1*(1-B+B*float64(docWordCount[docID])/avgDocLength)+tftd))
		}
	}

	documentIDs := make([]int, 0, len(documentScore))
	for k := range documentScore {
		if se.Idx.IsWikiData(k) {
			documentScore[k] += osmObjContainWikiDataWeight
		}
		documentIDs = append(documentIDs, k)
	}

	sort.SliceStable(documentIDs, func(i, j int) bool {
		return documentScore[documentIDs[i]] > documentScore[documentIDs[j]]
	})

	return documentIDs
}

func (se *Searcher) scoreTFIDFCosine(allPostings map[int][]int,
	queryWordCount map[int]int) []int {
	documentScore := make(map[int]float64) // menyimpan skor cosine tf-idf docs \dot tf-idf query

	docsCount := float64(se.Idx.GetDocsCount())
	docNorm := make(map[int]float64)
	queryNorm := 0.0
	for qTermID, postings := range allPostings {
		// iterate semua term di query, hitung tf-idf query dan tf-idf document, accumulate skor cosine di docScore

		termCountInDoc := make(map[int]int)
		for _, docID := range postings {
			termCountInDoc[docID]++ // conunt(t,d)
		}

		tfTermQuery := 1 + math.Log10(float64(queryWordCount[qTermID]))                  //  1 + log(count(t,q))
		idfTermQuery := math.Log10(docsCount) - math.Log10(float64(len(termCountInDoc))) // log(N/df_t)
		tfIDFTermQuery := tfTermQuery * idfTermQuery

		for docID, termCount := range termCountInDoc {
			tf := 1 + math.Log10(float64(termCount)) //  //  1 + log(count(t,d))

			tfIDFTermDoc := tf * idfTermQuery //tfidf docID

			documentScore[docID] += tfIDFTermDoc * tfIDFTermQuery // summation tfidfDoc*tfIDfquery over query terms

			docNorm[docID] += tfIDFTermDoc * tfIDFTermDoc // document Norm
		}

		queryNorm += tfIDFTermQuery * tfIDFTermQuery
	}

	queryNorm = math.Sqrt(queryNorm)

	documentIDs := make([]int, 0, len(documentScore))
	for k := range documentScore {
		if se.Idx.IsWikiData(k) {
			documentScore[k] += osmObjContainWikiDataWeight
		}
		documentIDs = append(documentIDs, k)
	}

	sort.SliceStable(documentIDs, func(i, j int) bool {
		return documentScore[documentIDs[i]] > documentScore[documentIDs[j]]
	})

	return documentIDs
}
