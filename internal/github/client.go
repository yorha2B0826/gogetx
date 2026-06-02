package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/mod/modfile"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

const defaultAPIBaseURL = "https://api.github.com"

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
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

func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

func NewClient(opts ...Option) *Client {
	client := &Client{
		baseURL: defaultAPIBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		token: os.Getenv("GITHUB_TOKEN"),
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (c *Client) Search(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("search keyword is required")
	}
	opts = packageinfo.NormalizeSearchOptions(opts)

	endpoint, err := url.Parse(c.baseURL + "/search/repositories")
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("q", keyword+" language:go")
	query.Set("per_page", fmt.Sprintf("%d", opts.Limit))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("github search failed: %s", resp.Status)
	}

	var payload struct {
		Items []struct {
			FullName      string `json:"full_name"`
			Description   string `json:"description"`
			DefaultBranch string `json:"default_branch"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode github search response: %w", err)
	}

	results := make([]packageinfo.PackageCandidate, 0, len(payload.Items))
	for _, item := range payload.Items {
		modulePath, err := c.fetchModulePath(ctx, item.FullName, item.DefaultBranch)
		if err != nil {
			continue
		}
		results = append(results, packageinfo.PackageCandidate{
			PackagePath: modulePath,
			ModulePath:  modulePath,
			Synopsis:    item.Description,
			Source:      packageinfo.SourceGitHub,
		})
	}
	return results, nil
}

func (c *Client) fetchModulePath(ctx context.Context, fullName string, ref string) (string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid github repository name %q", fullName)
	}
	if ref == "" {
		ref = "HEAD"
	}
	endpoint, err := url.JoinPath(c.baseURL, "repos", parts[0], parts[1], "contents", "go.mod")
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("ref", ref)
	parsed.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("github go.mod fetch failed: %s", resp.Status)
	}

	var payload struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Encoding != "base64" {
		return "", fmt.Errorf("unsupported github content encoding %q", payload.Encoding)
	}
	content, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(payload.Content, "\n", ""))
	if err != nil {
		return "", err
	}
	modulePath := modfile.ModulePath(content)
	if modulePath == "" {
		return "", fmt.Errorf("go.mod has no module directive")
	}
	return modulePath, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gogetx")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}
