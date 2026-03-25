package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bu80/internal/state"
)

func TestResolveSourcePrefersPromptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(path, []byte("from file\n"), 0o644); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}
	got, err := ResolveSource(SourceOptions{PromptFile: path, PromptArgs: []string{"fallback text"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from file" {
		t.Fatalf("expected prompt from file, got %q", got)
	}
}

func TestResolveSourceReadsSinglePositionalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(path, []byte("from positional file"), 0o644); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}
	got, err := ResolveSource(SourceOptions{PromptArgs: []string{path}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from positional file" {
		t.Fatalf("expected positional prompt file contents, got %q", got)
	}
}

func TestResolveSourceJoinsPromptArgs(t *testing.T) {
	got, err := ResolveSource(SourceOptions{PromptArgs: []string{"build", "the", "feature"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "build the feature" {
		t.Fatalf("unexpected joined prompt: %q", got)
	}
}

func TestResolveSourceFallsBackToResumePrompt(t *testing.T) {
	got, err := ResolveSource(SourceOptions{Resume: &state.LoopState{Active: true, Prompt: "resume prompt"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "resume prompt" {
		t.Fatalf("expected resume prompt, got %q", got)
	}
}

func TestResolveSourceRejectsEmptyPromptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(path, []byte(" \n\t"), 0o644); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}
	if _, err := ResolveSource(SourceOptions{PromptFile: path}); err == nil {
		t.Fatal("expected empty prompt file to fail")
	}
}

func TestRenderTemplateReplacesKnownVariables(t *testing.T) {
	template := "{{iteration}}|{{max_iterations}}|{{min_iterations}}|{{prompt}}|{{completion_promise}}|{{abort_promise}}|{{task_promise}}|{{context}}|{{tasks}}"
	got := RenderTemplate(template, Data{
		Iteration:         2,
		MaxIterations:     7,
		MinIterations:     1,
		Prompt:            "do work",
		CompletionPromise: "COMPLETE",
		AbortPromise:      "ABORT",
		TaskPromise:       "NEXT",
		Context:           "ctx",
		Tasks:             "- [ ] one",
	})
	want := "2|7|1|do work|COMPLETE|ABORT|NEXT|ctx|- [ ] one"
	if got != want {
		t.Fatalf("unexpected rendered template: %q", got)
	}
}

func TestBuildDefaultIncludesContextAndPromises(t *testing.T) {
	got := BuildDefault(Data{
		Iteration:         3,
		MinIterations:     1,
		MaxIterations:     5,
		Prompt:            "ship the feature",
		CompletionPromise: "COMPLETE",
		AbortPromise:      "ABORT",
		Context:           "Existing constraint",
	})
	if !strings.Contains(got, "Primary objective:\nship the feature") {
		t.Fatalf("expected primary objective in prompt, got %q", got)
	}
	if !strings.Contains(got, "Completion promise: COMPLETE") {
		t.Fatalf("expected completion promise in prompt, got %q", got)
	}
	if !strings.Contains(got, "Additional context:\nExisting constraint") {
		t.Fatalf("expected context block in prompt, got %q", got)
	}
	if !strings.Contains(got, "Abort promise: ABORT") {
		t.Fatalf("expected abort promise in prompt, got %q", got)
	}
}

func TestBuildTasksIncludesTaskFileAndTaskPromise(t *testing.T) {
	got := BuildTasks(Data{
		Iteration:         2,
		Prompt:            "work task by task",
		CompletionPromise: "COMPLETE",
		TaskPromise:       "NEXT",
		Tasks:             "# Bu80 Tasks\n- [ ] one\n",
	})
	if !strings.Contains(got, "Task promise: NEXT") {
		t.Fatalf("expected task promise in prompt, got %q", got)
	}
	if !strings.Contains(got, "Current tasks (.bu80/bu80-tasks.md):\n# Bu80 Tasks\n- [ ] one") {
		t.Fatalf("expected tasks file contents in prompt, got %q", got)
	}
}

func TestBuildUsesTemplateWhenProvided(t *testing.T) {
	got := Build(Data{Iteration: 4, Prompt: "base", CompletionPromise: "DONE"}, false, "iter={{iteration}} prompt={{prompt}} done={{completion_promise}}")
	if got != "iter=4 prompt=base done=DONE" {
		t.Fatalf("unexpected built template: %q", got)
	}
}
