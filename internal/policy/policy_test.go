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

// EOF
