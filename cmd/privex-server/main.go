package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"privex/internal/analysis"
	"privex/internal/api"
	"privex/internal/index"
)

type CorpusDoc struct {
	ID    int64  `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	corpus := flag.String("corpus", "testdata/corpus.json", "JSON corpus")
	flag.Parse()

	data, err := os.ReadFile(*corpus)
	if err != nil {
		log.Fatalf("read corpus: %v", err)
	}
	var corpusFile struct {
		Documents []CorpusDoc `json:"documents"`
	}
	if err := json.Unmarshal(data, &corpusFile); err != nil {
		log.Fatalf("parse corpus: %v", err)
	}

	analyzer := analysis.DefaultAnalyzer()
	idx := index.NewMemoryIndex()
	for _, doc := range corpusFile.Documents {
		tokens := analyzer.Analyze(doc.Title + " " + doc.Body)
		positions := make(map[string][]uint32)
		for _, token := range tokens {
			positions[token.Term] = append(positions[token.Term], token.Position)
		}
		titleTokens := analyzer.Analyze(doc.Title)
		bodyTokens := analyzer.Analyze(doc.Body)
		idx.AddDocument(index.Document{
			ExternalID:  doc.ID,
			URL:         doc.URL,
			Title:       doc.Title,
			Snippet:     doc.Body,
			Lengths:     index.FieldLengths{Title: uint32(len(titleTokens)), Body: uint32(len(bodyTokens)), Total: uint32(len(tokens))},
			TotalLength: uint32(len(tokens)),
		}, positions)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           api.New(idx).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Printf("PRIVEX listening on %s", *addr)
	log.Fatal(server.ListenAndServe())
}
