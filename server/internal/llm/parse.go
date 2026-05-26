package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseLLMJSON extracts valid JSON from LLM output using a 5-layer fallback
// strategy. Small models produce diverse formatting (fences, preambles, unescaped
// quotes in descriptions, trailing commas), so a single json.Unmarshal is
// insufficient. Each layer tries a progressively more aggressive fix.
func ParseLLMJSON(content string, v interface{}) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("empty llm response")
	}

	// Layer 1: Direct unmarshal (handles bare JSON from well-behaved models)
	if err := json.Unmarshal([]byte(content), v); err == nil {
		return nil
	}

	// Layer 2: Strip all markdown fence variations and try again.
	if cleaned := StripFences(content); cleaned != content {
		if err := json.Unmarshal([]byte(cleaned), v); err == nil {
			return nil
		}
	}

	// Layer 3: Extract the first complete {...} block using brace counting.
	extracted := ExtractJSONBlock(content)
	if extracted != "" {
		if err := json.Unmarshal([]byte(extracted), v); err == nil {
			return nil
		}

		// Layer 4: Fix common small-model JSON mistakes in the extracted block.
		if fixed := FixJSON(extracted); fixed != extracted {
			if err := json.Unmarshal([]byte(fixed), v); err == nil {
				return nil
			}
		}
	}

	// Layer 5: Regex field extraction as last resort (flat structs only).
	return extractFieldsRegex(content, v)
}

// StripFences removes markdown code fences from LLM output.
func StripFences(s string) string {
	for {
		trimmed := strings.TrimLeft(s, " \t")
		if !strings.HasPrefix(trimmed, "```") {
			break
		}
		nl := strings.IndexByte(trimmed, '\n')
		if nl < 0 {
			s = strings.TrimPrefix(trimmed, "```json")
			s = strings.TrimPrefix(s, "```")
			s = strings.TrimSpace(s)
			break
		}
		s = trimmed[nl+1:]
	}

	rev := strings.TrimRight(s, " \t\r\n")
	if strings.HasSuffix(rev, "```") {
		lastFence := strings.LastIndex(rev, "```")
		before := strings.TrimRight(rev[:lastFence], " \t\r\n")
		s = before
	}

	return strings.TrimSpace(s)
}

// ExtractJSONBlock finds the first complete JSON object {...} in s using brace
// counting, correctly skipping strings and escape sequences.
func ExtractJSONBlock(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// FixJSON attempts to repair common LLM JSON output mistakes:
// smart quotes → straight quotes, trailing commas, unescaped quotes in values.
func FixJSON(s string) string {
	s = strings.ReplaceAll(s, "\u201c", `"`)
	s = strings.ReplaceAll(s, "\u201d", `"`)
	s = strings.ReplaceAll(s, "\u2018", "'")
	s = strings.ReplaceAll(s, "\u2019", "'")

	// Remove trailing commas before } or ]
	for strings.Contains(s, ",}") {
		s = strings.ReplaceAll(s, ",}", "}")
	}
	for strings.Contains(s, ",]") {
		s = strings.ReplaceAll(s, ",]", "]")
	}

	s = fixUnescapedQuotesInValues(s)
	return s
}

// fixUnescapedQuotesInValues attempts to escape double-quotes that appear
// inside JSON string values. Scans for "key": " and then finds the matching
// closing quote; any " inside the value gets escaped.
func fixUnescapedQuotesInValues(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	i := 0
	for i < len(s) {
		colon := strings.Index(s[i:], `": "`)
		if colon < 0 {
			result.WriteString(s[i:])
			break
		}
		colon += i
		result.WriteString(s[i : colon+4])
		i = colon + 4

		for i < len(s) {
			if s[i] == '\\' && i+1 < len(s) {
				result.WriteByte(s[i])
				result.WriteByte(s[i+1])
				i += 2
				continue
			}
			if s[i] == '"' {
				after := strings.TrimLeft(s[i+1:], " \t\r\n")
				if len(after) == 0 || after[0] == ',' || after[0] == '}' || after[0] == ']' || after[0] == '\n' {
					result.WriteByte('"')
					i++
					break
				}
				result.WriteString(`\"`)
				i++
				continue
			}
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// extractFieldsRegex uses regular expressions to extract individual JSON
// fields from malformed LLM output. Last resort for simple flat structs.
// Supports struct targets via map[string]interface{} fallback.
func extractFieldsRegex(content string, v interface{}) error {
	getString := func(key string) string {
		re := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"\s*:\s*"([^"]*)"`)
		m := re.FindStringSubmatch(content)
		if len(m) >= 2 {
			return m[1]
		}
		return ""
	}

	// Try to extract to a map first, then marshal back into the target.
	// This works for any struct with JSON-tagged fields.
	fields := make(map[string]interface{})
	found := false

	// Try common field patterns from JSON tags — but we don't know the tags
	// at this level. Instead, just try unmarshaling what we can extract.
	// For the regex fallback, we extract whatever string/bool/int fields we
	// can find and try to re-serialize them into the target.

	switch dest := v.(type) {
	default:
		// Generic approach: try to extract as many key-value pairs as possible
		// using a broad regex, then re-marshal as JSON and unmarshal into target.
		// This handles flat JSON objects with string, bool, and int values.
		re := regexp.MustCompile(`"([^"]+)"\s*:\s*("(?:[^"\\]|\\.)*"|true|false|\d+(?:\.\d+)?)`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			if len(m) < 3 {
				continue
			}
			key := m[1]
			val := m[2]
			found = true

			// Parse the value
			if val == "true" {
				fields[key] = true
			} else if val == "false" {
				fields[key] = false
			} else if strings.HasPrefix(val, `"`) {
				// String value — unquote
				var s string
				if err := json.Unmarshal([]byte(val), &s); err == nil {
					fields[key] = s
				}
			} else if strings.Contains(val, ".") {
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					fields[key] = f
				}
			} else {
				if n, err := strconv.Atoi(val); err == nil {
					fields[key] = n
				}
			}
		}

		if !found {
			return fmt.Errorf("regex extraction failed: no fields found in LLM output")
		}

		// Re-marshal and unmarshal into the target
		reJSON, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("regex extraction: failed to marshal extracted fields: %w", err)
		}
		if err := json.Unmarshal(reJSON, dest); err != nil {
			return fmt.Errorf("regex extraction: failed to unmarshal into target: %w", err)
		}
		return nil

	case *string:
		// If the target is just a string, try to extract any string value
		if s := getString("result"); s != "" {
			*dest = s
			return nil
		}
		return fmt.Errorf("regex extraction: no string value found")
	}
}