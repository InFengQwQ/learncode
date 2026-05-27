package scraper

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

const userAgent = "LearnCode/1.0"

type Client struct {
	wiki    *http.Client // uses proxy (Accesser) for Wikipedia API
	direct  *http.Client // direct connection for general web
	baseURL string
}

// NewClient creates a scraper with Wikipedia proxy at localhost:7654 (Accesser).
func NewClient() *Client {
	return NewClientWithProxy("http://localhost:7654")
}

// NewClientWithProxy creates a scraper with two HTTP transports:
//   - wikiClient: routes through proxyURL to bypass DNS blocking for Wikipedia
//   - directClient: direct connection (no proxy) for general websites
func NewClientWithProxy(proxyURL string) *Client {
	proxyTransport := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}
	if u, err := url.Parse(proxyURL); err == nil && u.Host != "" {
		proxyTransport.Proxy = http.ProxyURL(u)
	}

	directTransport := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}

	return &Client{
		wiki:    &http.Client{Timeout: 120 * time.Second, Transport: proxyTransport},
		direct:  &http.Client{Timeout: 120 * time.Second, Transport: directTransport},
		baseURL: "https://en.wikipedia.org/w/api.php",
	}
}

// doWiki performs a Wikipedia API request and checks for errors.
// Uses the wiki client (proxy transport) to bypass DNS blocking.
func (c *Client) doWiki(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.wiki.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		return nil, &WikipediaError{Message: "Wikipedia API rate limited (429) — please wait and try again"}
	}
	return resp, nil
}

// WikipediaError is returned when the Wikipedia API returns an error.
type WikipediaError struct {
	Message string
}

func (e *WikipediaError) Error() string { return e.Message }
