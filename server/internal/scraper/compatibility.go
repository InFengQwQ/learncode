package scraper

import "strings"

// CompatibilityModel determines the compatibility model for a programming language.
// This is a deterministic Go-level decision — the LLM is NOT asked to classify this
// because small/local models produce inconsistent results (e.g., Rust flip-flopping
// between "strict" and "versioned" across queries).
//
// "strict": The language has a single authoritative specification and is backward-
// compatible by design. Old code runs on new versions without modification.
//
// "versioned": The language has multiple incompatible major versions. Code written
// for one version does NOT run on another. Users must choose a specific version.

func CompatibilityModel(name string, categories []string, info *InfoboxData) string {
	lower := strings.ToLower(name)

	// --- well-known strict languages ---
	// These are languages that intentionally maintain backward compatibility.
	strict := map[string]bool{
		"java": true, "go": true, "golang": true,
		"rust": true, "c#": true, "csharp": true,
		"kotlin": true, "swift": true, "dart": true,
		"typescript": true, "scala": true, "groovy": true,
		"haskell": true, "erlang": true, "elixir": true,
		"clojure": true, "ocaml": true, "f#": true, "fsharp": true,
		"zig": true, "nim": true, "crystal": true,
		"elisp": true, "emacs lisp": true, "scheme": true,
		"racket": true, "common lisp": true,
		"d": true, "dlang": true, "vala": true,
		"haxe": true, "reasonml": true, "elm": true,
		"purescript": true, "idris": true, "agda": true,
	}

	// --- well-known versioned languages ---
	// These languages have had major breaking changes between versions.
	versioned := map[string]bool{
		"python": true, "c++": true, "cpp": true, "c": true,
		"ruby": true, "php": true, "perl": true, "raku": true,
		"lua": true, "julia": true, "fortran": true,
		"cobol": true, "pascal": true, "ada": true,
		"matlab": true, "r": true, "actionscript": true,
		"delphi": true, "object pascal": true,
		"powershell": true, "basic": true, "visual basic": true,
	}

	if strict[lower] {
		return "strict"
	}
	if versioned[lower] {
		return "versioned"
	}

	// --- heuristic fallback for unknown languages ---
	// Check Wikipedia categories for versioning signals.
	for _, c := range categories {
		lowerCat := strings.ToLower(c)
		if strings.Contains(lowerCat, "esoteric") {
			return "none"
		}
	}

	// Check infobox data: if the language has a stable release and first appeared
	// more than 10 years ago, but no mention of major version splits, it's likely strict.
	// Conservative default: "strict" (single active version is the safer default).
	if info != nil && info.FirstAppeared != "" {
		// Languages that first appeared long ago and still have a single spec → strict
		return "strict"
	}

	// If nothing is known, default to "strict" — most modern languages maintain
	// backward compatibility. The user can adjust later if needed.
	return "strict"
}

// IsEsoteric determines whether a language is esoteric (deliberately minimal,
// obfuscated, or a joke language) by checking Wikipedia categories.
func IsEsoteric(categories []string) bool {
	for _, c := range categories {
		lower := strings.ToLower(c)
		if strings.Contains(lower, "esoteric programming language") {
			return true
		}
	}
	return false
}
