package scraper

import (
	"testing"
)

func TestNormalizeTitleEdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Python (programming language)", "Python"},
		{"C (programming language)", "C"},
		{"C++", "C++"},
		{"F# (programming language)", "F#"},
		{"HTML (programming language)", "HTML"},
		{"  R (programming language)  ", "R"},
		{"Python", "Python"},
		{"Java", "Java"},
		{"Java (programming language)", "Java"},
		{"React (JavaScript library)", "React (JavaScript library)"},
		{"", ""},
		{"  ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeTitle(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeSlugEdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"python", "python"},
		{"Python", "python"},
		{"C++", "c"},
		{"C#", "c"},
		{"F#", "f"},
		{"c sharp", "c-sharp"},
		{"Go Language", "go-language"},
		{"ALGOL 68", "algol-68"},
		{"", ""},
		{"  ", ""},
		{"---", ""},
		{"hello---world", "hello-world"},
		{"hello   world", "hello-world"},
		{"hello___world", "hello-world"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeSlug(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
