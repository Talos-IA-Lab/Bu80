package cli

import (
	"os"
	"strings"
	"testing"
	"bu80/internal/config"
)

func TestParseValidatesIterationBounds(t *testing.T) {
	_, _, err := Parse([]string{"--min-iterations", "3", "--max-iterations", "2", "prompt"})
	if err == nil {
		t.Fatal("expected invalid iteration bounds to fail")
	}
}

func TestParseValidatesDistinctTaskPromise(t *testing.T) {
	_, _, err := Parse([]string{
		"--tasks",
		"--completion-promise", "DONE",
		"--task-promise", "DONE",
		"prompt",
	})
	if err == nil {
		t.Fatal("expected equal promises in tasks mode to fail")
	}
}

func TestParseKeepsArgsAfterDoubleDash(t *testing.T) {
	opts, _, err := Parse([]string{"prompt", "--", "--sandbox", "danger-full-access"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts.ExtraArgs) != 2 {
		t.Fatalf("expected 2 extra args, got %d", len(opts.ExtraArgs))
	}
}

func TestParseQuestionsDisabledByFlag(t *testing.T) {
	opts, _, err := Parse([]string{"--no-questions", "prompt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoQuestions {
		t.Fatal("expected no-questions flag to be set")
	}
}

func TestRunInitConfigWritesDefaultFile(t *testing.T) {
	oldStdout := stdout
	stdout = os.Stdout
	defer func() { stdout = oldStdout }()

	dir := t.TempDir()
	path := dir + "/agents.json"
	if err := Run([]string{"--init-config", "--config", path}); err != nil {
		t.Fatalf("run init-config: %v", err)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read generated config: %v", readErr)
	}
	if !strings.Contains(string(data), "\"opencode\"") {
		t.Fatalf("unexpected generated config: %s", string(data))
	}
	if !strings.Contains(string(data), "\"questions_enabled\": true") {
		t.Fatalf("expected questions_enabled in default config: %s", string(data))
	}
}

func TestConfigLoadQuestionsEnabled(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.json"
	if err := os.WriteFile(path, []byte(`{"questions_enabled": false}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.QuestionsEnabled == nil || *cfg.QuestionsEnabled != false {
		t.Fatalf("expected questions_enabled to be false, got %v", cfg.QuestionsEnabled)
	}
}
