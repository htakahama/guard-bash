// Command guard-bash is the Claude Code PreToolUse hook entry point. It
// reads a JSON payload on stdin, parses the Bash command, walks the AST to
// collect every command name that would run, matches them against policy
// lists, verifies any leading `cd` target, and emits an allow/deny decision
// on stdout (for allow) or stderr + exit 2 (for deny).
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"sort"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/htakahama/guard-bash/internal/argcheck"
	"github.com/htakahama/guard-bash/internal/checkcd"
	"github.com/htakahama/guard-bash/internal/config"
	"github.com/htakahama/guard-bash/internal/extract"
	"github.com/htakahama/guard-bash/internal/hook"
	"github.com/htakahama/guard-bash/internal/logging"
	"github.com/htakahama/guard-bash/internal/parse"
	"github.com/htakahama/guard-bash/internal/policy"
)

// Injected via -ldflags at release build time by GoReleaser. In dev builds
// these remain at their default values.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("guard-bash %s (commit %s, built %s)\n", version, commit, date)
			return 0
		case "stat":
			tomlFlag := slices.Contains(os.Args[2:], "--toml")
			return runStat(tomlFlag)
		}
	}

	start := time.Now()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "BLOCKED: config: %v\n", err)
		return 2
	}

	logger, closeLog, err := logging.Init(cfg.Logging.File, cfg.Logging.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "BLOCKED: logging: %v\n", err)
		return 2
	}
	defer func() { _ = closeLog() }()

	if err := run(os.Stdin, os.Stdout, cfg, logger, start); err != nil {
		logger.Warn("deny", "err", err.Error(), "duration_ms", time.Since(start).Milliseconds())
		fmt.Fprintf(os.Stderr, "BLOCKED: %v\n", err)
		return 2
	}
	return 0
}

func runStat(tomlOut bool) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: config: %v\n", err)
		return 1
	}

	if tomlOut {
		return printEffectiveTOML(cfg)
	}
	printStat(cfg)
	return 0
}

func printStat(cfg *config.Config) {
	// Config source
	path := config.UserConfigPath()
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("config_file: %s\n", path)
	} else {
		fmt.Println("config_file: (embedded default)")
	}

	// Environment overrides
	fmt.Println()
	fmt.Println("environment:")
	envVars := []string{
		"GUARD_CONFIG",
		"GUARD_EXTRA_ALLOWED",
		"GUARD_EXTRA_DENIED",
		"GUARD_ALLOWED_DIRS",
		"GUARD_ARGCHECK_DISABLED",
		"GUARD_LOG_LEVEL",
		"GUARD_LOG_FILE",
	}
	for _, k := range envVars {
		v := os.Getenv(k)
		if v != "" {
			fmt.Printf("  %s=%s\n", k, v)
		} else {
			fmt.Printf("  %s=\n", k)
		}
	}

	// Policy summary
	allowed := cfg.MergedAllowed()
	denied := cfg.MergedDenied()
	sort.Strings(allowed)
	sort.Strings(denied)
	fmt.Println()
	fmt.Printf("policy.allowed: %d commands\n", len(allowed))
	fmt.Printf("policy.denied:  %d commands\n", len(denied))

	// Argcheck rules
	disabled := cfg.DisabledArgCheckSet()
	fmt.Println()
	fmt.Println("argcheck rules:")
	for _, id := range argcheck.RuleIDs() {
		status := "enabled"
		if disabled[id] {
			status = "disabled"
		}
		fmt.Printf("  %-25s %s\n", id, status)
	}

	// Logging
	logPath := cfg.Logging.File
	if logPath == "" {
		logPath = logging.DefaultPath()
	}
	fmt.Println()
	fmt.Printf("logging.level: %s\n", cfg.Logging.Level)
	fmt.Printf("logging.file:  %s\n", logPath)
}

// effectiveConfig is the flattened config written by --toml.
type effectiveConfig struct {
	Policy   effectivePolicy `toml:"policy"`
	CheckCD  effectiveCD     `toml:"checkcd"`
	ArgCheck effectiveAC     `toml:"argcheck"`
	Logging  effectiveLog    `toml:"logging"`
}

type effectivePolicy struct {
	Allowed []string `toml:"allowed"`
	Denied  []string `toml:"denied"`
}

type effectiveCD struct {
	AllowedDirs []string `toml:"allowed_dirs"`
}

type effectiveAC struct {
	Disabled []string `toml:"disabled"`
}

type effectiveLog struct {
	Level string `toml:"level"`
	File  string `toml:"file"`
}

func printEffectiveTOML(cfg *config.Config) int {
	allowed := cfg.MergedAllowed()
	denied := cfg.MergedDenied()
	sort.Strings(allowed)
	sort.Strings(denied)

	logFile := cfg.Logging.File
	if logFile == "" {
		logFile = logging.DefaultPath()
	}

	eff := effectiveConfig{
		Policy: effectivePolicy{
			Allowed: allowed,
			Denied:  denied,
		},
		CheckCD: effectiveCD{
			AllowedDirs: cfg.CheckCD.AllowedDirs,
		},
		ArgCheck: effectiveAC{
			Disabled: cfg.ArgCheck.Disabled,
		},
		Logging: effectiveLog{
			Level: cfg.Logging.Level,
			File:  logFile,
		},
	}

	// Ensure empty slices render as [] not omitted.
	if eff.CheckCD.AllowedDirs == nil {
		eff.CheckCD.AllowedDirs = []string{}
	}
	if eff.ArgCheck.Disabled == nil {
		eff.ArgCheck.Disabled = []string{}
	}

	data, err := toml.Marshal(eff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: marshal: %v\n", err)
		return 1
	}
	os.Stdout.Write(data)
	return 0
}

// run is the unit-testable core. It never touches os.Exit so tests can drive
// it with in-memory buffers.
func run(stdin *os.File, stdout *os.File, cfg *config.Config, logger *slog.Logger, start time.Time) error {
	in, err := hook.Read(stdin)
	if err != nil {
		return err
	}

	if err := verifyGitRepo(in.CWD); err != nil {
		return err
	}

	file, err := parse.Parse(in.ToolInput.Command)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	cmds := extract.Commands(file)
	p := policy.New(cfg.MergedAllowed(), cfg.MergedDenied())
	res := p.Check(cmds)
	if res.Decision != policy.DecisionAllow {
		return policyError(res, cmds)
	}

	ac := argcheck.New(cfg.DisabledArgCheckSet())
	if v := ac.Check(file, argcheck.Context{CWD: in.CWD, AllowedDirs: cfg.CheckCD.AllowedDirs}); v != nil {
		return fmt.Errorf("dangerous arguments blocked [%s]: %s", v.RuleID, v.Message)
	}

	verdict, err := checkcd.Check(file, in.CWD, cfg.CheckCD.AllowedDirs)
	if err != nil {
		return err
	}

	fixed := in.ToolInput.Command
	if verdict == checkcd.NeedsPrepend {
		fixed = fmt.Sprintf("cd %s && %s", in.CWD, in.ToolInput.Command)
	}

	if err := hook.WriteAllow(stdout, fixed, in.ToolInput.Description); err != nil {
		return err
	}

	logger.Info("allow",
		"cwd", in.CWD,
		"cmd", in.ToolInput.Command,
		"extracted", cmds,
		"fixed_cmd", fixed,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func verifyGitRepo(cwd string) error {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-dir")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cwd %q is not inside a git repository: %s", cwd, string(out))
	}
	return nil
}

func policyError(r policy.Result, _ []string) error {
	switch r.Decision {
	case policy.DecisionDenyListed:
		return fmt.Errorf("%q is in the denied command list", r.Name)
	case policy.DecisionNotAllowed:
		return fmt.Errorf("%q is not in the allowed command list", r.Name)
	case policy.DecisionDynamic:
		return errors.New("dynamic command name (variable / command substitution) is not allowed")
	}
	return errors.New("unknown policy decision")
}

// EOF
