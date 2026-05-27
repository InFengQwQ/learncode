package scraper

import "strings"

// CompatibilityModel returns a preliminary compatibility model based on Wikipedia categories.
// This is a fast Go-level heuristic — the LLM has final say during the analysis phase.
// Possible values:
//   - "none":      esoteric languages / DSLs without versioning concerns
//   - "versioned": default; most languages have multiple concurrent versions (Python, Rust, C++)
//   - "strict":    determined later by LLM; the language has a single stable version (Go, Java)
func CompatibilityModel(name string, categories []string, info *InfoboxData) string {
	for _, c := range categories {
		if strings.Contains(strings.ToLower(c), "esoteric") {
			return "none"
		}
	}
	return "versioned"
}

// IsEsoteric determines whether a language is esoteric by checking Wikipedia categories.
func IsEsoteric(categories []string) bool {
	for _, c := range categories {
		if strings.Contains(strings.ToLower(c), "esoteric programming language") {
			return true
		}
	}
	return false
}
