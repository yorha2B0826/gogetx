package goproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/module"
)

const defaultProxyBaseURL = "https://proxy.golang.org"

type VersionInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time,omitempty"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Option func(*Client)

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func NewClient(opts ...Option) *Client {
	client := &Client{
		baseURL: defaultProxyBaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (c *Client) Latest(ctx context.Context, modulePath string) (VersionInfo, error) {
	if err := module.CheckPath(modulePath); err != nil {
		return VersionInfo{}, fmt.Errorf("invalid module path %q: %w", modulePath, err)
	}
	escaped, err := module.EscapePath(modulePath)
	if err != nil {
		return VersionInfo{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+escaped+"/@latest", nil)
	if err != nil {
		return VersionInfo{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gogetx")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return VersionInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return VersionInfo{}, fmt.Errorf("proxy latest lookup failed: %s", resp.Status)
	}

	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return VersionInfo{}, fmt.Errorf("decode proxy latest response: %w", err)
	}
	return info, nil
}
