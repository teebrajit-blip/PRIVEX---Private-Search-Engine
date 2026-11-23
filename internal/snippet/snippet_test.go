package snippet

import (
	"strings"
	"testing"
)

func TestSnippetHighlightingAndEscaping(t *testing.T) {
	// Verifies XSS prevention: script tags in document body must be escaped!
	body := "Welcome to PRIVEX. <script>alert('xss')</script> This is a privacy-first search engine."
	queryTerms := []string{"privacy-first", "engine"}

	s := Generate(body, queryTerms, DefaultOptions())

	if strings.Contains(s, "<script>") {
		t.Errorf("expected script tags to be HTML-escaped, got: %s", s)
	}
	if !strings.Contains(s, "&lt;script&gt;") {
		t.Errorf("expected &lt;script&gt;, got: %s", s)
	}
	if !strings.Contains(s, "<mark>") || !strings.Contains(s, "</mark>") {
		t.Errorf("expected highlighted terms with <mark>, got: %s", s)
	}
}
