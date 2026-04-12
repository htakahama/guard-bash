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

func cleanEnv(t *testing.T) {
	for _, k := range []string{
		"GUARD_CONFIG", "XDG_CONFIG_HOME",
		"GUARD_EXTRA_ALLOWED", "GUARD_EXTRA_DENIED",
		"GUARD_ALLOWED_DIRS", "GUARD_LOG_LEVEL", "GUARD_LOG_FILE",
	} {
		t.Setenv(k, "")
	}
	t.Setenv("HOME", t.TempDir())
}

// EOF
