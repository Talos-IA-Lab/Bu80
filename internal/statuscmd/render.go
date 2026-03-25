package statuscmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"bu80/internal/history"
	"bu80/internal/state"
	"bu80/internal/tasks"
)

type Snapshot struct {
	Loop      *state.LoopState
	History   *history.History
	Context   string
	TasksRaw  string
	Questions *state.QuestionsFile
	Now       time.Time
}

func LoadSnapshot(now time.Time) (Snapshot, error) {
	loop, err := state.LoadLoopState()
	if err != nil {
		return Snapshot{}, err
	}
	hist, err := state.LoadHistory()
	if err != nil {
		return Snapshot{}, err
	}
	contextText, err := state.ReadContext()
	if err != nil {
		return Snapshot{}, err
	}
	tasksRaw, err := state.ReadTasks()
	if err != nil {
		return Snapshot{}, err
	}
	questions, err := state.LoadQuestions()
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Loop:      loop,
		History:   hist,
		Context:   contextText,
		TasksRaw:  tasksRaw,
		Questions: questions,
		Now:       now,
	}, nil
}

func Write(w io.Writer, snapshot Snapshot, showTasks bool) error {
	_, err := io.WriteString(w, Render(snapshot, showTasks))
	return err
}

func Render(snapshot Snapshot, showTasks bool) string {
	var b strings.Builder

	if snapshot.Loop == nil || !snapshot.Loop.Active {
		b.WriteString("No active loop.\n")
	} else {
		loop := snapshot.Loop
		b.WriteString("Active loop: yes\n")
		b.WriteString(fmt.Sprintf("Iteration: %d\n", loop.Iteration))
		b.WriteString(fmt.Sprintf("Started: %s\n", loop.StartedAt.Format(time.RFC3339)))
		if !loop.StartedAt.IsZero() && !snapshot.Now.IsZero() {
			b.WriteString(fmt.Sprintf("Elapsed: %s\n", snapshot.Now.Sub(loop.StartedAt).Round(time.Second)))
		}
		b.WriteString(fmt.Sprintf("Completion promise: %s\n", loop.CompletionPromise))
		b.WriteString(fmt.Sprintf("Agent: %s\n", loop.Agent))
		if loop.Model != "" {
			b.WriteString(fmt.Sprintf("Model: %s\n", loop.Model))
		}
		if len(loop.Rotation) > 0 {
			b.WriteString(fmt.Sprintf("Rotation: %d entries, index %d\n", len(loop.Rotation), loop.RotationIndex))
		}
		if prompt := strings.TrimSpace(loop.Prompt); prompt != "" {
			b.WriteString(fmt.Sprintf("Prompt preview: %s\n", preview(prompt, 120)))
		}
	}

	if strings.TrimSpace(snapshot.Context) != "" {
		b.WriteString(fmt.Sprintf("Pending context: %s\n", preview(snapshot.Context, 120)))
	}

	if snapshot.Questions != nil {
		pending := 0
		for _, record := range snapshot.Questions.Records {
			if record.Pending {
				pending++
			}
		}
		if pending > 0 {
			b.WriteString(fmt.Sprintf("Pending questions: %d\n", pending))
		}
	}

	if snapshot.History != nil {
		if len(snapshot.History.Iterations) > 0 {
			b.WriteString("Recent iterations:\n")
			start := 0
			if len(snapshot.History.Iterations) > 5 {
				start = len(snapshot.History.Iterations) - 5
			}
			for _, iteration := range snapshot.History.Iterations[start:] {
				b.WriteString(fmt.Sprintf("- #%d agent=%s exit=%d files=%d tools=%d completion=%t\n",
					iteration.Iteration,
					iteration.Agent,
					iteration.ExitCode,
					len(iteration.FilesModified),
					len(iteration.ToolsUsed),
					iteration.CompletionDetected,
				))
			}
		}
		indicators := snapshot.History.StruggleIndicators
		if indicators.RepeatedErrors > 0 || indicators.NoProgressIters > 0 || indicators.ShortIterations > 0 {
			b.WriteString(fmt.Sprintf("Struggle indicators: repeatedErrors=%d noProgressIterations=%d shortIterations=%d\n",
				indicators.RepeatedErrors,
				indicators.NoProgressIters,
				indicators.ShortIterations,
			))
		}
	}

	if showTasks || (snapshot.Loop != nil && snapshot.Loop.TasksMode) {
		parsed := tasks.Parse(snapshot.TasksRaw)
		complete, todo, progress := summarizeTasks(parsed)
		b.WriteString(fmt.Sprintf("Tasks: total=%d complete=%d in-progress=%d todo=%d\n", len(parsed), complete, progress, todo))
		if len(parsed) == 0 {
			b.WriteString("No tasks found.\n")
		} else {
			for idx, task := range parsed {
				b.WriteString(fmt.Sprintf("%d. [%s] %s\n", idx+1, statusLabel(task.Status), task.Title))
			}
		}
	}

	return b.String()
}

func summarizeTasks(parsed []tasks.Task) (complete int, todo int, progress int) {
	for _, task := range parsed {
		switch task.Status {
		case tasks.StatusComplete:
			complete++
		case tasks.StatusInProgress:
			progress++
		default:
			todo++
		}
	}
	return complete, todo, progress
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

func preview(text string, limit int) string {
	trimmed := strings.Join(strings.Fields(text), " ")
	if len(trimmed) <= limit {
		return trimmed
	}
	if limit <= 3 {
		return trimmed[:limit]
	}
	return trimmed[:limit-3] + "..."
}
