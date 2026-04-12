package policy_test

import (
	"testing"

	"github.com/htakahama/guard-bash/internal/extract"
	"github.com/htakahama/guard-bash/internal/policy"
)

func TestCheck(t *testing.T) {
	p := policy.New(
		[]string{"git", "cat", "env"},
		[]string{"sudo", "rm"},
	)

	cases := []struct {
		name  string
		input []string
		want  policy.Decision
		name2 string
	}{
		{"all allowed", []string{"git", "cat"}, policy.DecisionAllow, ""},
		{"empty", []string{}, policy.DecisionAllow, ""},
		{"denied first", []string{"sudo", "git"}, policy.DecisionDenyListed, "sudo"},
		{"not allowed", []string{"git", "wget"}, policy.DecisionNotAllowed, "wget"},
		{"dynamic", []string{"git", extract.Dynamic}, policy.DecisionDynamic, extract.Dynamic},
		{"deny beats allow", []string{"rm"}, policy.DecisionDenyListed, "rm"},
		{"nil input", nil, policy.DecisionAllow, ""},
		{"all dynamic", []string{extract.Dynamic, extract.Dynamic}, policy.DecisionDynamic, extract.Dynamic},
		{"denied in middle", []string{"git", "sudo", "cat"}, policy.DecisionDenyListed, "sudo"},
		{"not allowed at end", []string{"git", "cat", "unknown"}, policy.DecisionNotAllowed, "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := p.Check(tc.input)
			if got.Decision != tc.want {
				t.Errorf("want decision %v, got %v (name=%q)", tc.want, got.Decision, got.Name)
			}
			if got.Name != tc.name2 {
				t.Errorf("want name %q, got %q", tc.name2, got.Name)
			}
		})
	}
}

func TestCheckEmptyPolicy(t *testing.T) {
	p := policy.New(nil, nil)
	got := p.Check([]string{"git"})
	if got.Decision != policy.DecisionNotAllowed {
		t.Errorf("empty allowed should reject everything, got %v", got.Decision)
	}
}

func TestCheckDuplicateInBothLists(t *testing.T) {
	// A command in both allowed and denied should be denied (deny takes priority).
	p := policy.New([]string{"git", "rm"}, []string{"rm"})
	got := p.Check([]string{"rm"})
	if got.Decision != policy.DecisionDenyListed {
		t.Errorf("deny should take priority over allow, got %v", got.Decision)
	}
}

func TestDecisionString(t *testing.T) {
	cases := []struct {
		d    policy.Decision
		want string
	}{
		{policy.DecisionAllow, "allow"},
		{policy.DecisionDenyListed, "deny-listed"},
		{policy.DecisionNotAllowed, "not-allowed"},
		{policy.DecisionDynamic, "dynamic"},
		{policy.Decision(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.d.String(); got != tc.want {
			t.Errorf("Decision(%d).String() = %q, want %q", tc.d, got, tc.want)
		}
	}
}

// EOF
