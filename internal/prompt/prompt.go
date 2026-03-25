package prompt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bu80/internal/state"
)

type Data struct {
	Iteration         int
	MaxIterations     int
	MinIterations     int
	Prompt            string
	CompletionPromise string
	AbortPromise      string
	TaskPromise       string
	Context           string
	Tasks             string
}

type SourceOptions struct {
	PromptFile string
	PromptArgs []string
	Resume     *state.LoopState
}

func RenderTemplate(template string, data Data) string {
	replacer := strings.NewReplacer(
		"{{iteration}}", strconv.Itoa(data.Iteration),
		"{{max_iterations}}", strconv.Itoa(data.MaxIterations),
		"{{min_iterations}}", strconv.Itoa(data.MinIterations),
		"{{prompt}}", data.Prompt,
		"{{completion_promise}}", data.CompletionPromise,
		"{{abort_promise}}", data.AbortPromise,
		"{{task_promise}}", data.TaskPromise,
		"{{context}}", data.Context,
		"{{tasks}}", data.Tasks,
	)
	return replacer.Replace(template)
}

func BuildDefault(data Data) string {
	var b strings.Builder
	b.WriteString("You are running inside Bu80, an iterative coding loop.\n\n")
	b.WriteString("Primary objective:\n")
	b.WriteString(strings.TrimSpace(data.Prompt))
	b.WriteString("\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Inspect the repository before making assumptions.\n")
	b.WriteString("- Make concrete progress this iteration and verify your work when feasible.\n")
	if strings.TrimSpace(data.Context) != "" {
		b.WriteString("- Use the provided context before asking the user to repeat it.\n")
	}
	b.WriteString("- Only output the completion promise when the objective is actually complete.\n")
	if strings.TrimSpace(data.AbortPromise) != "" {
		b.WriteString("- If you are blocked and cannot continue safely, output the abort promise exactly.\n")
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Iteration: %d\n", data.Iteration))
	if data.MaxIterations > 0 {
		b.WriteString(fmt.Sprintf("Iteration limits: min=%d max=%d\n", data.MinIterations, data.MaxIterations))
	} else {
		b.WriteString(fmt.Sprintf("Iteration limits: min=%d max=unbounded\n", data.MinIterations))
	}
	b.WriteString(fmt.Sprintf("Completion promise: %s\n", strings.TrimSpace(data.CompletionPromise)))
	if strings.TrimSpace(data.AbortPromise) != "" {
		b.WriteString(fmt.Sprintf("Abort promise: %s\n", strings.TrimSpace(data.AbortPromise)))
	}
	if strings.TrimSpace(data.Context) != "" {
		b.WriteString("\nAdditional context:\n")
		b.WriteString(strings.TrimSpace(data.Context))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func BuildTasks(data Data) string {
	var b strings.Builder
	b.WriteString("You are running inside Bu80 tasks mode. Work through .bu80/bu80-tasks.md one task at a time.\n\n")
	b.WriteString("Primary objective:\n")
	b.WriteString(strings.TrimSpace(data.Prompt))
	b.WriteString("\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Choose one incomplete task and finish it before moving on.\n")
	b.WriteString("- Update the task file as you work, using [/] for in progress and [x] for complete.\n")
	b.WriteString("- Output the task promise exactly when the current task is complete and more tasks remain.\n")
	b.WriteString("- Output the completion promise only when every task is complete.\n")
	if strings.TrimSpace(data.AbortPromise) != "" {
		b.WriteString("- If you are blocked and cannot continue safely, output the abort promise exactly.\n")
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Iteration: %d\n", data.Iteration))
	b.WriteString(fmt.Sprintf("Task promise: %s\n", strings.TrimSpace(data.TaskPromise)))
	b.WriteString(fmt.Sprintf("Completion promise: %s\n", strings.TrimSpace(data.CompletionPromise)))
	if strings.TrimSpace(data.AbortPromise) != "" {
		b.WriteString(fmt.Sprintf("Abort promise: %s\n", strings.TrimSpace(data.AbortPromise)))
	}
	if strings.TrimSpace(data.Context) != "" {
		b.WriteString("\nAdditional context:\n")
		b.WriteString(strings.TrimSpace(data.Context))
		b.WriteString("\n")
	}
	b.WriteString("\nCurrent tasks (.bu80/bu80-tasks.md):\n")
	if strings.TrimSpace(data.Tasks) == "" {
		b.WriteString("# Bu80 Tasks\n")
	} else {
		b.WriteString(strings.TrimRight(data.Tasks, "\n"))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func Build(data Data, tasksMode bool, template string) string {
	if strings.TrimSpace(template) != "" {
		return strings.TrimSpace(RenderTemplate(template, data))
	}
	if tasksMode {
		return BuildTasks(data)
	}
	return BuildDefault(data)
}

func ResolveSource(opts SourceOptions) (string, error) {
	if strings.TrimSpace(opts.PromptFile) != "" {
		return readPromptFile(opts.PromptFile)
	}

	if len(opts.PromptArgs) == 1 {
		arg := opts.PromptArgs[0]
		if pathLooksLikeExistingFile(arg) {
			return readPromptFile(arg)
		}
	}

	if len(opts.PromptArgs) > 0 {
		joined := strings.TrimSpace(strings.Join(opts.PromptArgs, " "))
		if joined != "" {
			return joined, nil
		}
	}

	if opts.Resume != nil && opts.Resume.Active {
		prompt := strings.TrimSpace(opts.Resume.Prompt)
		if prompt != "" {
			return prompt, nil
		}
	}

	return "", errors.New("prompt is required")
}

func readPromptFile(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	info, err := os.Stat(cleanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("prompt file does not exist: %s", path)
		}
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("prompt path is not a file: %s", path)
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", err
	}
	prompt := strings.TrimSpace(string(data))
	if prompt == "" {
		return "", fmt.Errorf("prompt file is empty: %s", path)
	}
	return prompt, nil
}

func pathLooksLikeExistingFile(path string) bool {
	info, err := os.Stat(filepath.Clean(path))
	return err == nil && !info.IsDir()
}
