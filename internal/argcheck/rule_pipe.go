package argcheck

import (
	"fmt"

	"mvdan.cc/sh/v3/syntax"

	"github.com/htakahama/guard-bash/internal/parse"
)

func init() {
	register(Rule{ID: "pipe-to-shell", Check: checkPipeToShell})
}

func checkPipeToShell(file *syntax.File, _ Context) *Violation {
	var v *Violation
	syntax.Walk(file, func(n syntax.Node) bool {
		bin, ok := n.(*syntax.BinaryCmd)
		if !ok || v != nil {
			return v == nil
		}
		if bin.Op != syntax.Pipe {
			return true
		}
		// Inspect the right-hand side of the pipe.
		rhs := bin.Y
		if rhs == nil || rhs.Cmd == nil {
			return true
		}
		call, ok := rhs.Cmd.(*syntax.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		name, ok := parse.WordLiteral(call.Args[0])
		if !ok {
			return true
		}
		if shells[basename(name)] {
			v = &Violation{
				RuleID:  "pipe-to-shell",
				Message: fmt.Sprintf("piping to %q is blocked (remote code execution risk)", name),
			}
			return false
		}
		return true
	})
	return v
}

// EOF
