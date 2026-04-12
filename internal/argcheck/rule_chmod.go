package argcheck

import (
	"fmt"

	"mvdan.cc/sh/v3/syntax"

	"github.com/htakahama/guard-bash/internal/parse"
)

func init() {
	register(Rule{ID: "chmod-recursive-broad", Check: checkChmodRecursiveBroad})
	register(Rule{ID: "chown-recursive-broad", Check: checkChownRecursiveBroad})
}

func checkRecursiveBroad(file *syntax.File, cmd, ruleID string) *Violation {
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
		if !ok || basename(name) != cmd {
			return true
		}
		if !hasShortFlag(args[1:], 'R') {
			return true
		}
		for _, p := range nonFlagArgs(args[1:]) {
			if isBroadPath(p) {
				v = &Violation{
					RuleID:  ruleID,
					Message: fmt.Sprintf("%s -R on broad path %q is blocked", cmd, p),
				}
				return false
			}
		}
		return true
	})
	return v
}

func checkChmodRecursiveBroad(file *syntax.File, _ Context) *Violation {
	return checkRecursiveBroad(file, "chmod", "chmod-recursive-broad")
}

func checkChownRecursiveBroad(file *syntax.File, _ Context) *Violation {
	return checkRecursiveBroad(file, "chown", "chown-recursive-broad")
}

// EOF
