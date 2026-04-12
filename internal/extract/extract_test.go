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
		// edge cases
		{"empty script", ":", []string{":"}},
		{"triple pipe", "git log | grep foo | head", []string{"git", "grep", "head"}},
		{"heredoc", "cat <<EOF\nhello\nEOF", []string{"cat"}},
		{"env multiple assignments", "env A=1 B=2 C=3 git status", []string{"env", "git"}},
		{"env dynamic inner", "env FOO=bar $cmd", []string{"env", extract.Dynamic}},
		// Known limitation: -n takes an argument but wrapper parsing treats
		// any non-flag token as the inner command, so "10" is extracted.
		{"nice -n arg limitation", "nice -n 10 git log", []string{"nice", "10"}},
		{"pure assignment no cmd", "FOO=bar", nil},
		{"dblquote dynamic arg", `echo "$HOME"`, []string{"echo"}},
		{"while loop", "while git fetch; do sleep 1; done", []string{"git", "sleep"}},
		{"or chain", "false || git status", []string{"false", "git"}},
		{"nested subshell pipe", "(git log | head) && echo done", []string{"git", "head", "echo"}},
		{"semicolon separator", "git add .; git commit -m x", []string{"git", "git"}},
		{"env no inner cmd", "env FOO=bar", []string{"env"}},
		{"command wrapper no args", "command", []string{"command"}},
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
