package doc

import (
	"net/url"
	"strings"
)

func URLFor(target string) string {
	target = strings.TrimSpace(target)
	if looksLikePackagePath(target) {
		u := url.URL{Scheme: "https", Host: "pkg.go.dev", Path: "/" + target}
		return u.String()
	}
	u := url.URL{Scheme: "https", Host: "pkg.go.dev", Path: "/search"}
	query := u.Query()
	query.Set("q", target)
	u.RawQuery = query.Encode()
	return u.String()
}

func looksLikePackagePath(target string) bool {
	return strings.Contains(target, "/") || strings.Contains(target, ".")
}
