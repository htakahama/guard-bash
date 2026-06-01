// Package transcript extracts the current model name from a Claude Code
// session transcript (JSONL). The PreToolUse hook payload does not carry the
// model directly; it only provides transcript_path. Each assistant entry in
// the transcript records its model, so the most recent assistant line tells
// us which model issued the tool call being checked.
package transcript

import (
	"bytes"
	"io"
	"os"
	"regexp"
)

// tailBytes bounds how much of the (potentially large) transcript we read.
// The most recent assistant entry is always at the very end, so reading the
// final chunk is sufficient and keeps the hook fast.
const tailBytes = 256 * 1024

var (
	assistantMarker = []byte(`"type":"assistant"`)
	modelRe         = regexp.MustCompile(`"model":"([^"]+)"`)
)

// CurrentModel returns the model name from the most recent assistant entry in
// the transcript JSONL at path, or "" when it cannot be determined (missing
// path, unreadable file, or no assistant entry with a model field).
func CurrentModel(path string) string {
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return ""
	}
	var offset int64
	if info.Size() > tailBytes {
		offset = info.Size() - tailBytes
	}
	buf := make([]byte, info.Size()-offset)
	if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
		return ""
	}

	// Scan lines from the end; the last assistant entry is the current model.
	// The leading line may be truncated by the tail cut, but the trailing
	// assistant entry we care about is always complete.
	lines := bytes.Split(buf, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if !bytes.Contains(lines[i], assistantMarker) {
			continue
		}
		if m := modelRe.FindSubmatch(lines[i]); m != nil {
			return string(m[1])
		}
	}
	return ""
}

// EOF
