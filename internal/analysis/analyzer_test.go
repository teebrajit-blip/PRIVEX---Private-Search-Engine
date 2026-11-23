package analysis

import (
	"reflect"
	"testing"
)

func TestTokenizerPositions(t *testing.T) {
	tok := NewTokenizer()
	tokens := tok.Tokenize("To be, or not to be: that is the question.")

	expectedTerms := []string{"to", "be", "or", "not", "to", "be", "that", "is", "the", "question"}
	if len(tokens) != len(expectedTerms) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTerms), len(tokens))
	}

	for i, expected := range expectedTerms {
		if tokens[i].Term != expected {
			t.Errorf("token %d: expected term %q, got %q", i, expected, tokens[i].Term)
		}
		expectedPos := uint32(i + 1)
		if tokens[i].Position != expectedPos {
			t.Errorf("token %d: expected position %d, got %d", i, expectedPos, tokens[i].Position)
		}
	}
}

func TestTechnicalTokens(t *testing.T) {
	tok := NewTokenizer()
	tokens := tok.Tokenize("learning c++ and go1.22 architecture")

	var terms []string
	for _, tok := range tokens {
		terms = append(terms, tok.Term)
	}

	expected := []string{"learning", "c++", "and", "go1.22", "architecture"}
	if !reflect.DeepEqual(terms, expected) {
		t.Errorf("expected %v, got %v", expected, terms)
	}
}

func TestStemmer(t *testing.T) {
	stemmer := NewPorterStemmer()
	tests := map[string]string{
		"connections": "connect",
		"connecting":  "connect",
		"retrieval":   "retriev",
		"search":      "search",
		"engines":     "engin",
	}

	for in, expected := range tests {
		got := stemmer.Stem(in)
		if got != expected {
			t.Errorf("Stem(%q) = %q; want %q", in, got, expected)
		}
	}
}

func TestAnalyzerNoStopwordDrop(t *testing.T) {
	// Section 2 Correction #5: Stop words must never be removed at indexing time
	analyzer := DefaultAnalyzer()
	tokens := analyzer.Analyze("the who")

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Position != 1 || tokens[1].Position != 2 {
		t.Errorf("positions incorrect: %v", tokens)
	}
}
