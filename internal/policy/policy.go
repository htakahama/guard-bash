// Package policy matches extracted command names against a denylist and,
// in allowlist mode, an allowlist. The mode selects how strict the check is:
//
//   - ModeAllowlist (strict): every name must be on the allowlist, dynamic
//     command names are rejected, and denied names are blocked.
//   - ModeDenylist (lenient): only denied names are blocked; unknown and
//     dynamic command names pass through. Argument-level rules (argcheck) and
//     cwd management still apply downstream.
package policy

import (
	"github.com/htakahama/guard-bash/internal/extract"
)

// Mode selects the policy strategy. The zero value is ModeAllowlist so an
// unset / unknown config fails safe to the stricter behavior.
type Mode string

const (
	ModeAllowlist Mode = "allowlist"
	ModeDenylist  Mode = "denylist"
)

// ParseMode maps a config string to a Mode, defaulting unknown values to the
// stricter ModeAllowlist.
func ParseMode(s string) Mode {
	if Mode(s) == ModeDenylist {
		return ModeDenylist
	}
	return ModeAllowlist
}

// Decision describes why a command name was (not) allowed.
type Decision int

const (
	DecisionAllow Decision = iota
	DecisionDenyListed
	DecisionNotAllowed
	DecisionDynamic
)

func (d Decision) String() string {
	switch d {
	case DecisionAllow:
		return "allow"
	case DecisionDenyListed:
		return "deny-listed"
	case DecisionNotAllowed:
		return "not-allowed"
	case DecisionDynamic:
		return "dynamic"
	}
	return "unknown"
}

// Result is the verdict for a single command name. Decision is DecisionAllow
// when every name in the input was allowed; otherwise Name holds the first
// offending name.
type Result struct {
	Decision Decision
	Name     string
}

// Policy holds compiled allow/deny sets and the active mode.
type Policy struct {
	mode    Mode
	allowed map[string]struct{}
	denied  map[string]struct{}
}

// New builds a policy from lists of names (duplicates are fine). The mode
// selects whether the allowlist is enforced.
func New(mode Mode, allowed, denied []string) *Policy {
	p := &Policy{
		mode:    mode,
		allowed: make(map[string]struct{}, len(allowed)),
		denied:  make(map[string]struct{}, len(denied)),
	}
	for _, s := range allowed {
		p.allowed[s] = struct{}{}
	}
	for _, s := range denied {
		p.denied[s] = struct{}{}
	}
	return p
}

// Check walks commands in order and returns the first non-allow result.
// Returns {DecisionAllow, ""} when every name passes. In ModeDenylist the
// allowlist is not consulted and dynamic command names pass through; the
// denylist is enforced in both modes.
func (p *Policy) Check(commands []string) Result {
	for _, name := range commands {
		if name == extract.Dynamic {
			if p.mode == ModeDenylist {
				continue
			}
			return Result{DecisionDynamic, name}
		}
		if _, ok := p.denied[name]; ok {
			return Result{DecisionDenyListed, name}
		}
		if p.mode == ModeDenylist {
			continue
		}
		if _, ok := p.allowed[name]; !ok {
			return Result{DecisionNotAllowed, name}
		}
	}
	return Result{DecisionAllow, ""}
}

// EOF
