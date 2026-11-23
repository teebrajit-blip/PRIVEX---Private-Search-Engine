package ranking

import (
	"container/heap"
	"math"
	"sort"

	"privex/internal/index"
)

// Params defines parameters for BM25 ranking.
type Params struct {
	K1 float64 // Term saturation parameter (typically 1.2 to 2.0)
	B  float64 // Document length normalization parameter (typically 0.75)
}

// DefaultParams provides production baseline BM25 constants.
func DefaultParams() Params {
	return Params{
		K1: 1.2,
		B:  0.75,
	}
}

// ScoredDoc represents a document with its computed relevance score.
type ScoredDoc struct {
	DocID      uint32
	ExternalID int64
	Score      float64
	Document   index.Document
}

// Min-heap implementation for efficient Top-K retrieval
type docMinHeap []ScoredDoc

func (h docMinHeap) Len() int           { return len(h) }
func (h docMinHeap) Less(i, j int) bool { return h[i].Score < h[j].Score } // min on top
func (h docMinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *docMinHeap) Push(x any) {
	*h = append(*h, x.(ScoredDoc))
}

func (h *docMinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// ComputeIDF calculates the Robertson-Spärck Jones BM25 IDF:
// idf(t) = ln(1 + (N - n_t + 0.5) / (n_t + 0.5))
// This formula is strictly positive for any n_t <= N.
func ComputeIDF(N, docFreq uint32) float64 {
	if N == 0 {
		return 0
	}
	n := float64(docFreq)
	num := float64(N) - n + 0.5
	den := n + 0.5
	return math.Log(1.0 + (num / den))
}

// ScoreTerm calculates BM25 contribution for a single term in a document.
func ScoreTerm(tf, docLen, avgDocLen, idf, k1, b float64) float64 {
	if tf <= 0 || avgDocLen <= 0 {
		return 0
	}
	lenNorm := 1.0 - b + (b * (docLen / avgDocLen))
	tfSaturation := (tf * (k1 + 1.0)) / (tf + (k1 * lenNorm))
	return idf * tfSaturation
}

// Ranker executes BM25 scoring over an index.
type Ranker struct {
	params Params
}

// NewRanker creates a Ranker with specified parameters.
func NewRanker(params Params) *Ranker {
	return &Ranker{params: params}
}

// ScoreCandidates calculates BM25 scores for a list of candidate docIDs across query terms
// and returns the top-K highest scoring documents using a min-heap.
func (r *Ranker) ScoreCandidates(idx *index.MemoryIndex, candidateDocIDs []uint32, queryTerms []string, topK int) []ScoredDoc {
	if len(candidateDocIDs) == 0 || len(queryTerms) == 0 || topK <= 0 {
		return nil
	}

	totalDocs := idx.TotalDocs()
	avgdl := idx.AvgDocLength()
	if totalDocs == 0 || avgdl == 0 {
		return nil
	}

	// Pre-calculate IDF and retrieve postings for each query term
	idfs := make(map[string]float64, len(queryTerms))
	termPostings := make(map[string][]index.Posting, len(queryTerms))
	for _, term := range queryTerms {
		df := idx.DocFrequency(term)
		idfs[term] = ComputeIDF(totalDocs, df)
		termPostings[term] = idx.GetPostings(term)
	}

	// Build DocID -> Term -> TF map for fast lookup
	docTermTF := make(map[uint32]map[string]float64, len(candidateDocIDs))
	for _, term := range queryTerms {
		for _, p := range termPostings[term] {
			if docTermTF[p.DocID] == nil {
				docTermTF[p.DocID] = make(map[string]float64)
			}
			docTermTF[p.DocID][term] = float64(len(p.Positions))
		}
	}

	h := &docMinHeap{}
	heap.Init(h)

	for _, docID := range candidateDocIDs {
		doc, ok := idx.GetDocument(docID)
		if !ok {
			continue
		}

		totalScore := 0.0
		for _, term := range queryTerms {
			tf := docTermTF[docID][term]
			if tf > 0 {
				termScore := ScoreTerm(tf, float64(doc.TotalLength), avgdl, idfs[term], r.params.K1, r.params.B)
				totalScore += termScore
			}
		}

		if totalScore > 0 {
			scored := ScoredDoc{
				DocID:      docID,
				ExternalID: doc.ExternalID,
				Score:      totalScore,
				Document:   doc,
			}
			if h.Len() < topK {
				heap.Push(h, scored)
			} else if totalScore > (*h)[0].Score {
				heap.Pop(h)
				heap.Push(h, scored)
			}
		}
	}

	// Extract from heap in descending score order
	results := make([]ScoredDoc, h.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(h).(ScoredDoc)
	}

	return results
}

// Search coordinates candidate retrieval and ranking.
// Default semantics:
// 1. Try all terms (AND).
// 2. If fewer results than minResults, relax to OR with coordination boost (Section 5.4).
func (r *Ranker) Search(idx *index.MemoryIndex, terms []string, topK int) []ScoredDoc {
	if len(terms) == 0 {
		return nil
	}

	// Attempt strict AND intersection
	andCandidates := idx.IntersectAND(terms)
	if len(andCandidates) >= topK || len(terms) == 1 {
		return r.ScoreCandidates(idx, andCandidates, terms, topK)
	}

	// Relax to OR: collect all unique docIDs matching any term
	docIDSet := make(map[uint32]struct{})
	for _, docID := range andCandidates {
		docIDSet[docID] = struct{}{}
	}
	for _, term := range terms {
		for _, p := range idx.GetPostings(term) {
			docIDSet[p.DocID] = struct{}{}
		}
	}

	candidates := make([]uint32, 0, len(docIDSet))
	for docID := range docIDSet {
		candidates = append(candidates, docID)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i] < candidates[j]
	})

	return r.ScoreCandidates(idx, candidates, terms, topK)
}
