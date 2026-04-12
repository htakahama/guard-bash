package logging_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/htakahama/guard-bash/internal/logging"
)

func TestInit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger, closer, err := logging.Init(path, "info")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	defer closer()

	logger.Info("test message", "key", "value")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(data) == 0 {
		t.Error("log file is empty after writing")
	}
}

func TestInitCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "test.log")

	logger, closer, err := logging.Init(path, "debug")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	defer closer()

	logger.Debug("debug message")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("log file not created: %v", err)
	}
}

func TestInitLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "warning", "error", "unknown"} {
		t.Run(level, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.log")
			_, closer, err := logging.Init(path, level)
			if err != nil {
				t.Fatalf("init with level %q: %v", level, err)
			}
			closer()
		})
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/custom/state")
	got := logging.DefaultPath()
	want := "/custom/state/guard-bash/guard-bash.log"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDefaultPathFallback(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	got := logging.DefaultPath()
	if got == "" {
		t.Error("DefaultPath returned empty string")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("DefaultPath should return absolute path, got %q", got)
	}
}

func TestDiscard(t *testing.T) {
	logger := logging.Discard()
	// Should not panic.
	logger.Info("discarded message")
	logger.Error("discarded error")
}

// EOF
