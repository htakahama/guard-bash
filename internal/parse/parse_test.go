package parse_test

import (
	"testing"

	"github.com/htakahama/guard-bash/internal/parse"
	"mvdan.cc/sh/v3/syntax"
)

func TestWordLiteral(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
		ok   bool
	}{
		{"bare literal", "git", "git", true},
		{"single quoted", "'hello world'", "hello world", true},
		{"double quoted literal", `"hello"`, "hello", true},
		{"double quoted with expansion", `"$HOME/bin"`, "", false},
		{"param expansion", "$cmd", "", false},
		{"command substitution", "$(whoami)", "", false},
		{"mixed lit and single", "pre'suf'", "presuf", true},
		{"empty word", "", "", true},
		{"process substitution needs parse as arg", "<(cat f)", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse as argument of echo so we get a Word node.
			file, err := parse.Parse("echo " + tc.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			call := firstCall(t, file)
			if len(call.Args) < 2 {
				// empty word case: only "echo" itself
				if tc.src == "" {
					return
				}
				t.Fatalf("expected at least 2 args, got %d", len(call.Args))
			}
			got, ok := parse.WordLiteral(call.Args[1])
			if ok != tc.ok {
				t.Errorf("ok = %v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLeftmostCmd(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string // first word of the leftmost command
	}{
		{"single command", "git status", "git"},
		{"binary and", "git add && git commit", "git"},
		{"binary or", "false || echo fallback", "false"},
		{"triple chain", "a && b && c", "a"},
		{"pipe left", "git log | head", "git"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parse.Parse(tc.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(file.Stmts) == 0 {
				t.Fatal("no statements")
			}
			cmd := parse.LeftmostCmd(file.Stmts[0].Cmd)
			call, ok := cmd.(*syntax.CallExpr)
			if !ok {
				t.Fatalf("leftmost is not CallExpr: %T", cmd)
			}
			got, _ := parse.WordLiteral(call.Args[0])
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	_, err := parse.Parse("echo '")
	if err == nil {
		t.Fatal("expected parse error for unterminated quote")
	}
}

func firstCall(t *testing.T, file *syntax.File) *syntax.CallExpr {
	t.Helper()
	var found *syntax.CallExpr
	syntax.Walk(file, func(n syntax.Node) bool {
		if call, ok := n.(*syntax.CallExpr); ok && found == nil {
			found = call
		}
		return found == nil
	})
	if found == nil {
		t.Fatal("no CallExpr found")
	}
	return found
}

// EOF
