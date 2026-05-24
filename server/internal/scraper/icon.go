package scraper

import "strings"

// IconEmoji returns an emoji for a well-known programming language.
// Returns "" if the language is not in the mapping — callers should
// use a placeholder emoji in that case.
//
// This is a FALLBACK used only when Wikipedia has no page image.
// The mapping is intentional — these are the Unicode-recognized
// representations closest to each language's identity.
func IconEmoji(name string) string {
	lower := strings.ToLower(name)
	emoji, ok := langEmoji[lower]
	if !ok {
		return ""
	}
	return emoji
}

var langEmoji = map[string]string{
	// General purpose
	"python":     "🐍",
	"java":       "☕",
	"javascript": "🟨",
	"typescript": "🟦",
	"go":         "🔷",
	"golang":     "🔷",
	"rust":       "🦀",
	"c":          "⚙️",
	"c++":        "🔧",
	"cpp":        "🔧",
	"c#":         "🟪",
	"csharp":     "🟪",
	"ruby":       "💎",
	"php":        "🐘",
	"swift":      "🕊️",
	"kotlin":     "💜",
	"scala":      "🔴",
	"dart":       "🎯",
	"elixir":     "💧",
	"clojure":    "☯️",
	"haskell":    "λ",
	"erlang":     "📡",
	"julia":      "🔢",
	"lua":        "🌙",
	"perl":       "🐪",
	"groovy":     "⭐",
	"zig":        "⚡",
	"nim":        "👑",
	"crystal":    "💎",
	"raku":       "🦋",
	"f#":         "🔷",
	"fsharp":     "🔷",

	// Systems
	"assembly": "🔩",
	"asm":      "🔩",

	// Shell / scripting
	"bash":       "💻",
	"powershell": "💙",

	// Data / scientific
	"r":       "📊",
	"matlab":  "📐",
	"fortran": "🔢",
	"cobol":   "📋",
	"pascal":  "📝",
	"ada":     "🛡️",
	"delphi":  "🏛️",

	// Mobile
	"objective-c": "🍎",

	// Database
	"sql": "🗄️",

	// Markup / domain-specific
	"html": "🌐",
	"css":  "🎨",
	"xml":  "📋",

	// Esoteric
	"brainfuck": "🧠",
	"lolcode":   "😂",
	"piet":      "🎨",
	"whitespace": "⬜",
	"befunge":   "🔀",
}
