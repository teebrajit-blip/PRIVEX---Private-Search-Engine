package query

import (
	"reflect"
	"testing"

	"privex/internal/analysis"
)

func TestParseTermsAndPhrases(t *testing.T) {
	analyzer := analysis.DefaultAnalyzer()
	parser := NewParser(analyzer)

	q := parser.Parse(`privacy search "open source" "fast retrieval"`)

	if !q.HasPhrases {
		t.Fatalf("expected HasPhrases to be true")
	}

	if len(q.Phrases) != 2 {
		t.Fatalf("expected 2 phrases, got %d", len(q.Phrases))
	}

	expectedPhrase1 := []string{"open", "sourc"} // stemmed
	expectedPhrase2 := []string{"fast", "retriev"} // stemmed
	if !reflect.DeepEqual(q.Phrases[0], expectedPhrase1) {
		t.Errorf("phrase 1 mismatch: got %v, want %v", q.Phrases[0], expectedPhrase1)
	}
	if !reflect.DeepEqual(q.Phrases[1], expectedPhrase2) {
		t.Errorf("phrase 2 mismatch: got %v, want %v", q.Phrases[1], expectedPhrase2)
	}

	expectedTerms := []string{"privaci", "search"} // stemmed
	if !reflect.DeepEqual(q.Terms, expectedTerms) {
		t.Errorf("terms mismatch: got %v, want %v", q.Terms, expectedTerms)
	}
}
