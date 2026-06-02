package resolver

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/mod/module"

	"github.com/yorha2B0826/gogetx/internal/packageinfo"
)

type VersionLister interface {
	ListVersions(ctx context.Context, modulePath string) ([]string, error)
}

type Resolver struct {
	lister VersionLister
}

func New(lister VersionLister) *Resolver {
	return &Resolver{lister: lister}
}

func (r *Resolver) Resolve(ctx context.Context, candidate packageinfo.PackageCandidate) (string, error) {
	if candidate.ModulePath != "" {
		if err := module.CheckPath(candidate.ModulePath); err != nil {
			return "", fmt.Errorf("invalid module path %q: %w", candidate.ModulePath, err)
		}
		return candidate.ModulePath, nil
	}

	for _, candidatePath := range candidateModulePaths(candidate.PackagePath) {
		if err := module.CheckPath(candidatePath); err != nil {
			continue
		}
		if r.lister == nil {
			continue
		}
		if _, err := r.lister.ListVersions(ctx, candidatePath); err == nil {
			return candidatePath, nil
		}
	}

	return "", fmt.Errorf("could not resolve module path for package %q", candidate.PackagePath)
}

func candidateModulePaths(packagePath string) []string {
	parts := strings.Split(strings.Trim(packagePath, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return nil
	}

	seen := map[string]bool{}
	var out []string
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		out = append(out, path)
	}

	if len(parts) >= 3 && parts[0] == "github.com" {
		if len(parts) >= 4 && isMajorVersionSuffix(parts[3]) {
			add(strings.Join(parts[:4], "/"))
		}
		add(strings.Join(parts[:3], "/"))
	}

	for i := len(parts); i >= 2; i-- {
		add(strings.Join(parts[:i], "/"))
	}

	return out
}

func isMajorVersionSuffix(part string) bool {
	if len(part) < 2 || part[0] != 'v' {
		return false
	}
	if part == "v0" || part == "v1" {
		return false
	}
	for _, r := range part[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
