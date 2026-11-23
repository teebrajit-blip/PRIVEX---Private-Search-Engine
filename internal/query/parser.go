package query

import (
	"strings"

	"privex/internal/analysis"
)

// QueryType distinguishes between term search, phrase search, and combined queries.
type QueryType int

const (
	QueryTypeTerms QueryType = iota
	QueryTypePhrase
	QueryTypeBoolean
)

// ParsedQuery represents the structured output of parsing a user's raw query string.
type ParsedQuery struct {
	Raw        string
	Terms      []string   // Analyzed individual terms
	Phrases    [][]string // Analyzed phrases (sequences of terms that must be adjacent)
	StopWords  []string   // Stop words identified in the query
	HasPhrases bool
}

// Parser transforms a raw search query into normalized terms and phrases using the shared Analyzer.
type Parser struct {
	analyzer *analysis.Analyzer
}

// NewParser creates a new query parser sharing the index analyzer.
func NewParser(analyzer *analysis.Analyzer) *Parser {
	return &Parser{analyzer: analyzer}
}

// Parse extracts terms and quoted phrases from the raw query.
// It uses the same analyzer that was used at index time (Section 5.3).
func (p *Parser) Parse(raw string) ParsedQuery {
	trimmed := strings.TrimSpace(raw)
	res := ParsedQuery{Raw: trimmed}
	if len(trimmed) == 0 {
		return res
	}

	inQuotes := false
	var currentPhrase []rune
	var regularText []rune

	runes := []rune(trimmed)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '"' {
			if inQuotes {
				// Close phrase
				phraseStr := string(currentPhrase)
				tokens := p.analyzer.Analyze(phraseStr)
				if len(tokens) > 0 {
					phraseTerms := make([]string, len(tokens))
					for idx, tok := range tokens {
						phraseTerms[idx] = tok.Term
					}
					res.Phrases = append(res.Phrases, phraseTerms)
					res.HasPhrases = true
				}
				currentPhrase = nil
				inQuotes = false
			} else {
				// Open phrase
				inQuotes = true
			}
		} else {
			if inQuotes {
				currentPhrase = append(currentPhrase, r)
			} else {
				regularText = append(regularText, r)
			}
		}
	}

	// Any unclosed quote is treated as normal text
	if inQuotes && len(currentPhrase) > 0 {
		regularText = append(regularText, currentPhrase...)
	}

	// Analyze regular terms outside quotes
	if len(regularText) > 0 {
		tokens := p.analyzer.Analyze(string(regularText))
		for _, tok := range tokens {
			if analysis.IsStopWord(tok.Term) {
				res.StopWords = append(res.StopWords, tok.Term)
			}
			res.Terms = append(res.Terms, tok.Term)
		}
	}

	return res
}
