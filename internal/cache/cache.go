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
	Keyword   string                         `json:"keyword"`
	Results   []packageinfo.PackageCandidate `json:"results"`
	CreatedAt time.Time                      `json:"createdAt"`
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
	data, err := s.read()
	if err != nil {
		return nil, false, err
	}
	entry, ok := data.Entries[cacheKey(keyword, opts)]
	if !ok {
		return nil, false, nil
	}
	if entry.CreatedAt.Add(s.ttl).Before(s.now()) {
		return nil, false, nil
	}
	return entry.Results, true, nil
}

func (s *Store) Set(keyword string, opts packageinfo.SearchOptions, results []packageinfo.PackageCandidate) error {
	data, err := s.read()
	if err != nil {
		return err
	}
	data.Entries[cacheKey(keyword, opts)] = SearchCacheEntry{
		Keyword:   keyword,
		Results:   results,
		CreatedAt: s.now(),
	}
	return s.write(data)
}

func (s *Store) Path() string {
	return s.path
}

type cacheFile struct {
	Entries map[string]SearchCacheEntry `json:"entries"`
}

func (s *Store) read() (cacheFile, error) {
	data := cacheFile{Entries: map[string]SearchCacheEntry{}}
	content, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return data, nil
	}
	if err != nil {
		return data, err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return data, nil
	}
	if err := json.Unmarshal(content, &data); err != nil {
		return data, fmt.Errorf("read search cache: %w", err)
	}
	if data.Entries == nil {
		data.Entries = map[string]SearchCacheEntry{}
	}
	return data, nil
}

func (s *Store) write(data cacheFile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(content, '\n'), 0o644)
}

func cacheKey(keyword string, opts packageinfo.SearchOptions) string {
	opts = packageinfo.NormalizeSearchOptions(opts)
	return strings.ToLower(fmt.Sprintf("%s|%s|%d", opts.Source, strings.TrimSpace(keyword), opts.Limit))
}
