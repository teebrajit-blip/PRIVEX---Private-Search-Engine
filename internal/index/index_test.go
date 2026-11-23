package index

import (
	"reflect"
	"testing"
)

func TestIndexAddAndRetrieve(t *testing.T) {
	idx := NewMemoryIndex()

	doc1 := Document{
		ExternalID:  101,
		URL:         "https://example.com/go",
		Title:       "Go Programming",
		TotalLength: 5,
	}
	// "go": [1, 4], "fast": [2], "simple": [3]
	idx.AddDocument(doc1, map[string][]uint32{
		"go":     {1, 4},
		"fast":   {2},
		"simple": {3},
	})

	doc2 := Document{
		ExternalID:  102,
		URL:         "https://example.com/rust",
		Title:       "Rust Language",
		TotalLength: 4,
	}
	// "rust": [1], "fast": [2], "safe": [3]
	idx.AddDocument(doc2, map[string][]uint32{
		"rust": {1},
		"fast": {2},
		"safe": {3},
	})

	if idx.TotalDocs() != 2 {
		t.Fatalf("expected 2 documents, got %d", idx.TotalDocs())
	}

	if idx.AvgDocLength() != 4.5 {
		t.Fatalf("expected avg doc length 4.5, got %f", idx.AvgDocLength())
	}

	// AND intersection: "fast" is in doc 1 and doc 2
	fastDocs := idx.IntersectAND([]string{"fast"})
	if !reflect.DeepEqual(fastDocs, []uint32{1, 2}) {
		t.Errorf("expected [1, 2] for 'fast', got %v", fastDocs)
	}

	// AND intersection: "go" AND "fast" is only doc 1
	goFastDocs := idx.IntersectAND([]string{"go", "fast"})
	if !reflect.DeepEqual(goFastDocs, []uint32{1}) {
		t.Errorf("expected [1] for 'go' AND 'fast', got %v", goFastDocs)
	}
}

func TestPhraseMatching(t *testing.T) {
	idx := NewMemoryIndex()

	// Doc 1: "to be or not to be"
	// positions: to: [1, 5], be: [2, 6], or: [3], not: [4]
	idx.AddDocument(Document{ExternalID: 1, Title: "Hamlet", TotalLength: 6}, map[string][]uint32{
		"to":  {1, 5},
		"be":  {2, 6},
		"or":  {3},
		"not": {4},
	})

	// Doc 2: "be to not or" (same words, scrambled positions)
	idx.AddDocument(Document{ExternalID: 2, Title: "Scrambled", TotalLength: 4}, map[string][]uint32{
		"be":  {1},
		"to":  {2},
		"not": {3},
		"or":  {4},
	})

	// Test phrase "to be" -> should match Doc 1 only!
	res := idx.MatchPhrase([]string{"to", "be"})
	if !reflect.DeepEqual(res, []uint32{1}) {
		t.Errorf("phrase 'to be' expected [1], got %v", res)
	}

	// Test phrase "not to be" -> should match Doc 1 (pos 4, 5, 6)
	res3 := idx.MatchPhrase([]string{"not", "to", "be"})
	if !reflect.DeepEqual(res3, []uint32{1}) {
		t.Errorf("phrase 'not to be' expected [1], got %v", res3)
	}

	// Test non-matching phrase "be not"
	resNone := idx.MatchPhrase([]string{"be", "not"})
	if len(resNone) != 0 {
		t.Errorf("phrase 'be not' expected empty, got %v", resNone)
	}
}
