package analysis

import (
	"strings"
	"unicode"
)

// Standard English stop words
var EnglishStopWords = map[string]struct{}{
	"a": {}, "about": {}, "above": {}, "after": {}, "again": {}, "against": {}, "all": {}, "am": {},
	"an": {}, "and": {}, "any": {}, "are": {}, "aren't": {}, "as": {}, "at": {}, "be": {}, "because": {},
	"been": {}, "before": {}, "being": {}, "below": {}, "between": {}, "both": {}, "but": {}, "by": {},
	"can't": {}, "cannot": {}, "could": {}, "couldn't": {}, "did": {}, "didn't": {}, "do": {}, "does": {},
	"doesn't": {}, "doing": {}, "don't": {}, "down": {}, "during": {}, "each": {}, "few": {}, "for": {},
	"from": {}, "further": {}, "had": {}, "hadn't": {}, "has": {}, "hasn't": {}, "have": {}, "haven't": {},
	"having": {}, "he": {}, "he'd": {}, "he'll": {}, "he's": {}, "her": {}, "here": {}, "here's": {},
	"hers": {}, "herself": {}, "him": {}, "himself": {}, "his": {}, "how": {}, "how's": {}, "i": {},
	"i'd": {}, "i'll": {}, "i'm": {}, "i've": {}, "if": {}, "in": {}, "into": {}, "is": {}, "isn't": {},
	"it": {}, "it's": {}, "its": {}, "itself": {}, "let's": {}, "me": {}, "more": {}, "most": {},
	"mustn't": {}, "my": {}, "myself": {}, "no": {}, "nor": {}, "not": {}, "of": {}, "off": {}, "on": {},
	"once": {}, "only": {}, "or": {}, "other": {}, "ought": {}, "our": {}, "ours": {}, "ourselves": {},
	"out": {}, "over": {}, "own": {}, "same": {}, "shan't": {}, "she": {}, "she'd": {}, "she'll": {},
	"she's": {}, "should": {}, "shouldn't": {}, "so": {}, "some": {}, "such": {}, "than": {}, "that": {},
	"that's": {}, "the": {}, "their": {}, "theirs": {}, "them": {}, "themselves": {}, "then": {},
	"there": {}, "there's": {}, "these": {}, "they": {}, "they'd": {}, "they'll": {}, "they're": {},
	"they've": {}, "this": {}, "those": {}, "through": {}, "to": {}, "too": {}, "under": {}, "until": {},
	"up": {}, "very": {}, "was": {}, "wasn't": {}, "we": {}, "we'd": {}, "we'll": {}, "we're": {},
	"we've": {}, "were": {}, "weren't": {}, "what": {}, "what's": {}, "when": {}, "when's": {},
	"where": {}, "where's": {}, "which": {}, "while": {}, "who": {}, "who's": {}, "whom": {},
	"why": {}, "why's": {}, "with": {}, "won't": {}, "would": {}, "wouldn't": {}, "you": {},
	"you'd": {}, "you'll": {}, "you're": {}, "you've": {}, "your": {}, "yours": {}, "yourself": {},
	"yourselves": {},
}

// Config controls analyzer behaviors.
type Config struct {
	EnableStemming bool
	Language       string
}

// Analyzer orchestrates tokenization, normalization, and stemming.
// As specified in Section 5.3, the same Analyzer pipeline is applied identically at index and query time.
type Analyzer struct {
	tokenizer *Tokenizer
	stemmer   Stemmer
	cfg       Config
}

// NewAnalyzer creates an Analyzer with the given configuration.
func NewAnalyzer(cfg Config) *Analyzer {
	return &Analyzer{
		tokenizer: NewTokenizer(),
		stemmer:   NewPorterStemmer(),
		cfg:       cfg,
	}
}

// DefaultAnalyzer returns an Analyzer configured for English with stemming enabled.
func DefaultAnalyzer() *Analyzer {
	return NewAnalyzer(Config{
		EnableStemming: true,
		Language:       "en",
	})
}

// Normalize lowercases and normalizes whitespace/runes.
func (a *Analyzer) Normalize(s string) string {
	return strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s))
}

// Analyze processes text into tokens with their original positions preserved.
// Note: Stop words are NOT removed from the index (Section 2, correction 5).
func (a *Analyzer) Analyze(text string) []Token {
	normalized := a.Normalize(text)
	rawTokens := a.tokenizer.Tokenize(normalized)

	tokens := make([]Token, len(rawTokens))
	for i, tok := range rawTokens {
		term := tok.Term
		if a.cfg.EnableStemming {
			term = a.stemmer.Stem(term)
		}
		tokens[i] = Token{
			Term:     term,
			Position: tok.Position,
			Offset:   tok.Offset,
		}
	}
	return tokens
}

// IsStopWord checks if a word is in the stop word list.
func IsStopWord(term string) bool {
	_, ok := EnglishStopWords[strings.ToLower(term)]
	return ok
}
