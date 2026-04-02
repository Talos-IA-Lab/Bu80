package gitutil

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type Snapshot map[string]string

func CaptureSnapshot() (Snapshot, error) {
	tracked, err := gitLines("ls-files")
	if err != nil {
		if isNotGitRepo(err) {
			return Snapshot{}, nil
		}
		return nil, err
	}
	if len(tracked) == 0 {
		return Snapshot{}, nil
	}

	args := append([]string{"hash-object"}, tracked...)
	hashes, err := gitLines(args...)
	if err != nil {
		return nil, err
	}
	if len(hashes) != len(tracked) {
		return nil, fmt.Errorf("git hash-object returned %d hashes for %d files", len(hashes), len(tracked))
	}

	snapshot := make(Snapshot, len(tracked))
	for i, file := range tracked {
		snapshot[file] = hashes[i]
	}
	return snapshot, nil
}

func DetectModifiedFiles(before Snapshot) ([]string, error) {
	after, err := CaptureSnapshot()
	if err != nil {
		return nil, err
	}
	return DiffSnapshots(before, after), nil
}

func DiffSnapshots(before Snapshot, after Snapshot) []string {
	seen := make(map[string]struct{})
	var changed []string
	for file, hash := range before {
		seen[file] = struct{}{}
		if after[file] != hash {
			changed = append(changed, file)
		}
	}
	for file, hash := range after {
		if _, ok := seen[file]; ok {
			continue
		}
		if hash != "" {
			changed = append(changed, file)
		}
	}
	sort.Strings(changed)
	return changed
}

func AutoCommit(iteration int, modified []string, completed bool) error {
	status, err := gitLines("status", "--porcelain")
	if err != nil {
		if isNotGitRepo(err) {
			return nil
		}
		return err
	}
	if len(status) == 0 {
		return nil
	}
	if _, err := gitOutput("add", "-A"); err != nil {
		return err
	}

	msg := fmt.Sprintf("iteration %d: work in progress", iteration)
	if len(modified) > 0 {
		msg = fmt.Sprintf("iteration %d: updated %s", iteration, strings.Join(modified, ", "))
	}
	if completed {
		msg = "completion: " + msg
	} else {
		msg = msg + " (incomplete)"
	}

	_, err = gitOutput("commit", "-m", msg)
	return err
}

func gitLines(args ...string) ([]string, error) {
	out, err := gitOutput(args...)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func gitOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, errors.New(strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return out, nil
}

func isNotGitRepo(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "not a git repository")
}
