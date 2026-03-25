package statuscmd

import (
	"strings"
	"testing"
	"time"

	"bu80/internal/history"
	"bu80/internal/state"
)

func TestRenderNoActiveLoop(t *testing.T) {
	got := Render(Snapshot{Now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)}, false)
	if !strings.Contains(got, "No active loop.") {
		t.Fatalf("expected no-active-loop message, got %q", got)
	}
}

func TestRenderActiveLoopIncludesSummary(t *testing.T) {
	snapshot := Snapshot{
		Loop: &state.LoopState{
			Active:            true,
			Iteration:         3,
			StartedAt:         time.Date(2026, 3, 25, 11, 55, 0, 0, time.UTC),
			CompletionPromise: "COMPLETE",
			Agent:             "codex",
			Model:             "gpt-5",
			Prompt:            "Implement feature and verify with tests.",
		},
		History: &history.History{
			Iterations: []history.IterationRecord{{Iteration: 2, Agent: "codex", ExitCode: 0}},
		},
		Context: "Remember to preserve task semantics.",
		Now:     time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
	}

	got := Render(snapshot, false)
	for _, want := range []string{"Active loop: yes", "Iteration: 3", "Agent: codex", "Pending context:", "Recent iterations:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got %q", want, got)
		}
	}
}

func TestRenderTasksSummary(t *testing.T) {
	snapshot := Snapshot{
		TasksRaw: "- [x] done\n- [/] current\n- [ ] next\n",
	}
	got := Render(snapshot, true)
	for _, want := range []string{"Tasks: total=3 complete=1 in-progress=1 todo=1", "1. [x] done", "2. [/] current", "3. [ ] next"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got %q", want, got)
		}
	}
}
