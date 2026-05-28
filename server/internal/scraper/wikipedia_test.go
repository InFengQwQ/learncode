package scraper

import (
	"testing"
)

func TestRemoveStyleAndScript(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes style block with CSS",
			input: `<style>.mw-parser-output .plainlist ol{line-height:inherit}</style>oracle.com`,
			want:  `oracle.com`,
		},
		{
			name:  "removes style block with attributes",
			input: `<style data-mw-deduplicate="TemplateStyles:r123">.plainlist ul{list-style:none}</style>text`,
			want:  `text`,
		},
		{
			name:  "removes script block",
			input: `<script>alert('xss')</script>text`,
			want:  `text`,
		},
		{
			name:  "removes multiple blocks",
			input: `<style>.a{}</style>foo<style>.b{}</style>bar`,
			want:  `foobar`,
		},
		{
			name:  "no style blocks — unchanged",
			input: `<p>hello</p>`,
			want:  `<p>hello</p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeStyleAndScript(tt.input)
			if got != tt.want {
				t.Errorf("removeStyleAndScript() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractHrefFromRaw(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "simple href",
			html: `<a href="https://oracle.com/java/">link</a>`,
			want: "https://oracle.com/java/",
		},
		{
			name: "href with style block before anchor",
			html: `<style>.a{}</style><a href="https://python.org">python.org</a>`,
			want: "https://python.org",
		},
		{
			name: "no href — empty",
			html: `<a>no href</a>`,
			want: "",
		},
		{
			name: "single-quoted href",
			html: `<a href='https://golang.org'>go</a>`,
			want: "https://golang.org",
		},
		{
			name: "style content without href — empty",
			html: `<style>.mw-parser-output .plainlist ol,.mw-parser-output .plainlist ul{line-height:inherit;list-style:none;margin:0;padding:0}</style>oracle.com/java/`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHrefFromRaw(tt.html)
			if got != tt.want {
				t.Errorf("extractHrefFromRaw() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Python (programming language)", "Python"},
		{"Java (programming language)", "Java"},
		{"C++", "C++"},
		{"Go (programming language)", "Go"},
		{"Rust (programming_language)", "Rust"},
		{"Python (language)", "Python"},
		{"React (software)", "React"},
		{"  Python (programming language)  ", "Python"},
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

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"python", "python"},
		{"python-programming-language", "python-programming-language"},
		{"Python (programming language)", "python-programming-language"},
		{"C++", "c"},
		{"c-sharp", "c-sharp"},
		{"Java", "java"},
		{"go", "go"},
		{"", ""},
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

func TestStripTagsWithStyleBlock(t *testing.T) {
	// Reproduction case from the Java bug report:
	// Wikipedia infobox cells contain TemplateStyles that leak CSS into visible text.
	raw := `<style data-mw-deduplicate="TemplateStyles:r123">.mw-parser-output .plainlist ol,.mw-parser-output .plainlist ul{line-height:inherit;list-style:none;margin:0;padding:0}</style><a href="https://oracle.com/java/">oracle.com/java/</a>`
	got := stripTags(raw)

	// Should NOT contain CSS
	if contains(t, got, ".mw-parser-output") {
		t.Errorf("stripTags leaked CSS: %q", got)
	}
	if contains(t, got, "line-height") {
		t.Errorf("stripTags leaked CSS property: %q", got)
	}
	if contains(t, got, "plainlist") {
		t.Errorf("stripTags leaked CSS class: %q", got)
	}
	// Should contain the link text
	if !contains(t, got, "oracle.com") {
		t.Errorf("stripTags removed link text: %q", got)
	}
}

func contains(t *testing.T, s, substr string) bool {
	t.Helper()
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIsEsoteric(t *testing.T) {
	tests := []struct {
		cats []string
		want bool
	}{
		{[]string{"Programming languages", "Statically typed programming languages"}, false},
		{[]string{"Esoteric programming languages"}, true},
		{[]string{"Programming languages", "Esoteric programming languages"}, true},
		{nil, false},
	}

	for i, tt := range tests {
		got := IsEsoteric(tt.cats)
		if got != tt.want {
			t.Errorf("IsEsoteric(#%d) = %v, want %v", i, got, tt.want)
		}
	}
}

func TestIconEmoji(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"anything", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IconEmoji(tt.name)
			if got != tt.want {
				t.Errorf("IconEmoji(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
