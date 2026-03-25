package tasks

import (
	"fmt"
	"strings"
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in-progress"
	StatusComplete   Status = "complete"
)

type Task struct {
	Title    string
	Status   Status
	Subtasks []Task
}

func Parse(markdown string) []Task {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	var result []Task
	currentTop := -1

	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")

		if task, ok := parseTaskLine(line, false); ok {
			result = append(result, task)
			currentTop = len(result) - 1
			continue
		}

		if task, ok := parseTaskLine(line, true); ok && currentTop >= 0 {
			result[currentTop].Subtasks = append(result[currentTop].Subtasks, task)
		}
	}

	return result
}

func AllComplete(markdown string) bool {
	tasks := Parse(markdown)
	if len(tasks) == 0 {
		return false
	}

	for _, task := range tasks {
		if task.Status != StatusComplete {
			return false
		}
		for _, subtask := range task.Subtasks {
			if subtask.Status != StatusComplete {
				return false
			}
		}
	}

	return true
}

func parseTaskLine(line string, allowIndent bool) (Task, bool) {
	if !allowIndent && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
		return Task{}, false
	}
	if allowIndent {
		trimmedLeft := strings.TrimLeft(line, " \t")
		if trimmedLeft == line {
			return Task{}, false
		}
		line = trimmedLeft
	}

	if !strings.HasPrefix(line, "- [") || len(line) < 6 {
		return Task{}, false
	}

	statusRune := line[3]
	if line[4] != ']' {
		return Task{}, false
	}

	title := strings.TrimSpace(line[5:])
	status, ok := mapStatus(statusRune)
	if !ok || title == "" {
		return Task{}, false
	}

	return Task{Title: title, Status: status}, true
}

func mapStatus(marker byte) (Status, bool) {
	switch marker {
	case ' ':
		return StatusTodo, true
	case '/':
		return StatusInProgress, true
	case 'x', 'X':
		return StatusComplete, true
	default:
		return "", false
	}
}

func Format(tasks []Task) string {
	if len(tasks) == 0 {
		return "# Bu80 Tasks\n"
	}

	var lines []string
	lines = append(lines, "# Bu80 Tasks")
	for _, task := range tasks {
		lines = append(lines, formatTask(task, ""))
		for _, subtask := range task.Subtasks {
			lines = append(lines, formatTask(subtask, "  "))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatTask(task Task, indent string) string {
	return fmt.Sprintf("%s- [%s] %s", indent, statusMarker(task.Status), task.Title)
}

func statusMarker(status Status) string {
	switch status {
	case StatusTodo:
		return " "
	case StatusInProgress:
		return "/"
	case StatusComplete:
		return "x"
	default:
		return " "
	}
}
