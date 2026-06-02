package packageinfo

const (
	SourcePkgsite = "pkgsite"
	SourceGitHub  = "github"
	SourceAll     = "all"
)

type PackageCandidate struct {
	PackagePath string `json:"packagePath" yaml:"packagePath"`
	ModulePath  string `json:"modulePath" yaml:"modulePath"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Synopsis    string `json:"synopsis,omitempty" yaml:"synopsis,omitempty"`
	Source      string `json:"source" yaml:"source"`
}

type SearchOptions struct {
	Limit   int
	Source  string
	NoCache bool
	Refresh bool
}

func NormalizeSearchOptions(opts SearchOptions) SearchOptions {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Source == "" {
		opts.Source = SourcePkgsite
	}
	return opts
}
