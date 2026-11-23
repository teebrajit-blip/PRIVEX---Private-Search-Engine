package analysis

import (
	"strings"
)

// Stemmer defines an interface for word stemming.
type Stemmer interface {
	Stem(word string) string
}

// PorterStemmer implements the Porter Stemming Algorithm (English).
type PorterStemmer struct{}

// NewPorterStemmer creates a new PorterStemmer instance.
func NewPorterStemmer() *PorterStemmer {
	return &PorterStemmer{}
}

func isConsonant(s []rune, i int) bool {
	switch s[i] {
	case 'a', 'e', 'i', 'o', 'u':
		return false
	case 'y':
		if i == 0 {
			return true
		}
		return !isConsonant(s, i-1)
	default:
		return true
	}
}

// measure (m) of a word or part of word: [C](VC)^m[V]
func measure(s []rune) int {
	k := 0
	c := 0
	n := len(s)
	for k < n && isConsonant(s, k) {
		k++
	}
	for k < n {
		for k < n && !isConsonant(s, k) {
			k++
		}
		for k < n && isConsonant(s, k) {
			k++
		}
		c++
	}
	return c
}

func containsVowel(s []rune) bool {
	for i := range s {
		if !isConsonant(s, i) {
			return true
		}
	}
	return false
}

func endsWithDoubleConsonant(s []rune) bool {
	n := len(s)
	if n < 2 {
		return false
	}
	return s[n-1] == s[n-2] && isConsonant(s, n-1)
}

func cvc(s []rune) bool {
	n := len(s)
	if n < 3 {
		return false
	}
	if !isConsonant(s, n-1) || isConsonant(s, n-2) || !isConsonant(s, n-3) {
		return false
	}
	c := s[n-1]
	return c != 'w' && c != 'x' && c != 'y'
}

// Stem applies the standard Porter stemming rules to a lowercase word.
func (ps *PorterStemmer) Stem(word string) string {
	s := []rune(strings.ToLower(word))
	n := len(s)
	if n <= 2 {
		return word
	}

	// Step 1a
	if strings.HasSuffix(string(s), "sses") {
		s = s[:n-2]
	} else if strings.HasSuffix(string(s), "ies") {
		s = s[:n-2]
	} else if !strings.HasSuffix(string(s), "ss") && strings.HasSuffix(string(s), "s") {
		s = s[:n-1]
	}

	// Step 1b
	n = len(s)
	step1bExtra := false
	if strings.HasSuffix(string(s), "eed") {
		if measure(s[:n-3]) > 0 {
			s = s[:n-1]
		}
	} else if strings.HasSuffix(string(s), "ed") {
		if containsVowel(s[:n-2]) {
			s = s[:n-2]
			step1bExtra = true
		}
	} else if strings.HasSuffix(string(s), "ing") {
		if containsVowel(s[:n-3]) {
			s = s[:n-3]
			step1bExtra = true
		}
	}

	if step1bExtra {
		n = len(s)
		if strings.HasSuffix(string(s), "at") || strings.HasSuffix(string(s), "bl") || strings.HasSuffix(string(s), "iz") {
			s = append(s, 'e')
		} else if endsWithDoubleConsonant(s) && !(s[n-1] == 'l' || s[n-1] == 's' || s[n-1] == 'z') {
			s = s[:n-1]
		} else if measure(s) == 1 && cvc(s) {
			s = append(s, 'e')
		}
	}

	// Step 1c
	n = len(s)
	if strings.HasSuffix(string(s), "y") && containsVowel(s[:n-1]) {
		s[n-1] = 'i'
	}

	// Step 2
	n = len(s)
	type stepRule struct {
		suffix string
		repl   string
	}
	step2Rules := []stepRule{
		{"ational", "ate"}, {"tional", "tion"}, {"enci", "ence"}, {"anci", "ance"},
		{"izer", "ize"}, {"abli", "able"}, {"alli", "al"}, {"entli", "ent"},
		{"eli", "e"}, {"ousli", "ous"}, {"ization", "ize"}, {"ation", "ate"},
		{"ator", "ate"}, {"alism", "al"}, {"iveness", "ive"}, {"fulness", "ful"},
		{"ousness", "ous"}, {"aliti", "al"}, {"iviti", "ive"}, {"biliti", "ble"},
	}
	for _, rule := range step2Rules {
		if strings.HasSuffix(string(s), rule.suffix) {
			stem := s[:n-len(rule.suffix)]
			if measure(stem) > 0 {
				s = append(stem, []rune(rule.repl)...)
			}
			break
		}
	}

	// Step 3
	n = len(s)
	step3Rules := []stepRule{
		{"icate", "ic"}, {"ative", ""}, {"alize", "al"},
		{"iciti", "ic"}, {"ical", "ic"}, {"ful", ""}, {"ness", ""},
	}
	for _, rule := range step3Rules {
		if strings.HasSuffix(string(s), rule.suffix) {
			stem := s[:n-len(rule.suffix)]
			if measure(stem) > 0 {
				s = append(stem, []rune(rule.repl)...)
			}
			break
		}
	}

	// Step 4
	n = len(s)
	step4Suffixes := []string{
		"al", "ance", "ence", "er", "ic", "able", "ible", "ant", "ement",
		"ment", "ent", "ou", "ism", "ate", "iti", "ous", "ive", "ize",
	}
	for _, suffix := range step4Suffixes {
		if strings.HasSuffix(string(s), suffix) {
			stem := s[:n-len(suffix)]
			if measure(stem) > 1 {
				s = stem
			}
			break
		}
	}

	// Step 5
	n = len(s)
	if strings.HasSuffix(string(s), "e") {
		stem := s[:n-1]
		m := measure(stem)
		if m > 1 || (m == 1 && !cvc(stem)) {
			s = stem
		}
	}
	n = len(s)
	if n > 1 && endsWithDoubleConsonant(s) && s[n-1] == 'l' && measure(s[:n-1]) > 1 {
		s = s[:n-1]
	}

	return string(s)
}
