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

	// Symlink pointing into cwd.
	link := filepath.Join(t.TempDir(), "link-to-sub")
	if err := os.Symlink(sub, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	outside2 := t.TempDir()

	cases := []struct {
		name      string
		src       string
		want      checkcd.Verdict
		errIs     error
		extraDirs []string
	}{
		{"no cd", "git status", checkcd.NeedsPrepend, nil, nil},
		{"cd cwd", "cd " + cwd + " && git status", checkcd.AlreadyOK, nil, nil},
		{"cd under cwd", "cd " + sub + " && git status", checkcd.AlreadyOK, nil, nil},
		{"cd outside", "cd " + outside + " && ls", checkcd.NeedsPrepend, checkcd.ErrOutsideAllowed, nil},
		{"cd outside but allowed", "cd " + outside + " && ls", checkcd.AlreadyOK, nil, []string{outside}},
		{"cd dynamic", `cd "$DIR" && ls`, checkcd.NeedsPrepend, checkcd.ErrDynamicTarget, nil},
		{"cd missing arg", "cd && ls", checkcd.NeedsPrepend, checkcd.ErrDynamicTarget, nil},
		{"cd inside subshell only", "(cd /tmp && ls); git status", checkcd.NeedsPrepend, nil, nil},
		// additional edge cases
		{"cd relative under cwd", "cd sub && git status", checkcd.AlreadyOK, nil, nil},
		{"cd symlink into cwd", "cd " + link + " && ls", checkcd.AlreadyOK, nil, nil},
		{"multiple allowed dirs", "cd " + outside2 + " && ls", checkcd.AlreadyOK, nil, []string{outside, outside2}},
		{"empty stmts", ":", checkcd.NeedsPrepend, nil, nil},
		{"cd dot", "cd . && git status", checkcd.AlreadyOK, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parse.Parse(tc.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, err := checkcd.Check(file, cwd, tc.extraDirs)
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
