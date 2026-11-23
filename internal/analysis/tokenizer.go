package analysis

// Token represents a single token extracted from text, along with its 1-based sequential position.
type Token struct {
	Term     string
	Position uint32
	Offset   int // byte offset in original text
}

// Tokenizer breaks text into a sequence of normalized tokens while tracking positions.
type Tokenizer struct{}

// NewTokenizer creates a new Tokenizer.
func NewTokenizer() *Tokenizer {
	return &Tokenizer{}
}

// isAlphanumeric reports whether the rune is an ASCII/Unicode letter or digit.
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		(r >= 0x00C0 && r <= 0x02AF) || // Latin extended
		(r >= 0x0370 && r <= 0x03FF) || // Greek
		(r >= 0x0400 && r <= 0x04FF) || // Cyrillic
		(r >= 0x0900 && r <= 0x0D7F) || // Indic scripts (Devanagari, Bengali, etc.)
		(r >= 0x4E00 && r <= 0x9FFF) // CJK unified ideographs
}

// Tokenize converts an input string into a slice of Tokens with original sequential positions.
// It preserves technical tokens like "c++", "c#", and version numbers like "go1.22".
func (t *Tokenizer) Tokenize(text string) []Token {
	var tokens []Token
	runes := []rune(text)
	n := len(runes)
	pos := uint32(1)

	i := 0
	for i < n {
		// Skip non-alphanumeric runes
		for i < n && !isAlphanumeric(runes[i]) {
			i++
		}
		if i >= n {
			break
		}

		start := i
		// Consume alphanumeric sequence
		for i < n && isAlphanumeric(runes[i]) {
			i++
		}

		// Check for programming suffixes like "c++" or "c#" or version dots
		if i < n {
			if runes[i] == '+' && i+1 < n && runes[i+1] == '+' {
				i += 2
			} else if runes[i] == '#' {
				i++
			} else if runes[i] == '.' && i+1 < n && (runes[i+1] >= '0' && runes[i+1] <= '9') {
				// E.g. 1.22 or go1.22
				for i < n && (runes[i] == '.' || (runes[i] >= '0' && runes[i] <= '9')) {
					i++
				}
			}
		}

		tokenStr := string(runes[start:i])
		tokens = append(tokens, Token{
			Term:     tokenStr,
			Position: pos,
			Offset:   start,
		})
		pos++
	}

	return tokens
}
