package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"privex/internal/analysis"
	"privex/internal/index"
	"privex/internal/query"
	"privex/internal/ranking"
	"privex/internal/snippet"
)

type CorpusDoc struct {
	ID    int64  `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type CorpusFile struct {
	Documents []CorpusDoc `json:"documents"`
}

func main() {
	corpusPath := flag.String("corpus", "", "Path to JSON corpus file (defaults to testdata/corpus.json)")
	queryFlag := flag.String("query", "", "Single query to execute in batch mode")
	topK := flag.Int("k", 5, "Number of top results to return")
	flag.Parse()

	defaultPath := filepath.Join("testdata", "corpus.json")
	if *corpusPath == "" {
		if _, err := os.Stat(defaultPath); err == nil {
			*corpusPath = defaultPath
		}
	}

	fmt.Println("=================================================================")
	fmt.Println("  PRIVEX Search Engine — Milestone 1 In-Memory Retrieval Core   ")
	fmt.Println("=================================================================")

	analyzer := analysis.DefaultAnalyzer()
	idx := index.NewMemoryIndex()
	ranker := ranking.NewRanker(ranking.DefaultParams())
	queryParser := query.NewParser(analyzer)

	// Load corpus
	if *corpusPath != "" {
		docs, err := loadCorpus(*corpusPath)
		if err != nil {
			fmt.Printf("Error reading corpus from %s: %v\n", *corpusPath, err)
			os.Exit(1)
		}

		start := time.Now()
		for _, d := range docs {
			indexDocument(idx, analyzer, d)
		}
		duration := time.Since(start)
		fmt.Printf("Indexed %d documents in %v (avg doc length: %.2f tokens)\n\n",
			idx.TotalDocs(), duration, idx.AvgDocLength())
	} else {
		fmt.Println("No corpus file provided or found at testdata/corpus.json.")
		os.Exit(1)
	}

	if *queryFlag != "" {
		runQuery(idx, ranker, queryParser, *queryFlag, *topK)
		return
	}

	// Interactive mode
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter search query (supports terms and \"quoted phrases\"), or 'exit' to quit:")
	for {
		fmt.Print("\nsearch> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "exit" || line == "quit" {
			fmt.Println("Goodbye.")
			break
		}
		if line == "" {
			continue
		}
		runQuery(idx, ranker, queryParser, line, *topK)
	}
}

func loadCorpus(path string) ([]CorpusDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file CorpusFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return file.Documents, nil
}

func indexDocument(idx *index.MemoryIndex, analyzer *analysis.Analyzer, d CorpusDoc) {
	combined := d.Title + " " + d.Body
	tokens := analyzer.Analyze(combined)

	termPositions := make(map[string][]uint32)
	for _, tok := range tokens {
		termPositions[tok.Term] = append(termPositions[tok.Term], tok.Position)
	}

	titleTokens := analyzer.Analyze(d.Title)
	bodyTokens := analyzer.Analyze(d.Body)

	doc := index.Document{
		ExternalID: d.ID,
		URL:        d.URL,
		Title:      d.Title,
		Snippet:    d.Body,
		Lengths: index.FieldLengths{
			Title: uint32(len(titleTokens)),
			Body:  uint32(len(bodyTokens)),
			Total: uint32(len(tokens)),
		},
		TotalLength: uint32(len(tokens)),
	}

	idx.AddDocument(doc, termPositions)
}

func runQuery(idx *index.MemoryIndex, ranker *ranking.Ranker, parser *query.Parser, rawQuery string, topK int) {
	start := time.Now()
	pq := parser.Parse(rawQuery)

	var results []ranking.ScoredDoc

	if pq.HasPhrases && len(pq.Terms) == 0 {
		// Pure phrase search
		var phraseDocIDs []uint32
		for _, phrase := range pq.Phrases {
			matched := idx.MatchPhrase(phrase)
			if len(phraseDocIDs) == 0 {
				phraseDocIDs = matched
			} else {
				// Intersect if multiple phrases
				phraseDocIDs = intersectUint32(phraseDocIDs, matched)
			}
		}
		// Score matched phrase docs using phrase terms
		var allPhraseTerms []string
		for _, p := range pq.Phrases {
			allPhraseTerms = append(allPhraseTerms, p...)
		}
		results = ranker.ScoreCandidates(idx, phraseDocIDs, allPhraseTerms, topK)
	} else {
		// General terms search (with or without phrases)
		results = ranker.Search(idx, pq.Terms, topK)
	}

	latency := time.Since(start)

	fmt.Printf("\nQuery: %q (Parsed %d terms, %d phrases) | Latency: %v\n",
		rawQuery, len(pq.Terms), len(pq.Phrases), latency)
	fmt.Printf("Results found: %d\n", len(results))
	fmt.Println("-----------------------------------------------------------------")

	if len(results) == 0 {
		fmt.Println("No matching documents found.")
		return
	}

	for rank, res := range results {
		snipOpts := snippet.DefaultOptions()
		snipOpts.TagStart = "\033[1;33m" // Yellow highlight for terminal
		snipOpts.TagEnd = "\033[0m"
		snip := snippet.Generate(res.Document.Snippet, pq.Terms, snipOpts)

		fmt.Printf("[%d] Score: %.4f | ID: %d | %s\n", rank+1, res.Score, res.ExternalID, res.Document.Title)
		fmt.Printf("    URL: %s\n", res.Document.URL)
		fmt.Printf("    Snippet: %s\n\n", snip)
	}
}

func intersectUint32(a, b []uint32) []uint32 {
	m := make(map[uint32]bool)
	for _, x := range a {
		m[x] = true
	}
	var res []uint32
	for _, x := range b {
		if m[x] {
			res = append(res, x)
		}
	}
	return res
}
