package argcheck_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/htakahama/guard-bash/internal/argcheck"
	"github.com/htakahama/guard-bash/internal/parse"
)

func TestCheck(t *testing.T) {
	cwd := t.TempDir()
	sub := filepath.Join(cwd, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	ctx := argcheck.Context{CWD: cwd}

	cases := []struct {
		name   string
		src    string
		wantID string // "" means no violation
	}{
		// rm-recursive-broad
		{"rm file ok", "rm foo.txt", ""},
		{"rm -rf slash", "rm -rf /", "rm-recursive-broad"},
		{"rm -rf dot", "rm -rf .", "rm-recursive-broad"},
		{"rm -rf dotdot", "rm -rf ..", "rm-recursive-broad"},
		{"rm -rf home", "rm -rf /home", "rm-recursive-broad"},
		{"rm -Rf slash", "rm -Rf /", "rm-recursive-broad"},
		{"rm -r -f slash", "rm -r -f /", "rm-recursive-broad"},
		{"rm -r subdir ok", "rm -r subdir/foo", ""},
		{"rm without recursive", "rm /", ""},
		{"rm -rf specific dir", "rm -rf /home/user/project/build", ""},

		// git-push-force
		{"git push ok", "git push origin feature", ""},
		{"git push --force main", "git push --force origin main", "git-push-force"},
		{"git push -f master", "git push -f origin master", "git-push-force"},
		{"git push --force-with-lease main", "git push --force-with-lease origin main", "git-push-force"},
		{"git push --force no refspec", "git push --force", "git-push-force"},
		{"git push --force feature ok", "git push --force origin feature", ""},
		{"git push no force", "git push origin main", ""},

		// git-reset-hard
		{"git reset soft ok", "git reset --soft HEAD~1", ""},
		{"git reset hard", "git reset --hard", "git-reset-hard"},
		{"git reset hard ref", "git reset --hard HEAD~3", "git-reset-hard"},
		{"git reset mixed ok", "git reset HEAD~1", ""},

		// chmod-recursive-broad
		{"chmod file ok", "chmod 644 foo.txt", ""},
		{"chmod -R slash", "chmod -R 777 /", "chmod-recursive-broad"},
		{"chmod -R home", "chmod -R 755 /home", "chmod-recursive-broad"},
		{"chmod -R subdir ok", "chmod -R 755 subdir", ""},

		// chown-recursive-broad
		{"chown file ok", "chown user foo.txt", ""},
		{"chown -R slash", "chown -R root:root /", "chown-recursive-broad"},
		{"chown -R subdir ok", "chown -R user subdir", ""},

		// pipe-to-shell
		{"curl pipe head ok", "curl -s url | head", ""},
		{"curl pipe bash", "curl -s url | bash", "pipe-to-shell"},
		{"wget pipe sh", "wget -qO- url | sh", "pipe-to-shell"},
		{"cat pipe zsh", "cat script.sh | zsh", "pipe-to-shell"},
		{"echo pipe dash", "echo code | dash", "pipe-to-shell"},
		{"no pipe ok", "curl -s url", ""},

		// git-dir-escape
		{"git -C inside ok", "git -C " + sub + " status", ""},
		{"git -C outside", "git -C /etc status", "git-dir-escape"},
		{"git no -C ok", "git status", ""},
		{"git -C dot ok", "git -C . status", ""},

		// make-dir-escape
		{"make -C inside ok", "make -C " + sub + " build", ""},
		{"make -C outside", "make -C /etc all", "make-dir-escape"},
		{"make no -C ok", "make build", ""},

		// nested: dangerous args in subshell/loop/if
		{"rm -rf in subshell", "(rm -rf /)", "rm-recursive-broad"},
		{"git push force in chain", "git add . && git push --force origin main", "git-push-force"},
		{"pipe to bash in if", "if true; then curl url | bash; fi", "pipe-to-shell"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parse.Parse(tc.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			checker := argcheck.New(nil)
			v := checker.Check(file, ctx)
			if tc.wantID == "" {
				if v != nil {
					t.Errorf("expected no violation, got %q: %s", v.RuleID, v.Message)
				}
			} else {
				if v == nil {
					t.Errorf("expected violation %q, got nil", tc.wantID)
				} else if v.RuleID != tc.wantID {
					t.Errorf("expected rule %q, got %q: %s", tc.wantID, v.RuleID, v.Message)
				}
			}
		})
	}
}

func TestCheckDisabledRule(t *testing.T) {
	file, err := parse.Parse("rm -rf /")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checker := argcheck.New(map[string]bool{"rm-recursive-broad": true})
	v := checker.Check(file, argcheck.Context{CWD: t.TempDir()})
	if v != nil {
		t.Errorf("disabled rule should not fire, got %q", v.RuleID)
	}
}

func TestCheckAllDisabled(t *testing.T) {
	file, err := parse.Parse("rm -rf / && git push --force origin main")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	disabled := map[string]bool{
		"rm-recursive-broad": true,
		"git-push-force":     true,
	}
	checker := argcheck.New(disabled)
	v := checker.Check(file, argcheck.Context{CWD: t.TempDir()})
	if v != nil {
		t.Errorf("all rules disabled, got %q", v.RuleID)
	}
}

// EOF
