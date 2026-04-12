// Package pathutil provides shared path-resolution helpers used by checkcd
// and argcheck.
package pathutil

import (
	"path/filepath"
	"strings"
)

// Resolve makes path absolute (relative to cwd) and evaluates symlinks.
func Resolve(path, cwd string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return filepath.Clean(path)
}

// IsUnder reports whether target is equal to or nested inside allowed.
func IsUnder(target, allowed string) bool {
	rel, err := filepath.Rel(allowed, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

// EOF
