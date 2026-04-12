package argcheck

import (
	"fmt"

	"mvdan.cc/sh/v3/syntax"

	"github.com/htakahama/guard-bash/internal/parse"
	"github.com/htakahama/guard-bash/internal/pathutil"
)

func init() {
	register(Rule{ID: "make-dir-escape", Check: checkMakeDirEscape})
}

func checkMakeDirEscape(file *syntax.File, ctx Context) *Violation {
	var v *Violation
	syntax.Walk(file, func(n syntax.Node) bool {
		call, ok := n.(*syntax.CallExpr)
		if !ok || v != nil {
			return v == nil
		}
		args := callArgs(call)
		if len(args) < 3 {
			return true
		}
		name, ok := parse.WordLiteral(call.Args[0])
		if !ok || basename(name) != "make" {
			return true
		}
		for i := 1; i < len(args)-1; i++ {
			if args[i] != "-C" {
				continue
			}
			target := args[i+1]
			if target == "" {
				continue
			}
			abs := pathutil.Resolve(target, ctx.CWD)
			allowed := append([]string{ctx.CWD}, ctx.AllowedDirs...)
			inside := false
			for _, d := range allowed {
				if d != "" && pathutil.IsUnder(abs, pathutil.Resolve(d, ctx.CWD)) {
					inside = true
					break
				}
			}
			if !inside {
				v = &Violation{
					RuleID:  "make-dir-escape",
					Message: fmt.Sprintf("make -C %q escapes the working directory", target),
				}
				return false
			}
		}
		return true
	})
	return v
}

// EOF
