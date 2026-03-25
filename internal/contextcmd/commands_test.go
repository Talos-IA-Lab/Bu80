package contextcmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"bu80/internal/state"
)

func TestAddAppendsTimestampedSection(t *testing.T) {
	inTempDir(t)
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	if err := Add("Preserve task semantics.", now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(state.ContextFile)
	if err != nil {
		t.Fatalf("expected context file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "## 2026-03-25T12:00:00Z") || !strings.Contains(text, "Preserve task semantics.") {
		t.Fatalf("unexpected context file contents: %q", text)
	}
}

func TestClearWithoutFilePrintsMessage(t *testing.T) {
	inTempDir(t)
	var buf bytes.Buffer
	if err := Clear(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "No context to clear." {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestClearDeletesExistingFile(t *testing.T) {
	inTempDir(t)
	if err := state.WriteContext("hello"); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	var buf bytes.Buffer
	if err := Clear(&buf); err != nil {
		t.Fatalf("unexpected clear error: %v", err)
	}
	if _, err := os.Stat(state.ContextFile); !os.IsNotExist(err) {
		t.Fatalf("expected context file to be removed, err=%v", err)
	}
}

func inTempDir(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}
