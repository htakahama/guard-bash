package checkcd_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/htakahama/guard-bash/internal/checkcd"
	"github.com/htakahama/guard-bash/internal/parse"
)

func TestCheck(t *testing.T) {
	cwd := t.TempDir()
	sub := filepath.Join(cwd, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	outside := t.TempDir() // sibling, not under cwd

	cases := []struct {
		name     string
		src      string
		want     checkcd.Verdict
		errIs    error
		extraDir string
	}{
		{"no cd", "git status", checkcd.NeedsPrepend, nil, ""},
		{"cd cwd", "cd " + cwd + " && git status", checkcd.AlreadyOK, nil, ""},
		{"cd under cwd", "cd " + sub + " && git status", checkcd.AlreadyOK, nil, ""},
		{"cd outside", "cd " + outside + " && ls", checkcd.NeedsPrepend, checkcd.ErrOutsideAllowed, ""},
		{"cd outside but allowed", "cd " + outside + " && ls", checkcd.AlreadyOK, nil, outside},
		{"cd dynamic", `cd "$DIR" && ls`, checkcd.NeedsPrepend, checkcd.ErrDynamicTarget, ""},
		{"cd missing arg", "cd && ls", checkcd.NeedsPrepend, checkcd.ErrDynamicTarget, ""},
		{"cd inside subshell only", "(cd /tmp && ls); git status", checkcd.NeedsPrepend, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parse.Parse(tc.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			var allowed []string
			if tc.extraDir != "" {
				allowed = []string{tc.extraDir}
			}
			got, err := checkcd.Check(file, cwd, allowed)
			if tc.errIs != nil {
				if !errors.Is(err, tc.errIs) {
					t.Errorf("expected error %v, got %v", tc.errIs, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("want verdict %v, got %v", tc.want, got)
			}
		})
	}
}

// EOF
