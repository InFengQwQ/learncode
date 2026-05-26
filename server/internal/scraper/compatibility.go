package scraper

import "strings"

// CompatibilityModel determines the compatibility model for a programming language.
// It uses only Wikipedia data — no hardcoded knowledge about any specific language.
func CompatibilityModel(name string, categories []string, info *InfoboxData) string {
	// Esoteric languages don't have versioning concerns
	for _, c := range categories {
		if strings.Contains(strings.ToLower(c), "esoteric") {
			return "none"
		}
	}
	// Everything else: default to "versioned" — most languages have multiple versions.
	// The user can adjust later if needed.
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
