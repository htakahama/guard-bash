package pathutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/htakahama/guard-bash/internal/pathutil"
)

func TestResolve(t *testing.T) {
	cwd := t.TempDir()
	sub := filepath.Join(cwd, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cases := []struct {
		name string
		path string
		want string
	}{
		{"absolute stays", "/etc", "/etc"},
		{"relative resolves", "sub", sub},
		{"dot resolves", ".", cwd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pathutil.Resolve(tc.path, cwd)
			if got != tc.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", tc.path, cwd, got, tc.want)
			}
		})
	}
}

func TestIsUnder(t *testing.T) {
	cases := []struct {
		name    string
		target  string
		allowed string
		want    bool
	}{
		{"same dir", "/a/b", "/a/b", true},
		{"nested", "/a/b/c", "/a/b", true},
		{"outside", "/a/c", "/a/b", false},
		{"parent", "/a", "/a/b", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pathutil.IsUnder(tc.target, tc.allowed)
			if got != tc.want {
				t.Errorf("IsUnder(%q, %q) = %v, want %v", tc.target, tc.allowed, got, tc.want)
			}
		})
	}
}

// EOF
