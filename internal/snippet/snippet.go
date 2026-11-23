package snippet

import (
	"html"
	"strings"
	"unicode"
)

// Options configures snippet generation.
type Options struct {
	MaxLength int    // Maximum length of the snippet in characters (e.g. 160)
	TagStart  string // e.g. "<mark>"
	TagEnd    string // e.g. "</mark>"
}

// DefaultOptions returns sensible defaults for snippet generation.
func DefaultOptions() Options {
	return Options{
		MaxLength: 160,
		TagStart:  "<mark>",
		TagEnd:    "</mark>",
	}
}

// Generate selects the best window of text containing query terms,
// safely HTML-escapes all text, and highlights matched terms with <mark> tags (Section 5.6).
func Generate(body string, queryTerms []string, opts Options) string {
	if len(body) == 0 {
		return ""
	}

	bodyLower := strings.ToLower(body)

	// Find the earliest occurrence of any query term
	bestStart := -1
	for _, term := range queryTerms {
		if len(term) < 2 {
			continue
		}
		idx := strings.Index(bodyLower, strings.ToLower(term))
		if idx != -1 {
			if bestStart == -1 || idx < bestStart {
				bestStart = idx
			}
		}
	}

	if bestStart == -1 {
		// No direct match found, take the beginning
		bestStart = 0
	} else {
		// Provide slight leading context (approx 30 chars before term)
		if bestStart > 30 {
			bestStart -= 30
			// Wind forward to the next whitespace boundary
			for bestStart < len(body) && !unicode.IsSpace(rune(body[bestStart])) {
				bestStart++
			}
			if bestStart < len(body) {
				bestStart++ // skip whitespace
			}
		} else {
			bestStart = 0
		}
	}

	end := bestStart + opts.MaxLength
	prefix := ""
	suffix := ""
	if bestStart > 0 {
		prefix = "..."
	}
	if end < len(body) {
		suffix = "..."
	} else {
		end = len(body)
	}

	rawSnippet := body[bestStart:end]

	// HTML-escape before adding markup to prevent XSS
	escaped := html.EscapeString(rawSnippet)

	// Highlight matches
	highlighted := escaped
	for _, term := range queryTerms {
		if len(term) < 2 {
			continue
		}
		// Case-insensitive replacement
		termEscaped := html.EscapeString(term)
		termLower := strings.ToLower(termEscaped)
		textLower := strings.ToLower(highlighted)

		var buf strings.Builder
		last := 0
		for {
			idx := strings.Index(textLower[last:], termLower)
			if idx == -1 {
				buf.WriteString(highlighted[last:])
				break
			}
			actualIdx := last + idx
			buf.WriteString(highlighted[last:actualIdx])
			buf.WriteString(opts.TagStart)
			buf.WriteString(highlighted[actualIdx : actualIdx+len(termEscaped)])
			buf.WriteString(opts.TagEnd)
			last = actualIdx + len(termEscaped)
		}
		highlighted = buf.String()
	}

	return prefix + highlighted + suffix
}
