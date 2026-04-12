// Package argcheck walks a Bash AST and blocks dangerous argument patterns
// for commands that pass the allowlist. Each rule is a Go function keyed by
// a string ID; users can disable individual rules via config.
package argcheck

import (
	"mvdan.cc/sh/v3/syntax"
)

// Rule is a named checker that inspects the AST for a specific danger.
type Rule struct {
	ID    string
	Check func(file *syntax.File, ctx Context) *Violation
}

// Context carries environment that rules may need.
type Context struct {
	CWD         string
	AllowedDirs []string
}

// Violation describes why a command was blocked.
type Violation struct {
	RuleID  string
	Message string
}

// Checker holds the set of active rules.
type Checker struct {
	rules []Rule
}

// New builds a Checker from the default rule registry, excluding disabled IDs.
func New(disabled map[string]bool) *Checker {
	var active []Rule
	for _, r := range defaultRules {
		if !disabled[r.ID] {
			active = append(active, r)
		}
	}
	return &Checker{rules: active}
}

// Check runs every enabled rule against the AST and returns the first
// violation, or nil if all rules pass.
func (c *Checker) Check(file *syntax.File, ctx Context) *Violation {
	for _, r := range c.rules {
		if v := r.Check(file, ctx); v != nil {
			return v
		}
	}
	return nil
}

// defaultRules is the registry of built-in rules. Each rule_*.go file
// appends to this slice via init().
var defaultRules []Rule

func register(r Rule) {
	defaultRules = append(defaultRules, r)
}

// RuleIDs returns the IDs of all registered default rules in registration
// order.
func RuleIDs() []string {
	out := make([]string, len(defaultRules))
	for i, r := range defaultRules {
		out[i] = r.ID
	}
	return out
}

// EOF
