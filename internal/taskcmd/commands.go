package taskcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"bu80/internal/state"
	"bu80/internal/tasks"
)

func List(w io.Writer) error {
	content, err := state.ReadTasks()
	if err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		_, err := fmt.Fprintln(w, "No tasks found.")
		return err
	}

	parsed := tasks.Parse(content)
	if len(parsed) == 0 {
		_, err := fmt.Fprintln(w, "No tasks found.")
		return err
	}

	for idx, task := range parsed {
		if _, err := fmt.Fprintf(w, "%d. [%s] %s\n", idx+1, statusLabel(task.Status), task.Title); err != nil {
			return err
		}
	}
	return nil
}

func Add(description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return errors.New("task description is required")
	}

	content, err := state.ReadTasks()
	if err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		content = "# Bu80 Tasks\n"
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += fmt.Sprintf("- [ ] %s\n", description)
	return state.WriteTasks(content)
}

func Remove(index int) error {
	if index <= 0 {
		return fmt.Errorf("task index must be at least 1")
	}

	content, err := state.ReadTasks()
	if err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("task index %d out of range", index)
	}

	updated, removed, err := removeTopLevelTask(content, index)
	if err != nil {
		return err
	}
	if !removed {
		return fmt.Errorf("task index %d out of range", index)
	}
	return state.WriteTasks(updated)
}

func removeTopLevelTask(content string, index int) (string, bool, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	current := 0
	start := -1
	end := -1

	for i, line := range lines {
		if isTopLevelTaskLine(line) {
			current++
			if current == index {
				start = i
				continue
			}
			if start >= 0 {
				end = i
				break
			}
		}
	}

	if start < 0 {
		return "", false, nil
	}
	if end < 0 {
		end = len(lines)
	}

	kept := append([]string(nil), lines[:start]...)
	kept = append(kept, lines[end:]...)
	result := strings.Join(trimExtraEmptyLines(kept), "\n")
	if strings.TrimSpace(result) == "" {
		result = "# Bu80 Tasks\n"
	} else if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result, true, nil
}

func isTopLevelTaskLine(line string) bool {
	trimmed := strings.TrimRight(line, " \t")
	return strings.HasPrefix(trimmed, "- [")
}

func trimExtraEmptyLines(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func statusLabel(status tasks.Status) string {
	switch status {
	case tasks.StatusComplete:
		return "x"
	case tasks.StatusInProgress:
		return "/"
	default:
		return " "
	}
}

func Exists() bool {
	_, err := os.Stat(state.TasksFile)
	return err == nil
}
