// Package checkcd inspects the leading Stmt of a parsed Bash file to decide
// whether guard-bash should pass the command through unchanged (because the
// caller already supplied "cd <allowed-path> && ...") or prepend the hook
// cwd.
package checkcd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/htakahama/guard-bash/internal/parse"
)

// Verdict signals how main should rewrite the command.
type Verdict int

const (
	// NeedsPrepend: command does not begin with a static cd; caller should
	// prepend "cd $CWD && " to the original source.
	NeedsPrepend Verdict = iota
	// AlreadyOK: command begins with "cd <path>" where path resolves under
	// an allowed directory; caller passes the command through unchanged.
	AlreadyOK
)

// ErrOutsideAllowed is returned when the leading cd targets a path outside
// cwd / allowedDirs.
var ErrOutsideAllowed = errors.New("cd target is outside allowed directories")

// ErrDynamicTarget is returned when the leading cd has a variable or command
// substitution as its argument.
var ErrDynamicTarget = errors.New("cd target is dynamic or missing")

// Check inspects file.Stmts[0] and reports what main should do.
//
// The leftmost BinaryCmd operand is examined; nested subshells and other
// constructs are ignored (they still get their commands extracted by the
// extract package, which is the mechanism that enforces policy).
func Check(file *syntax.File, cwd string, allowedDirs []string) (Verdict, error) {
	if len(file.Stmts) == 0 {
		return NeedsPrepend, nil
	}

	cmd := parse.LeftmostCmd(file.Stmts[0].Cmd)
	call, ok := cmd.(*syntax.CallExpr)
	if !ok || len(call.Args) == 0 {
		return NeedsPrepend, nil
	}

	name, ok := parse.WordLiteral(call.Args[0])
	if !ok || name != "cd" {
		return NeedsPrepend, nil
	}

	if len(call.Args) < 2 {
		return NeedsPrepend, ErrDynamicTarget
	}
	target, ok := parse.WordLiteral(call.Args[1])
	if !ok || target == "" {
		return NeedsPrepend, ErrDynamicTarget
	}

	abs := resolve(target, cwd)
	dirs := append([]string{cwd}, allowedDirs...)
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if isUnder(abs, resolve(d, cwd)) {
			return AlreadyOK, nil
		}
	}
	return NeedsPrepend, fmt.Errorf("%w: %s (allowed: %v)", ErrOutsideAllowed, target, dirs)
}

func resolve(path, cwd string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return filepath.Clean(path)
}

func isUnder(target, allowed string) bool {
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
