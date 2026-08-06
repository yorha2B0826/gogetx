package packageinfo

const (
	SourcePkgsite = "pkgsite"
	SourceGitHub  = "github"
	SourceAll     = "all"

	DefaultSearchLimit = 30
)

type PackageCandidate struct {
	PackagePath string `json:"packagePath" yaml:"packagePath"`
	ModulePath  string `json:"modulePath" yaml:"modulePath"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Synopsis    string `json:"synopsis,omitempty" yaml:"synopsis,omitempty"`
	Source      string `json:"source" yaml:"source"`
}

// DedupeKey identifies a candidate across sources and pages.
func (c PackageCandidate) DedupeKey() string {
	return c.ModulePath + "|" + c.PackagePath
}

// DedupeCandidates removes repeated candidates while preserving order.
func DedupeCandidates(results []PackageCandidate) []PackageCandidate {
	seen := map[string]bool{}
	out := make([]PackageCandidate, 0, len(results))
	for _, result := range results {
		key := result.DedupeKey()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, result)
	}
	return out
}

type SearchOptions struct {
	Limit     int
	Source    string
	PageToken string
	NoCache   bool
	Refresh   bool
}

type SearchPage struct {
	Results       []PackageCandidate `json:"results"`
	NextPageToken string             `json:"nextPageToken,omitempty"`
	Total         int                `json:"total,omitempty"`
}

func NormalizeSearchOptions(opts SearchOptions) SearchOptions {
	if opts.Limit <= 0 {
		opts.Limit = DefaultSearchLimit
	}
	if opts.Source == "" {
		opts.Source = SourcePkgsite
	}
	return opts
}
