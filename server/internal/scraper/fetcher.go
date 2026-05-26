package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// FetchRaw performs a raw HTTP GET and returns the response body as a string,
// limited to 512KB. Use this for JSON APIs and other machine-readable endpoints.
func (c *Client) FetchRaw(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("fetch raw: %w", err)
	}
	req.Header.Set("User-Agent", "LearnCode/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.direct.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch raw: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch raw: status %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, 512*1024)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("fetch raw: %w", err)
	}
	return string(body), nil
}

// FetchPageText fetches a web page and extracts its visible text content.
// maxChars limits the output; 0 means no limit (up to the 512KB read limit).
// Strips scripts, styles, and navigation elements.
func (c *Client) FetchPageText(ctx context.Context, url string, maxChars int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("fetch page: %w", err)
	}
	req.Header.Set("User-Agent", "LearnCode/1.0")

	resp, err := c.direct.Do(req)
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
	if maxChars > 0 && len(result) > maxChars {
		result = result[:maxChars]
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
