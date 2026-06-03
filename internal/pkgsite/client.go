package pkgsite

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

const defaultBaseURL = "https://pkg.go.dev"

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
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (c *Client) Search(ctx context.Context, keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, error) {
	page, err := c.SearchPage(ctx, keyword, opts)
	if err != nil {
		return nil, err
	}
	return page.Results, nil
}

func (c *Client) SearchPage(ctx context.Context, keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return packageinfo.SearchPage{}, fmt.Errorf("search keyword is required")
	}
	opts = packageinfo.NormalizeSearchOptions(opts)

	endpoint, err := url.Parse(c.baseURL + "/v1beta/search")
	if err != nil {
		return packageinfo.SearchPage{}, err
	}
	query := endpoint.Query()
	query.Set("q", keyword)
	query.Set("limit", fmt.Sprintf("%d", opts.Limit))
	if opts.PageToken != "" {
		query.Set("token", opts.PageToken)
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return packageinfo.SearchPage{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gogetx")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return packageinfo.SearchPage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return packageinfo.SearchPage{}, fmt.Errorf("pkg.go.dev search failed: %s", resp.Status)
	}

	var payload struct {
		Items []struct {
			PackagePath string `json:"packagePath"`
			ModulePath  string `json:"modulePath"`
			Version     string `json:"version"`
			Synopsis    string `json:"synopsis"`
		} `json:"items"`
		NextPageToken string `json:"nextPageToken"`
		Total         int    `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return packageinfo.SearchPage{}, fmt.Errorf("decode pkg.go.dev search response: %w", err)
	}

	results := make([]packageinfo.PackageCandidate, 0, len(payload.Items))
	for _, item := range payload.Items {
		results = append(results, packageinfo.PackageCandidate{
			PackagePath: item.PackagePath,
			ModulePath:  item.ModulePath,
			Version:     item.Version,
			Synopsis:    item.Synopsis,
			Source:      packageinfo.SourcePkgsite,
		})
	}
	return packageinfo.SearchPage{
		Results:       results,
		NextPageToken: payload.NextPageToken,
		Total:         payload.Total,
	}, nil
}
