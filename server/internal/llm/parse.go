package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func ParseLLMJSON(content string, v interface{}) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("empty llm response")
	}

	if err := json.Unmarshal([]byte(content), v); err == nil {
		return nil
	}

	if cleaned := StripFences(content); cleaned != content {
		if err := json.Unmarshal([]byte(cleaned), v); err == nil {
			return nil
		}
	}

	extracted := ExtractJSONBlock(content)
	if extracted != "" {
		if err := json.Unmarshal([]byte(extracted), v); err == nil {
			return nil
		}

		if fixed := FixJSON(extracted); fixed != extracted {
			if err := json.Unmarshal([]byte(fixed), v); err == nil {
				return nil
			}
		}
	}

	return extractFieldsRegex(content, v)
}

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

func FixJSON(s string) string {
	s = strings.ReplaceAll(s, "\u201c", `"`)
	s = strings.ReplaceAll(s, "\u201d", `"`)
	s = strings.ReplaceAll(s, "\u2018", "'")
	s = strings.ReplaceAll(s, "\u2019", "'")

	for strings.Contains(s, ",}") {
		s = strings.ReplaceAll(s, ",}", "}")
	}
	for strings.Contains(s, ",]") {
		s = strings.ReplaceAll(s, ",]", "]")
	}

	s = fixUnescapedQuotesInValues(s)
	return s
}

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

func extractFieldsRegex(content string, v interface{}) error {
	getString := func(key string) string {
		re := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"\s*:\s*"([^"]*)"`)
		m := re.FindStringSubmatch(content)
		if len(m) >= 2 {
			return m[1]
		}
		return ""
	}

	fields := make(map[string]interface{})
	found := false

	switch dest := v.(type) {
	default:
		re := regexp.MustCompile(`"([^"]+)"\s*:\s*("(?:[^"\\]|\\.)*"|true|false|\d+(?:\.\d+)?)`)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			if len(m) < 3 {
				continue
			}
			key := m[1]
			val := m[2]
			found = true

			if val == "true" {
				fields[key] = true
			} else if val == "false" {
				fields[key] = false
			} else if strings.HasPrefix(val, `"`) {
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
			preview := content
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			return fmt.Errorf("regex extraction failed: no fields found in LLM output (raw: %s)", preview)
		}

		reJSON, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("regex extraction: failed to marshal extracted fields: %w", err)
		}
		if err := json.Unmarshal(reJSON, dest); err != nil {
			return fmt.Errorf("regex extraction: failed to unmarshal into target: %w", err)
		}
		return nil

	case *string:
		if s := getString("result"); s != "" {
			*dest = s
			return nil
		}
		return fmt.Errorf("regex extraction: no string value found")
	}
}
