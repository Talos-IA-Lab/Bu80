package taskcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bu80/internal/state"
)

func TestListWithoutTaskFilePrintsMessage(t *testing.T) {
	inTempDir(t)
	var buf bytes.Buffer
	if err := List(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "No tasks found." {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestAddInitializesTaskFile(t *testing.T) {
	inTempDir(t)
	if err := Add("ship status command"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(state.TasksFile)
	if err != nil {
		t.Fatalf("expected task file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "# Bu80 Tasks") || !strings.Contains(text, "- [ ] ship status command") {
		t.Fatalf("unexpected file contents: %q", text)
	}
}

func TestRemoveDeletesTopLevelTaskAndIndentedLines(t *testing.T) {
	inTempDir(t)
	content := "# Bu80 Tasks\n- [ ] one\n  - [x] child\n- [ ] two\n"
	if err := state.WriteTasks(content); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := Remove(1); err != nil {
		t.Fatalf("unexpected remove error: %v", err)
	}
	data, err := os.ReadFile(state.TasksFile)
	if err != nil {
		t.Fatalf("expected task file: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "one") || strings.Contains(text, "child") || !strings.Contains(text, "two") {
		t.Fatalf("unexpected file contents after remove: %q", text)
	}
}

func TestRemoveRejectsOutOfRangeIndex(t *testing.T) {
	inTempDir(t)
	if err := state.WriteTasks("# Bu80 Tasks\n- [ ] one\n"); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if err := Remove(2); err == nil {
		t.Fatal("expected out-of-range error")
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
	if err := os.MkdirAll(filepath.Join(dir, state.DirName), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}
