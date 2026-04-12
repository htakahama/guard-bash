// Package hook handles the stdin/stdout JSON contract that Claude Code uses
// for PreToolUse hooks.
package hook

import (
	"encoding/json"
	"fmt"
	"io"
)

// Input is the subset of the PreToolUse payload guard-bash consumes.
type Input struct {
	CWD       string    `json:"cwd"`
	ToolInput ToolInput `json:"tool_input"`
}

// ToolInput mirrors .tool_input for the Bash tool.
type ToolInput struct {
	Command     string `json:"command"`
	Description string `json:"description,omitempty"`
}

// Read parses a hook payload from r and validates the required fields.
func Read(r io.Reader) (*Input, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read hook input: %w", err)
	}
	var in Input
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("parse hook input: %w", err)
	}
	if in.CWD == "" {
		return nil, fmt.Errorf("hook input missing .cwd")
	}
	if in.ToolInput.Command == "" {
		return nil, fmt.Errorf("hook input missing .tool_input.command")
	}
	return &in, nil
}

// Output is the JSON guard-bash writes to stdout on allow.
type Output struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

// HookSpecificOutput is the nested envelope required by Claude Code.
type HookSpecificOutput struct {
	HookEventName      string    `json:"hookEventName"`
	PermissionDecision string    `json:"permissionDecision"`
	UpdatedInput       ToolInput `json:"updatedInput"`
}

// WriteAllow emits an "allow" response whose updatedInput.command is set to
// cmd (and description to desc, if non-empty).
func WriteAllow(w io.Writer, cmd, desc string) error {
	out := Output{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName:      "PreToolUse",
			PermissionDecision: "allow",
			UpdatedInput:       ToolInput{Command: cmd, Description: desc},
		},
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// EOF
