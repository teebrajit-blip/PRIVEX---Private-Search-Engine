package ranking

import (
	"math"
	"testing"

	"privex/internal/index"
)

func TestIDFAlwaysPositive(t *testing.T) {
	// Section 2 Correction #9: idf = ln(1 + (N - n + 0.5)/(n + 0.5)) must always be > 0
	tests := []struct {
		N  uint32
		df uint32
	}{
		{N: 100, df: 100}, // Term in every single document
		{N: 100, df: 1},   // Rare term
		{N: 1, df: 1},     // Single document corpus
		{N: 1000000, df: 1000000},
	}

	for _, tt := range tests {
		idf := ComputeIDF(tt.N, tt.df)
		if idf <= 0 {
			t.Errorf("ComputeIDF(%d, %d) = %f, expected > 0", tt.N, tt.df, idf)
		}
	}
}

func TestBM25HandComputedExample(t *testing.T) {
	// Section 8 (Evaluation Plan): BM25 scores verified against hand-computed example.
	// N = 2, doc1 len = 5, doc2 len = 4. avgdl = 4.5.
	// term 'search' in doc1 with tf = 2, df = 1.
	// idf = ln(1 + (2 - 1 + 0.5)/(1 + 0.5)) = ln(1 + 1) = ln(2) approx 0.693147
	// k1 = 1.2, b = 0.75
	// lenNorm = 1 - 0.75 + 0.75 * (5.0 / 4.5) = 0.25 + 0.833333333 = 1.083333333
	// saturation = (2.0 * 2.2) / (2.0 + 1.2 * 1.083333333) = 4.4 / (2.0 + 1.3) = 4.4 / 3.3 = 4/3
	// expected score = ln(2) * (4.0 / 3.0) approx 0.92419624

	N := uint32(2)
	df := uint32(1)
	tf := 2.0
	docLen := 5.0
	avgdl := 4.5
	k1 := 1.2
	b := 0.75

	idf := ComputeIDF(N, df)
	expectedIDF := math.Log(2.0)
	if math.Abs(idf-expectedIDF) > 1e-6 {
		t.Fatalf("idf mismatch: got %f, want %f", idf, expectedIDF)
	}

	score := ScoreTerm(tf, docLen, avgdl, idf, k1, b)
	expectedScore := math.Log(2.0) * (4.0 / 3.0)
	if math.Abs(score-expectedScore) > 1e-6 {
		t.Fatalf("score mismatch: got %f, want %f", score, expectedScore)
	}
}

func TestRankerTopK(t *testing.T) {
	idx := index.NewMemoryIndex()

	idx.AddDocument(index.Document{
		ExternalID:  1,
		Title:       "Low occurrence",
		TotalLength: 10,
	}, map[string][]uint32{
		"privacy": {1},
	})

	idx.AddDocument(index.Document{
		ExternalID:  2,
		Title:       "High occurrence short doc",
		TotalLength: 5,
	}, map[string][]uint32{
		"privacy": {1, 3, 5},
	})

	idx.AddDocument(index.Document{
		ExternalID:  3,
		Title:       "High occurrence long doc",
		TotalLength: 20,
	}, map[string][]uint32{
		"privacy": {1, 5, 10},
	})

	ranker := NewRanker(DefaultParams())
	results := ranker.Search(idx, []string{"privacy"}, 3)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Doc 2 should rank highest due to higher TF (3) and shorter document length (5)
	if results[0].ExternalID != 2 {
		t.Errorf("expected top result to be Doc 2, got %d", results[0].ExternalID)
	}

	// Scores should strictly decrease
	if results[0].Score < results[1].Score || results[1].Score < results[2].Score {
		t.Errorf("results not strictly sorted descending: %v", results)
	}
}
