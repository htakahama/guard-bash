package policy_test

import (
	"testing"

	"github.com/htakahama/guard-bash/internal/extract"
	"github.com/htakahama/guard-bash/internal/policy"
)

func TestCheck(t *testing.T) {
	p := policy.New(
		policy.ModeAllowlist,
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
	p := policy.New(policy.ModeAllowlist, nil, nil)
	got := p.Check([]string{"git"})
	if got.Decision != policy.DecisionNotAllowed {
		t.Errorf("empty allowed should reject everything, got %v", got.Decision)
	}
}

func TestCheckDuplicateInBothLists(t *testing.T) {
	// A command in both allowed and denied should be denied (deny takes priority).
	p := policy.New(policy.ModeAllowlist, []string{"git", "rm"}, []string{"rm"})
	got := p.Check([]string{"rm"})
	if got.Decision != policy.DecisionDenyListed {
		t.Errorf("deny should take priority over allow, got %v", got.Decision)
	}
}

func TestCheckDenylistMode(t *testing.T) {
	// In denylist mode the allowlist is ignored: unknown and dynamic names
	// pass, but denied names are still blocked.
	p := policy.New(policy.ModeDenylist, nil, []string{"sudo", "dd"})

	cases := []struct {
		name  string
		input []string
		want  policy.Decision
		name2 string
	}{
		{"unknown passes", []string{"wget", "anything"}, policy.DecisionAllow, ""},
		{"dynamic passes", []string{extract.Dynamic, "git"}, policy.DecisionAllow, ""},
		{"denied still blocked", []string{"git", "sudo"}, policy.DecisionDenyListed, "sudo"},
		{"denied in middle", []string{"wget", "dd", "cat"}, policy.DecisionDenyListed, "dd"},
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

func TestParseMode(t *testing.T) {
	cases := []struct {
		in   string
		want policy.Mode
	}{
		{"denylist", policy.ModeDenylist},
		{"allowlist", policy.ModeAllowlist},
		{"", policy.ModeAllowlist},
		{"bogus", policy.ModeAllowlist},
	}
	for _, tc := range cases {
		if got := policy.ParseMode(tc.in); got != tc.want {
			t.Errorf("ParseMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
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
