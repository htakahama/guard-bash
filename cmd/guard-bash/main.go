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
	"time"

	"github.com/htakahama/guard-bash/internal/checkcd"
	"github.com/htakahama/guard-bash/internal/config"
	"github.com/htakahama/guard-bash/internal/extract"
	"github.com/htakahama/guard-bash/internal/hook"
	"github.com/htakahama/guard-bash/internal/logging"
	"github.com/htakahama/guard-bash/internal/parse"
	"github.com/htakahama/guard-bash/internal/policy"
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
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
