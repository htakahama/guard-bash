package extract_test

import (
	"reflect"
	"testing"

	"github.com/htakahama/guard-bash/internal/extract"
	"github.com/htakahama/guard-bash/internal/parse"
)

func TestCommands(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"simple", "git status", []string{"git"}},
		{"pipe", "git log | head", []string{"git", "head"}},
		{"chain", "git add . && git commit -m foo", []string{"git", "git"}},
		{"for + cmdsubst", `for f in $(git ls-files); do cat "$f"; done`, []string{"git", "cat"}},
		{"if + test", "if [ -f x ]; then git status; fi", []string{"[", "git"}},
		{"env FOO=bar cmd", "env FOO=bar git status", []string{"env", "git"}},
		{"nohup wrapper", "nohup git log", []string{"nohup", "git"}},
		{"time clause", "time git log", []string{"git"}},
		{"assignment only", "VAR=x git log", []string{"git"}},
		{"dynamic first", "$cmd arg", []string{extract.Dynamic}},
		{"eval", "eval 'git status'", []string{"eval"}},
		{"subshell", "(cd /tmp && git init)", []string{"cd", "git"}},
		{"cmdsubst nested denied", `x=$(sudo rm -rf /); echo $x`, []string{"sudo", "echo"}},
		{"basename normalisation", "/usr/bin/git status", []string{"git"}},
		{"command -v", "command -v git", []string{"command", "git"}},
		{"case clause", "case $x in a) git log;; b) cat f;; esac", []string{"git", "cat"}},
		{"until loop", "until grep foo bar; do sleep 1; done", []string{"grep", "sleep"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parse.Parse(tc.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := extract.Commands(file)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("\nsrc:  %s\nwant: %v\ngot:  %v", tc.src, tc.want, got)
			}
		})
	}
}

// EOF
