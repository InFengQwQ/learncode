package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type WikiHit struct {
	Title   string
	Snippet string
	URL     string
}

func (c *Client) SearchWikipedia(ctx context.Context, query string) ([]WikiHit, error) {
	u := fmt.Sprintf("%s?action=opensearch&search=%s&limit=5&format=json",
		c.baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doWiki(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search status %d", resp.StatusCode)
	}

	var raw []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("search parse: %w", err)
	}
	if len(raw) < 4 {
		return nil, nil
	}

	var titles []string
	var snippets []string
	var urls []string
	json.Unmarshal(raw[1], &titles)
	json.Unmarshal(raw[2], &snippets)
	json.Unmarshal(raw[3], &urls)

	hits := make([]WikiHit, len(titles))
	for i := range titles {
		hits[i] = WikiHit{
			Title:   titles[i],
			Snippet: cleanSnippet(snippets[i]),
			URL:     urls[i],
		}
	}
	return hits, nil
}

func (c *Client) GetPageCategories(ctx context.Context, title string) ([]string, error) {
	u := fmt.Sprintf("%s?action=parse&page=%s&prop=categories&format=json",
		c.baseURL, url.QueryEscape(title))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doWiki(req)
	if err != nil {
		return nil, fmt.Errorf("categories request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Parse struct {
			Categories []struct {
				SortKey string `json:"sortkey"`
				Hidden  string `json:"hidden"`
				Star    string `json:"*"`
			} `json:"categories"`
		} `json:"parse"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("categories parse: %w", err)
	}

	cats := make([]string, len(result.Parse.Categories))
	for i, c := range result.Parse.Categories {
		cats[i] = strings.TrimPrefix(c.Star, "Category:")
	}
	return cats, nil
}

type InfoboxData struct {
	InfoboxType   string
	Website       string
	LatestVersion string
	Developer     string
	Typing        string
	FirstAppeared string
}

func (c *Client) GetInfobox(ctx context.Context, title string) (*InfoboxData, error) {
	u := fmt.Sprintf("%s?action=parse&page=%s&prop=text&section=0&format=json",
		c.baseURL, url.QueryEscape(title))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doWiki(req)
	if err != nil {
		return nil, fmt.Errorf("infobox request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Parse struct {
			Text struct {
				Star string `json:"*"`
			} `json:"text"`
		} `json:"parse"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("infobox parse: %w", err)
	}

	return parseInfoboxHTML(result.Parse.Text.Star), nil
}

func (c *Client) GetPageImage(ctx context.Context, title string) (string, error) {
	u := fmt.Sprintf("%s?action=query&titles=%s&prop=pageimages&format=json&pithumbsize=120&redirects=true",
		c.baseURL, url.QueryEscape(title))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.doWiki(req)
	if err != nil {
		return "", fmt.Errorf("pageimage request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Query struct {
			Pages map[string]struct {
				Thumbnail struct {
					Source string `json:"source"`
				} `json:"thumbnail"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("pageimage parse: %w", err)
	}

	for _, page := range result.Query.Pages {
		if page.Thumbnail.Source != "" {
			return page.Thumbnail.Source, nil
		}
	}
	return "", nil
}

func (c *Client) GetExternalLinks(ctx context.Context, title string) ([]string, error) {
	u := fmt.Sprintf("%s?action=parse&page=%s&prop=externallinks&format=json",
		c.baseURL, url.QueryEscape(title))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doWiki(req)
	if err != nil {
		return nil, fmt.Errorf("externallinks request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Parse struct {
			Externallinks []string `json:"externallinks"`
		} `json:"parse"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("externallinks parse: %w", err)
	}

	var links []string
	for _, link := range result.Parse.Externallinks {
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			links = append(links, link)
		}
	}
	return links, nil
}

func ScoreSignal(cats []string, info *InfoboxData) (score int, reject bool) {
	for _, c := range cats {
		lower := strings.ToLower(c)
		if strings.Contains(lower, "programming language") {
			score += 2
		}
		if strings.Contains(lower, "software framework") ||
			strings.Contains(lower, "javascript library") ||
			strings.Contains(lower, "web framework") {
			return 0, true
		}
	}
	if info != nil && strings.Contains(strings.ToLower(info.InfoboxType), "programming language") {
		score++
	}
	return score, false
}

func parseInfoboxHTML(html string) *InfoboxData {
	start := findInfoboxStart(html)
	if start < 0 {
		return nil
	}
	end := findTableEnd(html, start)
	if end < 0 || end > len(html) {
		end = len(html)
	}
	tableHTML := html[start:end]

	info := &InfoboxData{}
	info.InfoboxType = extractInfoboxType(tableHTML)

	rows := extractRows(tableHTML)
	for _, row := range rows {
		key := normalizeKey(row.key)
		val := row.val
		switch {
		case key == "website" || key == "homepage":
			if u := extractHrefFromRaw(row.rawVal); u != "" {
				info.Website = u
			}
		case strings.Contains(key, "stable release") || key == "latest release":
			info.LatestVersion = val
		case key == "developer" || key == "developers":
			info.Developer = val
		case strings.Contains(key, "typing") || key == "type system":
			info.Typing = val
		case strings.Contains(key, "first appeared") || key == "first appeared":
			info.FirstAppeared = val
		}
	}
	return info
}

func findInfoboxStart(html string) int {
	idx := strings.Index(html, `class="infobox"`)
	if idx < 0 {
		idx = strings.Index(html, `class="infobox`)
	}
	if idx < 0 {
		return -1
	}
	back := strings.LastIndex(html[:idx], "<table")
	return back
}

func findTableEnd(html string, start int) int {
	depth := 0
	for i := start; i < len(html)-6; i++ {
		if html[i] == '<' && strings.HasPrefix(html[i:], "</table") {
			if depth == 0 {
				return i
			}
			depth--
		}
		if html[i] == '<' && strings.HasPrefix(html[i:], "<table") {
			depth++
		}
	}
	return len(html)
}

func extractInfoboxType(html string) string {
	if idx := strings.Index(html, "infobox-title"); idx >= 0 {
		start := strings.Index(html[idx:], ">") + idx + 1
		end := strings.Index(html[start:], "<") + start
		if end > start {
			return strings.TrimSpace(html[start:end])
		}
	}
	if idx := strings.Index(html, "<caption"); idx >= 0 {
		start := strings.Index(html[idx:], ">") + idx + 1
		end := strings.Index(html[start:], "</caption>") + start
		if end > start {
			return strings.TrimSpace(stripTags(html[start:end]))
		}
	}
	return ""
}

type rowKV struct {
	key    string
	val    string
	rawVal string
}

func extractRows(html string) []rowKV {
	var rows []rowKV
	for {
		trStart := strings.Index(html, "<tr")
		if trStart < 0 {
			break
		}
		trEnd := strings.Index(html[trStart:], "</tr>") + trStart + 5
		if trEnd < trStart+5 {
			trEnd = len(html)
		}
		trHTML := html[trStart:trEnd]

		key := extractCell(trHTML, "th")
		val, raw := extractCellWithRaw(trHTML, "td")
		if key != "" {
			rows = append(rows, rowKV{key: key, val: val, rawVal: raw})
		}

		html = html[trEnd:]
	}
	return rows
}

func extractCell(html, tag string) string {
	v, _ := extractCellWithRaw(html, tag)
	return v
}

func extractCellWithRaw(html, tag string) (stripped string, raw string) {
	openTag := "<" + tag
	closeTag := "</" + tag + ">"
	start := strings.Index(html, openTag)
	if start < 0 {
		return "", ""
	}
	start = strings.Index(html[start:], ">") + start + 1
	end := strings.Index(html[start:], closeTag)
	if end < 0 {
		return "", ""
	}
	raw = html[start : start+end]
	return strings.TrimSpace(stripTags(raw)), raw
}

func extractHrefFromRaw(html string) string {
	html = removeStyleAndScript(html)

	idx := strings.Index(strings.ToLower(html), "<a ")
	if idx < 0 {
		return ""
	}
	endTag := strings.Index(html[idx:], ">")
	if endTag < 0 {
		return ""
	}
	anchorHTML := html[idx : idx+endTag]

	hrefIdx := strings.Index(anchorHTML, "href=")
	if hrefIdx < 0 {
		return ""
	}

	rest := anchorHTML[hrefIdx+5:]
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		end := strings.IndexAny(rest, " >")
		if end < 0 {
			return strings.TrimSpace(rest)
		}
		return strings.TrimSpace(rest[:end])
	}
	rest = rest[1:]
	end := strings.IndexByte(rest, quote)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func NormalizeTitle(title string) string {
	title = strings.TrimSpace(title)
	if idx := strings.Index(title, " ("); idx >= 0 {
		suffix := strings.ToLower(title[idx:])
		for _, pattern := range []string{
			" (programming language)",
			" (programming_language)",
			" (language)",
			" (software)",
		} {
			if suffix == pattern {
				return strings.TrimSpace(title[:idx])
			}
		}
	}
	return strings.TrimSpace(title)
}

func NormalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
		} else if r == ' ' || r == '_' || r == '-' {
			if !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return result
}

func normalizeKey(s string) string {
	s = strings.TrimSpace(stripTags(s))
	s = strings.ToLower(s)
	for {
		bracket := strings.Index(s, "[")
		if bracket < 0 {
			break
		}
		closeB := strings.Index(s[bracket:], "]")
		if closeB < 0 {
			break
		}
		s = s[:bracket] + s[bracket+closeB+1:]
	}
	return strings.TrimSpace(s)
}

func removeStyleAndScript(s string) string {
	for _, tag := range []string{"style", "script"} {
		for {
			start := findOpenTag(s, tag)
			if start < 0 {
				break
			}
			end := findCloseTag(s, tag, start)
			if end < 0 {
				break
			}
			s = s[:start] + s[end:]
		}
	}
	return s
}

func findOpenTag(s, tag string) int {
	lower := strings.ToLower(s)
	for {
		idx := strings.Index(lower, "<"+tag)
		if idx < 0 {
			return -1
		}
		after := idx + 1 + len(tag)
		if after < len(lower) && (lower[after] == '>' || lower[after] == ' ' || lower[after] == '\t' || lower[after] == '\n') {
			return idx
		}
		lower = lower[after:]
		s = s[after:]
	}
}

func findCloseTag(s, tag string, start int) int {
	closeTag := "</" + tag + ">"
	end := strings.Index(strings.ToLower(s[start:]), closeTag)
	if end < 0 {
		return -1
	}
	return start + end + len(closeTag)
}

func stripTags(s string) string {
	s = removeStyleAndScript(s)
	var b strings.Builder
	inTag := false
	for i := 0; i < len(s); i++ {
		if s[i] == '<' {
			inTag = true
			continue
		}
		if s[i] == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func cleanSnippet(s string) string {
	return strings.TrimSpace(stripTags(s))
}
