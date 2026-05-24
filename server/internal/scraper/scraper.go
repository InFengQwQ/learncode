package scraper

import (
	"net/http"
	"time"
)

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: "https://en.wikipedia.org/w/api.php",
	}
}
