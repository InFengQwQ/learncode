package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// FetchPageText fetches a web page and extracts its visible text content.
// Returns up to maxChars of text, stripped of scripts, styles, and navigation.
func (c *Client) FetchPageText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("fetch page: %w", err)
	}
	req.Header.Set("User-Agent", "LearnCode/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch page: status %d", resp.StatusCode)
	}

	// Limit read to 512KB
	limited := io.LimitReader(resp.Body, 512*1024)
	doc, err := html.Parse(limited)
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}

	var text strings.Builder
	extractText(doc, &text)

	result := collapseSpace(text.String())
	if len(result) > 3000 {
		result = result[:3000]
	}
	return result, nil
}

// skipTags contains HTML tags whose content should not be extracted.
var skipTags = map[string]bool{
	"script": true, "style": true, "noscript": true,
	"nav": true, "footer": true, "header": true,
	"code": true, "pre": true, "svg": true,
}

func extractText(n *html.Node, out *strings.Builder) {
	if n.Type == html.TextNode {
		out.WriteString(n.Data)
		out.WriteByte(' ')
		return
	}
	if n.Type == html.ElementNode {
		if skipTags[n.Data] {
			return
		}
		// Add line breaks for block elements
		switch n.Data {
		case "br", "p", "div", "li", "h1", "h2", "h3", "h4", "h5", "h6", "tr":
			out.WriteByte('\n')
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, out)
	}
}

func collapseSpace(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == '\n' || r == '\t' || r == ' ' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}
