package output

import "testing"

func TestSimplifyDisplayLineForClaudeJSON(t *testing.T) {
	line := `{"message":{"content":[{"text":"Hello from Claude"}]}}`
	got := SimplifyDisplayLine("claude-code", line)
	if got != "Hello from Claude" {
		t.Fatalf("unexpected simplified line: %q", got)
	}
}

func TestSimplifyDisplayLineLeavesOtherAgentsUntouched(t *testing.T) {
	line := `{"message":{"content":[{"text":"Hello from Claude"}]}}`
	got := SimplifyDisplayLine("codex", line)
	if got != line {
		t.Fatalf("expected non-Claude line to remain unchanged, got %q", got)
	}
}

func TestFormatToolSummary(t *testing.T) {
	got := FormatToolSummary(map[string]int{"question": 1, "edit": 2})
	if got != "[loop] tools: edit=2, question=1" {
		t.Fatalf("unexpected tool summary: %q", got)
	}
}
