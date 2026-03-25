package agent

import "testing"

func TestParseRotation(t *testing.T) {
	got, err := ParseRotation("codex:gpt-5,claude-code:sonnet", Builtins())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].Agent != "codex" || got[0].Model != "gpt-5" {
		t.Fatalf("unexpected first entry: %+v", got[0])
	}
}

func TestParseRotationRejectsBadShape(t *testing.T) {
	if _, err := ParseRotation("codex", Builtins()); err == nil {
		t.Fatal("expected malformed entry to fail")
	}
}

func TestParseRotationRejectsUnknownAgent(t *testing.T) {
	if _, err := ParseRotation("unknown:model", Builtins()); err == nil {
		t.Fatal("expected unknown agent to fail")
	}
}

func TestFormatRotation(t *testing.T) {
	got := FormatRotation([]RotationEntry{{Agent: "codex", Model: "gpt-5"}, {Agent: "claude-code", Model: "sonnet"}})
	if len(got) != 2 || got[0] != "codex:gpt-5" || got[1] != "claude-code:sonnet" {
		t.Fatalf("unexpected formatted rotation: %v", got)
	}
}

func TestCurrentRotationEntryFallsBackAndWraps(t *testing.T) {
	got := CurrentRotationEntry([]string{"codex:gpt-5", "claude-code:sonnet"}, 3, "copilot", "fallback")
	if got.Agent != "claude-code" || got.Model != "sonnet" {
		t.Fatalf("unexpected rotation entry: %+v", got)
	}
	fallback := CurrentRotationEntry(nil, 0, "copilot", "fallback")
	if fallback.Agent != "copilot" || fallback.Model != "fallback" {
		t.Fatalf("unexpected fallback entry: %+v", fallback)
	}
}

func TestResolveCommandUsesEnvOverride(t *testing.T) {
	def := Builtins()["codex"]
	got := ResolveCommand(def, map[string]string{"BU80_CODEX_BINARY": "/tmp/codex-custom"})
	if got != "/tmp/codex-custom" {
		t.Fatalf("expected env override, got %q", got)
	}
}

func TestBuildArgsForCodex(t *testing.T) {
	def := Builtins()["codex"]
	got, err := BuildArgs(def, InvocationOptions{
		Prompt:    "fix tests",
		Model:     "gpt-5",
		AllowAll:  true,
		ExtraArgs: []string{"--sandbox", "workspace-write"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"exec", "--model", "gpt-5", "--full-auto", "--sandbox", "workspace-write", "fix tests"}
	assertArgs(t, got, want)
}

func TestBuildArgsForOpenCode(t *testing.T) {
	def := Builtins()["opencode"]
	got, err := BuildArgs(def, InvocationOptions{Prompt: "ship it", Model: "gpt-5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"run", "--model", "gpt-5", "ship it"}
	assertArgs(t, got, want)
}

func TestBuildArgsForClaudeCode(t *testing.T) {
	def := Builtins()["claude-code"]
	got, err := BuildArgs(def, InvocationOptions{Prompt: "review", Model: "sonnet", ExtraArgs: []string{"--dangerously-skip-permissions"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"-p", "review", "--model", "sonnet", "--dangerously-skip-permissions"}
	assertArgs(t, got, want)
}

func TestBuildArgsRequiresPrompt(t *testing.T) {
	def := Builtins()["copilot"]
	if _, err := BuildArgs(def, InvocationOptions{}); err == nil {
		t.Fatal("expected missing prompt to fail")
	}
}

func assertArgs(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected arg count: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected args: got=%v want=%v", got, want)
		}
	}
}
