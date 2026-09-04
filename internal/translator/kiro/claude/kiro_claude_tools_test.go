package claude

import (
	"encoding/json"
	"testing"
)

// TestRepairJSON_QuirkyButComplete verifies that RepairJSON preserves valid
// JSON that contains shell-escape characters, embedded newlines, and
// trailing-comma-free structures — i.e. the cases that historically tripped
// the heuristic and produced "input: {}" downstream.
func TestRepairJSON_QuirkyButComplete(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]interface{}
	}{
		{
			name: "shell command with embedded quotes",
			in:   `{"command":"echo \"hello world\""}`,
			want: map[string]interface{}{"command": "echo \"hello world\""},
		},
		{
			name: "shell command with literal newline",
			in:   "{\"command\":\"echo a\\necho b\"}",
			want: map[string]interface{}{"command": "echo a\necho b"},
		},
		{
			name: "bash with description and timeout",
			in:   `{"command":"ls -la","description":"list files","timeout":5000}`,
			want: map[string]interface{}{
				"command":     "ls -la",
				"description": "list files",
				"timeout":     float64(5000),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repaired := RepairJSON(tc.in)
			var got map[string]interface{}
			if err := json.Unmarshal([]byte(repaired), &got); err != nil {
				t.Fatalf("RepairJSON output failed to parse: %v\nrepaired: %s", err, repaired)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d keys, want %d (got=%v want=%v)", len(got), len(tc.want), got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("key %q: got %v (%T), want %v (%T)", k, got[k], got[k], v, v)
				}
			}
		})
	}
}

// TestRecoverPartialInput_Empty confirms the fallback helper returns a
// non-nil empty map when nothing can be salvaged, instead of nil (which
// would marshal to JSON null).
func TestRecoverPartialInput_Empty(t *testing.T) {
	got := RecoverPartialInput(nil)
	if got == nil {
		t.Fatal("RecoverPartialInput(nil) returned nil; want empty map")
	}
	if len(got) != 0 {
		t.Fatalf("RecoverPartialInput(nil) returned %d keys; want 0", len(got))
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "{}" {
		t.Fatalf("marshal output = %s, want {}", string(b))
	}
}

// TestRecoverPartialInput_PreservesFields verifies that when extractPartialFields
// finds recoverable fields (e.g. "command" prefix), they survive the type
// conversion into the format BuildClaudeResponse emits as tool_use input.
func TestRecoverPartialInput_PreservesFields(t *testing.T) {
	partial := map[string]string{
		"command":     "echo hello",
		"description": "test command",
	}
	got := RecoverPartialInput(partial)
	if got["command"] != "echo hello" {
		t.Errorf("command lost in conversion: got %v", got["command"])
	}
	if got["description"] != "test command" {
		t.Errorf("description lost in conversion: got %v", got["description"])
	}
}

// TestProcessToolUseEvent_RepairFailureRecoversPartial is the regression test
// for the user-reported bug: kiro streams a tool_use buffer that fails
// RepairJSON+Unmarshal, and the tool_use block is emitted with input: {} —
// causing the downstream SDK to reject the call with
// "Invalid args for tool \"Bash\": must have required property 'command'".
//
// The fix replaces the silent empty-map fallback with extractPartialFields
// recovery. This test feeds a stream whose final buffer is unparseable but
// still contains a recognizable "command" field.
func TestProcessToolUseEvent_RepairFailureRecoversPartial(t *testing.T) {
	processed := map[string]bool{}

	// Start the tool use with a complete first chunk.
	startEvent := map[string]interface{}{
		"toolUseEvent": map[string]interface{}{
			"toolUseId": "tu_test_1",
			"name":      "Bash",
			"stop":      false,
			"input":     `{"command":`,
		},
	}
	_, state := ProcessToolUseEvent(startEvent, nil, processed)
	if state == nil {
		t.Fatal("expected non-nil state from tool use start")
	}
	if state.ToolUseID != "tu_test_1" {
		t.Fatalf("state.ToolUseID = %s, want tu_test_1", state.ToolUseID)
	}

	// Append fragments. The cumulative buffer becomes an unparseable JSON
	// (truncated mid-key), simulating the upstream failure mode that
	// historically led to input: {}.
	state.InputBuffer.WriteString(`"echo a; echo b","descrip`)

	stopEvent := map[string]interface{}{
		"toolUseEvent": map[string]interface{}{
			"toolUseId": "tu_test_1",
			"name":      "Bash",
			"stop":      true,
		},
	}
	results, finalState := ProcessToolUseEvent(stopEvent, state, processed)
	if finalState != nil {
		t.Errorf("expected nil state after stop, got %+v", finalState)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	tu := results[0]
	if tu.Name != "Bash" {
		t.Errorf("name = %s, want Bash", tu.Name)
	}
	// The critical assertion: input MUST NOT be empty for a tool with required
	 // fields, otherwise the downstream SDK rejects the call.
	if tu.Input == nil {
		t.Fatal("input is nil — should be empty map or partial fields")
	}
	if _, hasCmd := tu.Input["command"]; !hasCmd {
		t.Fatalf("input missing 'command' field — SDK will reject the call. got=%v", tu.Input)
	}
}