// Package config loads guard-bash settings from an embedded default TOML
// and optionally merges user overrides from a file and environment
// variables.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

//go:embed default.toml
var defaultTOML []byte

// Config is the whole guard-bash configuration.
type Config struct {
	Policy   PolicyConfig   `toml:"policy"`
	CheckCD  CheckCDConfig  `toml:"checkcd"`
	ArgCheck ArgCheckConfig `toml:"argcheck"`
	Logging  LoggingConfig  `toml:"logging"`
}

// PolicyConfig controls allow/deny lists. Allowed/Denied replace the default
// entirely when set by the user; ExtraAllowed/ExtraDenied append.
type PolicyConfig struct {
	Allowed      []string `toml:"allowed"`
	Denied       []string `toml:"denied"`
	ExtraAllowed []string `toml:"extra_allowed"`
	ExtraDenied  []string `toml:"extra_denied"`
}

// CheckCDConfig adds directories beyond the hook's cwd that a leading `cd`
// may target without being rewritten.
type CheckCDConfig struct {
	AllowedDirs []string `toml:"allowed_dirs"`
}

// ArgCheckConfig controls argument-level safety rules.
type ArgCheckConfig struct {
	Disabled []string `toml:"disabled"`
}

// LoggingConfig configures slog output.
type LoggingConfig struct {
	Level string `toml:"level"`
	File  string `toml:"file"`
}

// Load reads embedded defaults, merges a user config file (if any), and
// applies environment-variable overrides.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := toml.Unmarshal(defaultTOML, cfg); err != nil {
		return nil, fmt.Errorf("parse embedded default.toml: %w", err)
	}

	if path := userConfigPath(); path != "" {
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			var user Config
			if err := toml.Unmarshal(data, &user); err != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}
			mergeUser(cfg, &user)
		case os.IsNotExist(err):
			// user config is optional
		default:
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
	}

	applyEnv(cfg)
	return cfg, nil
}

// UserConfigPath returns the path that would be used for the user config
// file, or "" if none is found.
func UserConfigPath() string {
	return userConfigPath()
}

func userConfigPath() string {
	if p := os.Getenv("GUARD_CONFIG"); p != "" {
		return p
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "guard-bash", "config.toml")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "guard-bash", "config.toml")
	}
	return ""
}

func mergeUser(base, user *Config) {
	if len(user.Policy.Allowed) > 0 {
		base.Policy.Allowed = user.Policy.Allowed
	}
	if len(user.Policy.Denied) > 0 {
		base.Policy.Denied = user.Policy.Denied
	}
	base.Policy.ExtraAllowed = append(base.Policy.ExtraAllowed, user.Policy.ExtraAllowed...)
	base.Policy.ExtraDenied = append(base.Policy.ExtraDenied, user.Policy.ExtraDenied...)
	base.CheckCD.AllowedDirs = append(base.CheckCD.AllowedDirs, user.CheckCD.AllowedDirs...)
	base.ArgCheck.Disabled = append(base.ArgCheck.Disabled, user.ArgCheck.Disabled...)
	if user.Logging.Level != "" {
		base.Logging.Level = user.Logging.Level
	}
	if user.Logging.File != "" {
		base.Logging.File = user.Logging.File
	}
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("GUARD_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("GUARD_LOG_FILE"); v != "" {
		cfg.Logging.File = v
	}
	if v := os.Getenv("GUARD_EXTRA_ALLOWED"); v != "" {
		cfg.Policy.ExtraAllowed = append(cfg.Policy.ExtraAllowed, splitColon(v)...)
	}
	if v := os.Getenv("GUARD_EXTRA_DENIED"); v != "" {
		cfg.Policy.ExtraDenied = append(cfg.Policy.ExtraDenied, splitColon(v)...)
	}
	if v := os.Getenv("GUARD_ALLOWED_DIRS"); v != "" {
		cfg.CheckCD.AllowedDirs = append(cfg.CheckCD.AllowedDirs, splitColon(v)...)
	}
	if v := os.Getenv("GUARD_ARGCHECK_DISABLED"); v != "" {
		cfg.ArgCheck.Disabled = append(cfg.ArgCheck.Disabled, splitColon(v)...)
	}
}

func splitColon(s string) []string {
	parts := strings.Split(s, ":")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// MergedAllowed returns the effective allow set after unioning Allowed and
// ExtraAllowed and removing anything in ExtraDenied.
func (c *Config) MergedAllowed() []string {
	set := make(map[string]struct{}, len(c.Policy.Allowed)+len(c.Policy.ExtraAllowed))
	for _, s := range c.Policy.Allowed {
		set[s] = struct{}{}
	}
	for _, s := range c.Policy.ExtraAllowed {
		set[s] = struct{}{}
	}
	for _, s := range c.Policy.ExtraDenied {
		delete(set, s)
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	return out
}

// MergedDenied returns the union of Denied and ExtraDenied.
func (c *Config) MergedDenied() []string {
	set := make(map[string]struct{}, len(c.Policy.Denied)+len(c.Policy.ExtraDenied))
	for _, s := range c.Policy.Denied {
		set[s] = struct{}{}
	}
	for _, s := range c.Policy.ExtraDenied {
		set[s] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	return out
}

// DisabledArgCheckSet returns the set of disabled argcheck rule IDs.
func (c *Config) DisabledArgCheckSet() map[string]bool {
	set := make(map[string]bool, len(c.ArgCheck.Disabled))
	for _, id := range c.ArgCheck.Disabled {
		set[id] = true
	}
	return set
}

// EOF
