package argcheck

import (
	"fmt"

	"mvdan.cc/sh/v3/syntax"

	"github.com/htakahama/guard-bash/internal/pathutil"
)

var protectedBranches = map[string]bool{
	"main":   true,
	"master": true,
}

func init() {
	register(Rule{ID: "git-push-force", Check: checkGitPushForce})
	register(Rule{ID: "git-reset-hard", Check: checkGitResetHard})
	register(Rule{ID: "git-dir-escape", Check: checkGitDirEscape})
}

func checkGitPushForce(file *syntax.File, _ Context) *Violation {
	var v *Violation
	syntax.Walk(file, func(n syntax.Node) bool {
		call, ok := n.(*syntax.CallExpr)
		if !ok || v != nil {
			return v == nil
		}
		args := callArgs(call)
		if len(args) < 2 {
			return true
		}
		if basename(args[0]) != "git" {
			return true
		}
		// Find the "push" subcommand, skipping git-level flags like -C.
		subIdx := gitSubcommandIndex(args)
		if subIdx < 0 || args[subIdx] != "push" {
			return true
		}
		pushArgs := args[subIdx+1:]
		hasForce := hasLongFlag(pushArgs, "--force") ||
			hasLongFlag(pushArgs, "--force-with-lease") ||
			hasShortFlag(pushArgs, 'f')
		if !hasForce {
			return true
		}
		// Check if any non-flag arg is a protected branch name.
		positional := nonFlagArgs(pushArgs)
		if len(positional) == 0 {
			// No refspec: conservative block.
			v = &Violation{
				RuleID:  "git-push-force",
				Message: "git push --force without explicit refspec is blocked",
			}
			return false
		}
		for _, arg := range positional {
			if protectedBranches[arg] {
				v = &Violation{
					RuleID:  "git-push-force",
					Message: fmt.Sprintf("git push --force to protected branch %q is blocked", arg),
				}
				return false
			}
		}
		return true
	})
	return v
}

func checkGitResetHard(file *syntax.File, _ Context) *Violation {
	var v *Violation
	syntax.Walk(file, func(n syntax.Node) bool {
		call, ok := n.(*syntax.CallExpr)
		if !ok || v != nil {
			return v == nil
		}
		args := callArgs(call)
		if len(args) < 2 {
			return true
		}
		if basename(args[0]) != "git" {
			return true
		}
		subIdx := gitSubcommandIndex(args)
		if subIdx < 0 || args[subIdx] != "reset" {
			return true
		}
		if hasLongFlag(args[subIdx+1:], "--hard") {
			v = &Violation{
				RuleID:  "git-reset-hard",
				Message: "git reset --hard is blocked (discards uncommitted changes)",
			}
			return false
		}
		return true
	})
	return v
}

func checkGitDirEscape(file *syntax.File, ctx Context) *Violation {
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
		if basename(args[0]) != "git" {
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
					RuleID:  "git-dir-escape",
					Message: fmt.Sprintf("git -C %q escapes the working directory", target),
				}
				return false
			}
		}
		return true
	})
	return v
}

// gitSubcommandIndex returns the index of the first non-flag argument after
// the "git" command, skipping flags like -C <path> and -c <key=val>.
func gitSubcommandIndex(args []string) int {
	i := 1
	for i < len(args) {
		a := args[i]
		if !isGitFlag(a) {
			return i
		}
		// -C and -c take an argument.
		if a == "-C" || a == "-c" {
			i += 2
			continue
		}
		i++
	}
	return -1
}

func isGitFlag(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

// EOF
