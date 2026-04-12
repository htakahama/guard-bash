package argcheck

import (
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/htakahama/guard-bash/internal/parse"
)

// callArgs returns all statically-resolvable argument strings for a CallExpr,
// including Args[0] (the command name).  Returns nil for dynamic args.
func callArgs(call *syntax.CallExpr) []string {
	var out []string
	for _, w := range call.Args {
		s, ok := parse.WordLiteral(w)
		if !ok {
			out = append(out, "")
			continue
		}
		out = append(out, s)
	}
	return out
}

// hasShortFlag reports whether args contain a short flag matching r.
// Handles merged flags like -rf (contains both r and f).
// Stops scanning at "--".
func hasShortFlag(args []string, r rune) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if !strings.HasPrefix(a, "-") || strings.HasPrefix(a, "--") {
			continue
		}
		for _, c := range a[1:] {
			if c == r {
				return true
			}
		}
	}
	return false
}

// hasLongFlag reports whether args contain the given long flag (e.g. "--force").
// Also matches --force-with-lease style flags if exact match.
// Stops scanning at "--".
func hasLongFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if a == flag {
			return true
		}
	}
	return false
}

// broadPaths is the set of paths considered dangerously broad for recursive
// operations.
var broadPaths = map[string]bool{
	"/":     true,
	"~":     true,
	".":     true,
	"..":    true,
	"/home": true,
	"/etc":  true,
	"/usr":  true,
	"/var":  true,
	"/tmp":  true,
	"/root": true,
	"/opt":  true,
	"/bin":  true,
	"/sbin": true,
	"/lib":  true,
	"/dev":  true,
	"/mnt":  true,
	"/srv":  true,
}

// isBroadPath reports whether path is dangerously broad for recursive
// operations.
func isBroadPath(path string) bool {
	cleaned := filepath.Clean(path)
	return broadPaths[cleaned]
}

// nonFlagArgs returns positional arguments (not flags, not "--").
func nonFlagArgs(args []string) []string {
	var out []string
	pastFlags := false
	for _, a := range args {
		if a == "--" {
			pastFlags = true
			continue
		}
		if !pastFlags && strings.HasPrefix(a, "-") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// shells is the set of command names considered shell interpreters.
var shells = map[string]bool{
	"sh":   true,
	"bash": true,
	"zsh":  true,
	"dash": true,
	"ksh":  true,
}

// EOF
