package index

import (
	"sort"
	"sync"
)

// MemoryIndex is a thread-safe in-memory inverted index write buffer.
type MemoryIndex struct {
	mu          sync.RWMutex
	docs        map[uint32]Document
	postings    map[string][]Posting
	docFreq     map[string]uint32
	totalLength uint64
	nextDocID   uint32
}

// NewMemoryIndex initializes an empty in-memory index.
func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{
		docs:      make(map[uint32]Document),
		postings:  make(map[string][]Posting),
		docFreq:   make(map[string]uint32),
		nextDocID: 1,
	}
}

// AddDocument indexes a new document with token positions and field lengths.
// Postings are naturally appended in increasing DocID order.
func (idx *MemoryIndex) AddDocument(doc Document, termPositions map[string][]uint32) uint32 {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	docID := idx.nextDocID
	idx.nextDocID++

	doc.DocID = docID
	idx.docs[docID] = doc
	idx.totalLength += uint64(doc.TotalLength)

	for term, positions := range termPositions {
		if len(positions) == 0 {
			continue
		}
		// Copy positions to avoid external mutations
		posCopy := make([]uint32, len(positions))
		copy(posCopy, positions)

		idx.postings[term] = append(idx.postings[term], Posting{
			DocID:     docID,
			Positions: posCopy,
		})
		idx.docFreq[term]++
	}

	return docID
}

// TotalDocs returns the total number of indexed documents (N).
func (idx *MemoryIndex) TotalDocs() uint32 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return uint32(len(idx.docs))
}

// AvgDocLength returns the average document length across all documents in the index.
func (idx *MemoryIndex) AvgDocLength() float64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if len(idx.docs) == 0 {
		return 0
	}
	return float64(idx.totalLength) / float64(len(idx.docs))
}

// GetDocument returns the Document by its internal DocID.
func (idx *MemoryIndex) GetDocument(docID uint32) (Document, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	doc, ok := idx.docs[docID]
	return doc, ok
}

// GetPostings returns a copy of the posting list for a given term.
func (idx *MemoryIndex) GetPostings(term string) []Posting {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	list, ok := idx.postings[term]
	if !ok {
		return nil
	}
	res := make([]Posting, len(list))
	copy(res, list)
	return res
}

// DocFrequency returns how many documents contain the given term (n_t).
func (idx *MemoryIndex) DocFrequency(term string) uint32 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docFreq[term]
}

// IntersectAND intersects posting lists for multiple terms, shortest first (Section 5.4).
func (idx *MemoryIndex) IntersectAND(terms []string) []uint32 {
	if len(terms) == 0 {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	type termList struct {
		term     string
		postings []Posting
	}
	lists := make([]termList, 0, len(terms))
	for _, term := range terms {
		p, ok := idx.postings[term]
		if !ok || len(p) == 0 {
			// One term has 0 postings, intersection is empty
			return nil
		}
		lists = append(lists, termList{term: term, postings: p})
	}

	// Sort lists by length ascending (shortest list first)
	sort.Slice(lists, func(i, j int) bool {
		return len(lists[i].postings) < len(lists[j].postings)
	})

	// Start with shortest list's docIDs
	candidates := make([]uint32, len(lists[0].postings))
	for i, p := range lists[0].postings {
		candidates[i] = p.DocID
	}

	// Intersect with remaining lists
	for i := 1; i < len(lists); i++ {
		other := lists[i].postings
		matched := make([]uint32, 0, len(candidates))
		p1, p2 := 0, 0
		for p1 < len(candidates) && p2 < len(other) {
			d1 := candidates[p1]
			d2 := other[p2].DocID
			if d1 == d2 {
				matched = append(matched, d1)
				p1++
				p2++
			} else if d1 < d2 {
				p1++
			} else {
				p2++
			}
		}
		candidates = matched
		if len(candidates) == 0 {
			break
		}
	}

	return candidates
}

// MatchPhrase finds documents where the phrase appears in exact sequential order:
// pos(t_{i+1}) == pos(t_i) + 1 (Section 5.4).
func (idx *MemoryIndex) MatchPhrase(terms []string) []uint32 {
	if len(terms) == 0 {
		return nil
	}
	if len(terms) == 1 {
		postings := idx.GetPostings(terms[0])
		res := make([]uint32, len(postings))
		for i, p := range postings {
			res[i] = p.DocID
		}
		return res
	}

	commonDocIDs := idx.IntersectAND(terms)
	if len(commonDocIDs) == 0 {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var matchingDocs []uint32

	// For each common document, check if positions align sequentially
	for _, docID := range commonDocIDs {
		// Collect position slices for each term for this doc
		termPositions := make([][]uint32, len(terms))
		for i, term := range terms {
			postings := idx.postings[term]
			// Binary search docID in postings list
			idxPos := sort.Search(len(postings), func(k int) bool {
				return postings[k].DocID >= docID
			})
			if idxPos < len(postings) && postings[idxPos].DocID == docID {
				termPositions[i] = postings[idxPos].Positions
			}
		}

		if phraseInDoc(termPositions) {
			matchingDocs = append(matchingDocs, docID)
		}
	}

	return matchingDocs
}

// phraseInDoc checks if any sequence of positions satisfies pos[i+1] == pos[i] + 1.
func phraseInDoc(positions [][]uint32) bool {
	if len(positions) == 0 {
		return false
	}
	firstPositions := positions[0]

	for _, startPos := range firstPositions {
		match := true
		expectedPos := startPos + 1
		for step := 1; step < len(positions); step++ {
			stepPositions := positions[step]
			// Check if expectedPos is present in stepPositions
			idx := sort.Search(len(stepPositions), func(k int) bool {
				return stepPositions[k] >= expectedPos
			})
			if idx < len(stepPositions) && stepPositions[idx] == expectedPos {
				expectedPos++
			} else {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
