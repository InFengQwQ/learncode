package scraper

import "strings"

func IsEsoteric(categories []string) bool {
	for _, c := range categories {
		if strings.Contains(strings.ToLower(c), "esoteric programming language") {
			return true
		}
	}
	return false
}
