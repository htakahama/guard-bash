package hook_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/htakahama/guard-bash/internal/hook"
)

func TestRead(t *testing.T) {
	src := `{"cwd":"/home/user/x","tool_input":{"command":"git status","description":"check"}}`
	in, err := hook.Read(strings.NewReader(src))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if in.CWD != "/home/user/x" {
		t.Errorf("cwd = %q", in.CWD)
	}
	if in.ToolInput.Command != "git status" {
		t.Errorf("command = %q", in.ToolInput.Command)
	}
	if in.ToolInput.Description != "check" {
		t.Errorf("description = %q", in.ToolInput.Description)
	}
}

func TestReadMissingCwd(t *testing.T) {
	_, err := hook.Read(strings.NewReader(`{"tool_input":{"command":"git status"}}`))
	if err == nil {
		t.Fatal("expected error for missing cwd")
	}
}

func TestReadMissingCommand(t *testing.T) {
	_, err := hook.Read(strings.NewReader(`{"cwd":"/x","tool_input":{}}`))
	if err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestReadInvalidJSON(t *testing.T) {
	_, err := hook.Read(strings.NewReader("not json at all"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadEmptyInput(t *testing.T) {
	_, err := hook.Read(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestReadExtraFields(t *testing.T) {
	src := `{"cwd":"/x","tool_input":{"command":"git status"},"extra":"ignored"}`
	in, err := hook.Read(strings.NewReader(src))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if in.ToolInput.Command != "git status" {
		t.Errorf("command = %q", in.ToolInput.Command)
	}
}

func TestWriteAllow(t *testing.T) {
	var buf bytes.Buffer
	if err := hook.WriteAllow(&buf, "cd /x && git status", "doc"); err != nil {
		t.Fatalf("write: %v", err)
	}
	raw := buf.String()
	if strings.Contains(raw, `\u0026`) {
		t.Errorf("JSON contains escaped ampersand: %s", raw)
	}
	var out hook.Output
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("decision = %q", out.HookSpecificOutput.PermissionDecision)
	}
	if out.HookSpecificOutput.UpdatedInput.Command != "cd /x && git status" {
		t.Errorf("command = %q", out.HookSpecificOutput.UpdatedInput.Command)
	}
	if out.HookSpecificOutput.UpdatedInput.Description != "doc" {
		t.Errorf("description = %q", out.HookSpecificOutput.UpdatedInput.Description)
	}
}

func TestWriteAllowEmptyDescription(t *testing.T) {
	var buf bytes.Buffer
	if err := hook.WriteAllow(&buf, "git status", ""); err != nil {
		t.Fatalf("write: %v", err)
	}
	var out hook.Output
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.HookSpecificOutput.UpdatedInput.Description != "" {
		t.Errorf("expected empty description, got %q", out.HookSpecificOutput.UpdatedInput.Description)
	}
}

func TestWriteAllowSpecialChars(t *testing.T) {
	var buf bytes.Buffer
	if err := hook.WriteAllow(&buf, `echo "hello & world"`, "test <desc>"); err != nil {
		t.Fatalf("write: %v", err)
	}
	raw := buf.String()
	// Ensure HTML-sensitive characters are NOT escaped.
	if strings.Contains(raw, `\u0026`) {
		t.Errorf("& should not be escaped: %s", raw)
	}
	if strings.Contains(raw, `\u003c`) {
		t.Errorf("< should not be escaped: %s", raw)
	}
}

// EOF
