// Package policy matches extracted command names against an allowlist and a
// denylist. Dynamic sentinels (extract.Dynamic) are always rejected.
package policy

import (
	"github.com/htakahama/guard-bash/internal/extract"
)

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

// Policy holds compiled allow/deny sets.
type Policy struct {
	allowed map[string]struct{}
	denied  map[string]struct{}
}

// New builds a policy from lists of names (duplicates are fine).
func New(allowed, denied []string) *Policy {
	p := &Policy{
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
// Returns {DecisionAllow, ""} when every name passes.
func (p *Policy) Check(commands []string) Result {
	for _, name := range commands {
		if name == extract.Dynamic {
			return Result{DecisionDynamic, name}
		}
		if _, ok := p.denied[name]; ok {
			return Result{DecisionDenyListed, name}
		}
		if _, ok := p.allowed[name]; !ok {
			return Result{DecisionNotAllowed, name}
		}
	}
	return Result{DecisionAllow, ""}
}

// EOF
