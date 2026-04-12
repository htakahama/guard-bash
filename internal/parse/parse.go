// Package parse wraps mvdan.cc/sh/v3/syntax with helpers tailored to
// guard-bash's needs (parsing bash source and reducing Word nodes to their
// static literal value).
package parse

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Parse parses src as a Bash script and returns the syntax tree.
func Parse(src string) (*syntax.File, error) {
	p := syntax.NewParser(syntax.Variant(syntax.LangBash))
	file, err := p.Parse(strings.NewReader(src), "")
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return file, nil
}

// WordLiteral reduces w to its static string value. Returns ok=false when w
// contains ParamExp, CmdSubst, ArithmExp, ProcSubst, or any other dynamic
// part that we cannot resolve without executing the shell.
func WordLiteral(w *syntax.Word) (string, bool) {
	var b strings.Builder
	for _, p := range w.Parts {
		switch p := p.(type) {
		case *syntax.Lit:
			b.WriteString(p.Value)
		case *syntax.SglQuoted:
			b.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, sub := range p.Parts {
				lit, ok := sub.(*syntax.Lit)
				if !ok {
					return "", false
				}
				b.WriteString(lit.Value)
			}
		default:
			return "", false
		}
	}
	return b.String(), true
}

// LeftmostCmd walks down the left operand of chained BinaryCmds and returns
// the innermost leaf command.
func LeftmostCmd(c syntax.Command) syntax.Command {
	for {
		bin, ok := c.(*syntax.BinaryCmd)
		if !ok {
			return c
		}
		c = bin.X.Cmd
	}
}

// EOF
