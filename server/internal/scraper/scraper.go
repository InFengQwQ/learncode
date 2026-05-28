package scraper

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

const userAgent = "LearnCode/1.0"

type Client struct {
	wiki    *http.Client
	direct  *http.Client
	baseURL string
}

func NewClient() *Client {
	return NewClientWithProxy("http://localhost:7654")
}

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

func (c *Client) doWiki(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.wiki.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wikipedia proxy unavailable (is Accesser running?): %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		return nil, &WikipediaError{Message: "Wikipedia API rate limited (429) — please wait and try again"}
	}
	return resp, nil
}

type WikipediaError struct {
	Message string
}

func (e *WikipediaError) Error() string { return e.Message }
