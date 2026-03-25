package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureSnapshotAndDetectModifiedFiles(t *testing.T) {
	inRepo(t)
	writeFile(t, "note.txt", "one\n")
	runGit(t, "add", "note.txt")
	runGit(t, "commit", "-m", "initial")

	before, err := CaptureSnapshot()
	if err != nil {
		t.Fatalf("capture snapshot: %v", err)
	}
	writeFile(t, "note.txt", "two\n")
	changed, err := DetectModifiedFiles(before)
	if err != nil {
		t.Fatalf("detect modified files: %v", err)
	}
	if len(changed) != 1 || changed[0] != "note.txt" {
		t.Fatalf("unexpected changed files: %v", changed)
	}
}

func TestAutoCommitCommitsWhenChangesExist(t *testing.T) {
	inRepo(t)
	writeFile(t, "note.txt", "one\n")
	runGit(t, "add", "note.txt")
	runGit(t, "commit", "-m", "initial")
	writeFile(t, "note.txt", "two\n")
	if err := AutoCommit(2); err != nil {
		t.Fatalf("auto commit: %v", err)
	}
	log := runGit(t, "log", "--oneline", "-n", "1")
	if !strings.Contains(log, "Bu80 iteration 2: work in progress") {
		t.Fatalf("unexpected log: %q", log)
	}
}

func inRepo(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	runGit(t, "init")
	runGit(t, "config", "user.email", "codex@example.com")
	runGit(t, "config", "user.name", "Codex")
}

func writeFile(t *testing.T, rel string, content string) {
	t.Helper()
	path := filepath.Join(".", rel)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func runGit(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}
