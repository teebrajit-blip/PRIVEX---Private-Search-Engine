package index

import (
	"os"
	"testing"
)

func TestSegmentRoundTrip(t *testing.T) {
	idx := NewMemoryIndex()
	idx.AddDocument(
		Document{ExternalID: 42, URL: "https://example.com", Title: "Example", Snippet: "body", TotalLength: 2},
		map[string][]uint32{"privacy": {1}, "search": {2}},
	)
	path := t.TempDir() + "/segment.json"
	if err := idx.WriteSegment(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadSegment(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TotalDocs() != 1 {
		t.Fatalf("total docs = %d", loaded.TotalDocs())
	}
	if loaded.DocFrequency("privacy") != 1 {
		t.Fatalf("privacy df = %d", loaded.DocFrequency("privacy"))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
