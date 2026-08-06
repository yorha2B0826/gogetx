package goproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
		baseURL: proxyBaseURLFromEnv(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// proxyBaseURLFromEnv returns the first http(s) entry of the user's GOPROXY
// setting so that non-default proxies (e.g. goproxy.cn) are honored. Falls
// back to proxy.golang.org when GOPROXY is unset, off, direct, or file-based.
func proxyBaseURLFromEnv() string {
	for _, entry := range strings.Split(os.Getenv("GOPROXY"), ",") {
		entry = strings.TrimSpace(entry)
		if strings.HasPrefix(entry, "http://") || strings.HasPrefix(entry, "https://") {
			return strings.TrimRight(entry, "/")
		}
	}
	return defaultProxyBaseURL
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

func (c *Client) ListVersions(ctx context.Context, modulePath string) ([]string, error) {
	if err := module.CheckPath(modulePath); err != nil {
		return nil, fmt.Errorf("invalid module path %q: %w", modulePath, err)
	}
	escaped, err := module.EscapePath(modulePath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+escaped+"/@v/list", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", "gogetx")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("proxy version list lookup failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, line := range strings.Split(string(body), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			versions = append(versions, line)
		}
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for module %q", modulePath)
	}
	return versions, nil
}
