// Package extract walks a Bash syntax tree and returns every command name
// that would be invoked at runtime. It understands compound constructs
// (for/while/if/case/pipeline/subshell/command substitution) because
// syntax.Walk visits all descendants; each CallExpr becomes one command name.
package extract

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/htakahama/guard-bash/internal/parse"
)

// Dynamic is the sentinel emitted for command names that cannot be resolved
// statically (variable expansion, command substitution, etc.).
const Dynamic = "__DYNAMIC__"

// wrapperCommands prefix another command whose name should also be extracted.
// "time" is not listed here because the bash parser models it as TimeClause
// with a child Stmt, so the inner CallExpr is visited naturally.
var wrapperCommands = map[string]struct{}{
	"env":     {},
	"command": {},
	"nice":    {},
	"nohup":   {},
}

// Commands walks the AST and returns an ordered slice of command names.
// Duplicates are preserved so the caller can see every invocation site.
func Commands(file *syntax.File) []string {
	var out []string
	syntax.Walk(file, func(n syntax.Node) bool {
		if call, ok := n.(*syntax.CallExpr); ok {
			out = append(out, fromCallExpr(call)...)
		}
		return true
	})
	return out
}

func fromCallExpr(call *syntax.CallExpr) []string {
	if len(call.Args) == 0 {
		return nil
	}
	first, ok := parse.WordLiteral(call.Args[0])
	if !ok {
		return []string{Dynamic}
	}
	if first == "" {
		return nil
	}
	first = basename(first)
	out := []string{first}

	if _, isWrapper := wrapperCommands[first]; !isWrapper {
		return out
	}

	for i := 1; i < len(call.Args); i++ {
		w, ok := parse.WordLiteral(call.Args[i])
		if !ok {
			out = append(out, Dynamic)
			return out
		}
		if first == "env" && isAssignToken(w) {
			continue
		}
		if strings.HasPrefix(w, "-") {
			continue
		}
		out = append(out, basename(w))
		return out
	}
	return out
}

func basename(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// isAssignToken reports whether s looks like KEY=VALUE.
func isAssignToken(s string) bool {
	idx := strings.Index(s, "=")
	if idx <= 0 {
		return false
	}
	name := s[:idx]
	for i, r := range name {
		if i == 0 && r >= '0' && r <= '9' {
			return false
		}
		alnum := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !alnum {
			return false
		}
	}
	return true
}

// EOF
