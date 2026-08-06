package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type SearchCacheEntry struct {
	Keyword       string                         `json:"keyword"`
	Results       []packageinfo.PackageCandidate `json:"results"`
	NextPageToken string                         `json:"nextPageToken,omitempty"`
	Total         int                            `json:"total,omitempty"`
	CreatedAt     time.Time                      `json:"createdAt"`
}

type Store struct {
	path string
	ttl  time.Duration
	now  func() time.Time
}

func NewStore(path string, ttl time.Duration) *Store {
	return &Store{
		path: path,
		ttl:  ttl,
		now:  time.Now,
	}
}

func (s *Store) Get(keyword string, opts packageinfo.SearchOptions) ([]packageinfo.PackageCandidate, bool, error) {
	page, ok, err := s.GetPage(keyword, opts)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return page.Results, true, nil
}

func (s *Store) GetPage(keyword string, opts packageinfo.SearchOptions) (packageinfo.SearchPage, bool, error) {
	entry, ok := s.read().Entries[cacheKey(keyword, opts)]
	if !ok {
		return packageinfo.SearchPage{}, false, nil
	}
	if entry.CreatedAt.Add(s.ttl).Before(s.now()) {
		return packageinfo.SearchPage{}, false, nil
	}
	return packageinfo.SearchPage{
		Results:       entry.Results,
		NextPageToken: entry.NextPageToken,
		Total:         entry.Total,
	}, true, nil
}

func (s *Store) Set(keyword string, opts packageinfo.SearchOptions, results []packageinfo.PackageCandidate) error {
	return s.SetPage(keyword, opts, packageinfo.SearchPage{Results: results})
}

func (s *Store) SetPage(keyword string, opts packageinfo.SearchOptions, page packageinfo.SearchPage) error {
	data := s.read()
	pruneExpired(data.Entries, s.ttl, s.now())
	data.Entries[cacheKey(keyword, opts)] = SearchCacheEntry{
		Keyword:       keyword,
		Results:       page.Results,
		NextPageToken: page.NextPageToken,
		Total:         page.Total,
		CreatedAt:     s.now(),
	}
	return s.write(data)
}

func (s *Store) Path() string {
	return s.path
}

type cacheFile struct {
	Entries map[string]SearchCacheEntry `json:"entries"`
}

// read loads the cache file as best-effort: a missing, unreadable, or corrupt
// file is treated as an empty cache so a bad cache can never break a search.
// The next Set overwrites it, so a corrupt file self-heals.
func (s *Store) read() cacheFile {
	data := cacheFile{Entries: map[string]SearchCacheEntry{}}
	content, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return data
	}
	if err != nil {
		return data
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return data
	}
	if err := json.Unmarshal(content, &data); err != nil {
		return data
	}
	if data.Entries == nil {
		data.Entries = map[string]SearchCacheEntry{}
	}
	return data
}

// write persists the cache atomically by writing to a temp file in the same
// directory and renaming over the target, so a crash mid-write never leaves a
// truncated cache behind.
func (s *Store) write(data cacheFile) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')

	tmp, err := os.CreateTemp(dir, "search-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}

// pruneExpired drops entries whose TTL has passed, keeping the cache file from
// growing without bound.
func pruneExpired(entries map[string]SearchCacheEntry, ttl time.Duration, now time.Time) {
	for key, entry := range entries {
		if !entry.CreatedAt.Add(ttl).After(now) {
			delete(entries, key)
		}
	}
}

func cacheKey(keyword string, opts packageinfo.SearchOptions) string {
	opts = packageinfo.NormalizeSearchOptions(opts)
	return strings.ToLower(fmt.Sprintf("%s|%s|%d|%s", opts.Source, strings.TrimSpace(keyword), opts.Limit, opts.PageToken))
}
