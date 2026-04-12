// Package logging initialises a slog JSON handler that appends to a log file
// under XDG_STATE_HOME (or $HOME/.local/state). The file is rotated externally
// (logrotate, journald, or similar); guard-bash only appends.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Init opens the log file (creating directories as needed) and returns a
// logger plus a close function that callers should defer. If filePath is
// empty, the default XDG path is used.
func Init(filePath, level string) (*slog.Logger, func() error, error) {
	path := filePath
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	h := slog.NewJSONHandler(f, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(h), f.Close, nil
}

// DefaultPath returns the conventional log file path.
func DefaultPath() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, ".local", "state")
		}
	}
	return filepath.Join(base, "guard-bash", "guard-bash.log")
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Discard returns a logger that throws all output away. Useful in tests.
func Discard() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// EOF
