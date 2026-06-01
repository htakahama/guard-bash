package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/htakahama/guard-bash/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	cleanEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	allowed := cfg.MergedAllowed()
	for _, want := range []string{"git", "go", "cat", "cd"} {
		if !slices.Contains(allowed, want) {
			t.Errorf("missing %q in default allowed", want)
		}
	}
	denied := cfg.MergedDenied()
	for _, want := range []string{"sudo", "eval", "reboot"} {
		if !slices.Contains(denied, want) {
			t.Errorf("missing %q in default denied", want)
		}
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("default level = %q, want info", cfg.Logging.Level)
	}
	if cfg.Policy.Mode != "denylist" {
		t.Errorf("default policy.mode = %q, want denylist", cfg.Policy.Mode)
	}
}

func TestResolveMode(t *testing.T) {
	cfg := &config.Config{Policy: config.PolicyConfig{
		Mode: "allowlist",
		ModelRules: []config.ModelRule{
			{Match: "opus", Mode: "denylist"},
		},
	}}
	cases := []struct {
		model string
		want  string
	}{
		{"claude-opus-4-8", "denylist"},
		{"claude-sonnet-4-6", "allowlist"}, // no rule match -> fallback
		{"", "allowlist"},                  // unknown model -> fallback
	}
	for _, tc := range cases {
		if got := cfg.ResolveMode(tc.model); got != tc.want {
			t.Errorf("ResolveMode(%q) = %q, want %q", tc.model, got, tc.want)
		}
	}
}

func TestModelRulesFromUserConfig(t *testing.T) {
	cleanEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[policy]
mode = "allowlist"

[[policy.model_rules]]
match = "opus"
mode  = "denylist"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("GUARD_CONFIG", path)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := cfg.ResolveMode("claude-opus-4-8"); got != "denylist" {
		t.Errorf("opus -> %q, want denylist", got)
	}
	if got := cfg.ResolveMode("claude-sonnet-4-6"); got != "allowlist" {
		t.Errorf("sonnet -> %q, want allowlist", got)
	}
}

func TestEnvOverrideClearsModelRules(t *testing.T) {
	cleanEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[policy]
mode = "allowlist"

[[policy.model_rules]]
match = "opus"
mode  = "denylist"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("GUARD_CONFIG", path)
	t.Setenv("GUARD_POLICY_MODE", "denylist")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Explicit env override forces the mode for every model, ignoring rules.
	if got := cfg.ResolveMode("claude-sonnet-4-6"); got != "denylist" {
		t.Errorf("env override should force denylist, got %q", got)
	}
	if len(cfg.Policy.ModelRules) != 0 {
		t.Errorf("env override should clear model rules, got %v", cfg.Policy.ModelRules)
	}
}

func TestEnvOverrides(t *testing.T) {
	cleanEnv(t)
	t.Setenv("GUARD_EXTRA_ALLOWED", "foo:bar")
	t.Setenv("GUARD_EXTRA_DENIED", "git")
	t.Setenv("GUARD_LOG_LEVEL", "debug")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	allowed := cfg.MergedAllowed()
	if !slices.Contains(allowed, "foo") || !slices.Contains(allowed, "bar") {
		t.Errorf("extra_allowed not merged: %v", allowed)
	}
	if slices.Contains(allowed, "git") {
		t.Errorf("git should be removed from allowed via extra_denied")
	}
	if !slices.Contains(cfg.MergedDenied(), "git") {
		t.Errorf("git should be in denied via extra_denied")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("level = %q, want debug", cfg.Logging.Level)
	}
}

func TestUserConfigFile(t *testing.T) {
	cleanEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[policy]
extra_allowed = ["mytool"]

[logging]
level = "warn"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("GUARD_CONFIG", path)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !slices.Contains(cfg.MergedAllowed(), "mytool") {
		t.Errorf("mytool should be merged from user config")
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("level = %q, want warn", cfg.Logging.Level)
	}
}

func TestInvalidTOMLConfig(t *testing.T) {
	cleanEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[[invalid toml!@#"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("GUARD_CONFIG", path)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestAllowedFullReplace(t *testing.T) {
	cleanEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[policy]
allowed = ["mytool", "myother"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("GUARD_CONFIG", path)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	allowed := cfg.MergedAllowed()
	// Should contain only the user-specified commands, not defaults like "git".
	if slices.Contains(allowed, "git") {
		t.Errorf("full replace should not include default 'git'")
	}
	if !slices.Contains(allowed, "mytool") {
		t.Errorf("missing mytool in allowed")
	}
}

func TestSplitColonEdgeCases(t *testing.T) {
	cleanEnv(t)
	// Leading/trailing colons and empty segments should be ignored.
	t.Setenv("GUARD_EXTRA_ALLOWED", ":foo::bar:")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	allowed := cfg.MergedAllowed()
	if !slices.Contains(allowed, "foo") || !slices.Contains(allowed, "bar") {
		t.Errorf("expected foo and bar in allowed, got %v", allowed)
	}
}

func TestMergedDeniedUnion(t *testing.T) {
	cleanEnv(t)
	t.Setenv("GUARD_EXTRA_DENIED", "badtool")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	denied := cfg.MergedDenied()
	// Should contain both default denied and extra denied.
	if !slices.Contains(denied, "sudo") {
		t.Errorf("missing default 'sudo' in denied")
	}
	if !slices.Contains(denied, "badtool") {
		t.Errorf("missing extra 'badtool' in denied")
	}
}

func cleanEnv(t *testing.T) {
	for _, k := range []string{
		"GUARD_CONFIG", "XDG_CONFIG_HOME", "GUARD_POLICY_MODE",
		"GUARD_EXTRA_ALLOWED", "GUARD_EXTRA_DENIED",
		"GUARD_ALLOWED_DIRS", "GUARD_ARGCHECK_DISABLED",
		"GUARD_LOG_LEVEL", "GUARD_LOG_FILE",
	} {
		t.Setenv(k, "")
	}
	t.Setenv("HOME", t.TempDir())
}

// EOF
