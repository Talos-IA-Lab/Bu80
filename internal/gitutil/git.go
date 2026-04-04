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

const deletedEntryHash = "<deleted>"

func CaptureSnapshot() (Snapshot, error) {
	tracked, err := gitNulFields("ls-files", "-z")
	if err != nil {
		if isNotGitRepo(err) {
			return Snapshot{}, nil
		}
		return nil, err
	}
	deleted, err := gitNulFields("ls-files", "--deleted", "-z")
	if err != nil {
		return nil, err
	}
	untracked, err := gitNulFields("ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}

	if len(tracked) == 0 && len(untracked) == 0 {
		return Snapshot{}, nil
	}

	deletedSet := make(map[string]struct{}, len(deleted))
	for _, file := range deleted {
		deletedSet[file] = struct{}{}
	}

	existingTracked := make([]string, 0, len(tracked))
	for _, file := range tracked {
		if _, isDeleted := deletedSet[file]; isDeleted {
			continue
		}
		existingTracked = append(existingTracked, file)
	}

	snapshot := make(Snapshot, len(existingTracked)+len(deleted)+len(untracked))
	if err := appendHashes(snapshot, existingTracked); err != nil {
		return nil, err
	}
	for _, file := range deleted {
		snapshot[file] = deletedEntryHash
	}
	if err := appendHashes(snapshot, untracked); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func appendHashes(snapshot Snapshot, files []string) error {
	if len(files) == 0 {
		return nil
	}

	args := append([]string{"hash-object", "--"}, files...)
	hashes, err := gitLines(args...)
	if err != nil {
		return err
	}
	if len(hashes) != len(files) {
		return fmt.Errorf("git hash-object returned %d hashes for %d files", len(hashes), len(files))
	}
	for i, file := range files {
		snapshot[file] = hashes[i]
	}
	return nil
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

func gitNulFields(args ...string) ([]string, error) {
	out, err := gitOutput(args...)
	if err != nil {
		return nil, err
	}
	out = bytes.TrimSuffix(out, []byte{0})
	if len(out) == 0 {
		return nil, nil
	}
	return strings.Split(string(out), "\x00"), nil
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
