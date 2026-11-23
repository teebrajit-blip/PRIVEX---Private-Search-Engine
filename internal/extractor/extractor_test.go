package extractor

import (
	"strings"
	"testing"
)

func TestExtractDocument(t *testing.T) {
	input := `<html lang="en"><head><title>Privacy Search</title><meta name="description" content="Private search"><link rel="canonical" href="/docs/privacy"></head><body><main><h1>Privacy</h1><p>Search results stay private.</p></main><a href="/next">Next page</a></body></html>`
	doc, err := Extract("https://example.com/docs/start", strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "Privacy Search" {
		t.Fatalf("title = %q", doc.Title)
	}
	if doc.Description != "Private search" {
		t.Fatalf("description = %q", doc.Description)
	}
	if doc.Canonical != "https://example.com/docs/privacy" {
		t.Fatalf("canonical = %q", doc.Canonical)
	}
	if len(doc.Links) != 1 || doc.Links[0].URL != "https://example.com/next" {
		t.Fatalf("links = %#v", doc.Links)
	}
}
