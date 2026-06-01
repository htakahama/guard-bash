package transcript_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/htakahama/guard-bash/internal/transcript"
)

func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	return path
}

func TestCurrentModel(t *testing.T) {
	user := `{"type":"user","message":{"role":"user"}}`
	opus := `{"type":"assistant","model":"claude-opus-4-8","message":{"role":"assistant"}}`
	sonnet := `{"type":"assistant","model":"claude-sonnet-4-6","message":{"role":"assistant"}}`

	cases := []struct {
		name  string
		lines []string
		want  string
	}{
		{"single assistant", []string{user, opus}, "claude-opus-4-8"},
		{"last assistant wins", []string{opus, user, sonnet}, "claude-sonnet-4-6"},
		{"ignores user lines", []string{sonnet, user, user}, "claude-sonnet-4-6"},
		{"no assistant", []string{user, user}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := transcript.CurrentModel(writeTranscript(t, tc.lines...))
			if got != tc.want {
				t.Errorf("CurrentModel = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCurrentModelMissingOrEmpty(t *testing.T) {
	if got := transcript.CurrentModel(""); got != "" {
		t.Errorf("empty path: got %q, want \"\"", got)
	}
	if got := transcript.CurrentModel("/nonexistent/transcript.jsonl"); got != "" {
		t.Errorf("missing file: got %q, want \"\"", got)
	}
}

// EOF
