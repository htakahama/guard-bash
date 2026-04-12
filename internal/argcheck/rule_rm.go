package argcheck

import (
	"fmt"

	"mvdan.cc/sh/v3/syntax"

	"github.com/htakahama/guard-bash/internal/parse"
)

func init() {
	register(Rule{
		ID:    "rm-recursive-broad",
		Check: checkRmRecursiveBroad,
	})
}

func checkRmRecursiveBroad(file *syntax.File, _ Context) *Violation {
	var v *Violation
	syntax.Walk(file, func(n syntax.Node) bool {
		call, ok := n.(*syntax.CallExpr)
		if !ok || v != nil {
			return v == nil
		}
		args := callArgs(call)
		if len(args) == 0 {
			return true
		}
		name, ok := parse.WordLiteral(call.Args[0])
		if !ok || basename(name) != "rm" {
			return true
		}
		if !hasShortFlag(args[1:], 'r') && !hasShortFlag(args[1:], 'R') {
			return true
		}
		for _, p := range nonFlagArgs(args[1:]) {
			if isBroadPath(p) {
				v = &Violation{
					RuleID:  "rm-recursive-broad",
					Message: fmt.Sprintf("rm with recursive flag on broad path %q is blocked", p),
				}
				return false
			}
		}
		return true
	})
	return v
}

func basename(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return s[i+1:]
		}
	}
	return s
}

// EOF
