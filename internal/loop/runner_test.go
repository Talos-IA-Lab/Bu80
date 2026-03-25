package loop

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bu80/internal/state"
)

func TestRunCompletesWhenPromiseDetected(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho working\necho '<promise>COMPLETE</promise>'\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"do work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     3,
		Stdout:            &stdout,
		Stderr:            &stderr,
		Now:               fixedNow(),
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Completion detected.") {
		t.Fatalf("expected completion output, got %q", stdout.String())
	}
	if _, err := os.Stat(state.LoopStateFile); !os.IsNotExist(err) {
		t.Fatalf("expected state to be cleared, err=%v", err)
	}
}

func TestRunStopsAtMaxIterationsAndPreservesHistory(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho still-running\n")
	var stdout bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"do work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Now:               fixedNow(),
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Reached max iterations: 2") {
		t.Fatalf("expected max-iterations message, got %q", stdout.String())
	}
	if _, err := os.Stat(state.HistoryFile); err != nil {
		t.Fatalf("expected history to remain, err=%v", err)
	}
}

func TestRunContinuesAfterNonZeroExit(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\ncount_file=.count\ncount=0\nif [ -f \"$count_file\" ]; then count=$(cat \"$count_file\"); fi\ncount=$((count + 1))\necho \"$count\" > \"$count_file\"\nif [ \"$count\" -eq 1 ]; then echo 'error: failed once'; exit 2; fi\necho '<promise>COMPLETE</promise>'\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"do work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     3,
		Stdout:            &stdout,
		Stderr:            &stderr,
		Now:               fixedNow(),
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Agent exited with code 2") {
		t.Fatalf("expected warning for non-zero exit, got %q", stderr.String())
	}
}

func TestRunRecordsModifiedFiles(t *testing.T) {
	inRepo(t)
	writeFile(t, "note.txt", "before\n")
	runGit(t, "add", "note.txt")
	runGit(t, "commit", "-m", "initial")
	script := writeScript(t, "#!/bin/sh\necho after > note.txt\necho '<promise>COMPLETE</promise>'\n")
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"do work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     2,
		MaxIterations:     2,
		NoCommit:          true,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		Now:               fixedNow(),
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	histBytes, readErr := os.ReadFile(state.HistoryFile)
	if readErr == nil {
		t.Fatalf("expected history to be cleared on completion, found %q", string(histBytes))
	}
	log := runGit(t, "log", "--oneline", "-n", "2")
	if strings.Contains(log, "Bu80 iteration") {
		t.Fatalf("did not expect auto-commit when no-commit is true, log=%q", log)
	}
}

func TestRunAutoCommitBestEffort(t *testing.T) {
	inRepo(t)
	writeFile(t, "note.txt", "before\n")
	runGit(t, "add", "note.txt")
	runGit(t, "commit", "-m", "initial")
	script := writeScript(t, "#!/bin/sh\necho after > note.txt\necho '<promise>COMPLETE</promise>'\n")
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"do work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     2,
		MaxIterations:     2,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		Now:               fixedNow(),
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	log := runGit(t, "log", "--oneline", "-n", "2")
	if !strings.Contains(log, "Bu80 iteration 1: work in progress") {
		t.Fatalf("expected auto-commit in log, got %q", log)
	}
}

func TestRunStreamingHeartbeat(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\nsleep 2\necho '<promise>COMPLETE</promise>'\n")
	var stdout bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"do work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stream:            true,
		HeartbeatInterval: 500 * time.Millisecond,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script, "NODE_ENV", "test"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "[bu80] still working...") {
		t.Fatalf("expected heartbeat output, got %q", stdout.String())
	}
}

func TestRunAbortClearsHistory(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho '<promise>ABORT</promise>'\n")
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"do work"},
		CompletionPromise: "COMPLETE",
		AbortPromise:      "ABORT",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script),
	})
	if err == nil {
		t.Fatal("expected abort error")
	}
	if _, statErr := os.Stat(state.HistoryFile); !os.IsNotExist(statErr) {
		t.Fatalf("expected history to be cleared on abort, err=%v", statErr)
	}
}

func TestRunTaskPromiseDoesNotFinishLoop(t *testing.T) {
	inRepo(t)
	if err := state.WriteTasks("# Bu80 Tasks\n- [ ] one\n"); err != nil {
		t.Fatalf("write tasks: %v", err)
	}
	script := writeScript(t, "#!/bin/sh\necho '<promise>READY_FOR_NEXT_TASK</promise>'\n")
	var stdout bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"do work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		TasksMode:         true,
		MinIterations:     1,
		MaxIterations:     1,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Task promise detected.") {
		t.Fatalf("expected task promise message, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "Completion detected.") {
		t.Fatalf("did not expect final completion, got %q", stdout.String())
	}
}

func TestRunInterruptClearsStateAndStopsHeartbeat(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\nsleep 5\necho late\n")
	interrupts := make(chan os.Signal, 2)
	var stdout bytes.Buffer
	resultCh := make(chan error, 1)
	go func() {
		_, err := RunWithResult(Options{
			Agent:             "codex",
			PromptArgs:        []string{"do work"},
			CompletionPromise: "COMPLETE",
			TaskPromise:       state.DefaultTaskPromise,
			MinIterations:     1,
			MaxIterations:     2,
			Stream:            true,
			HeartbeatInterval: 200 * time.Millisecond,
			Stdout:            &stdout,
			Stderr:            &bytes.Buffer{},
			Env:               envWithScript(script),
			Interrupts:        interrupts,
			ForceExit:         func(int) {},
		})
		resultCh <- err
	}()
	time.Sleep(700 * time.Millisecond)
	interrupts <- os.Interrupt
	if err := <-resultCh; err != nil {
		t.Fatalf("expected graceful interrupt, got %v", err)
	}
	if _, err := os.Stat(state.LoopStateFile); !os.IsNotExist(err) {
		t.Fatalf("expected state to be cleared, err=%v", err)
	}
	before := stdout.String()
	time.Sleep(500 * time.Millisecond)
	after := stdout.String()
	if before != after {
		t.Fatalf("expected heartbeat to stop after interrupt, before=%q after=%q", before, after)
	}
}

func TestRunSecondInterruptTriggersForceExitHook(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\nsleep 5\n")
	interrupts := make(chan os.Signal, 2)
	forced := make(chan int, 1)
	go func() {
		_, _ = RunWithResult(Options{
			Agent:             "codex",
			PromptArgs:        []string{"do work"},
			CompletionPromise: "COMPLETE",
			TaskPromise:       state.DefaultTaskPromise,
			MinIterations:     1,
			MaxIterations:     2,
			Stream:            true,
			HeartbeatInterval: 200 * time.Millisecond,
			Stdout:            &bytes.Buffer{},
			Stderr:            &bytes.Buffer{},
			Env:               envWithScript(script),
			Interrupts:        interrupts,
			ForceExit:         func(code int) { forced <- code },
		})
	}()
	time.Sleep(200 * time.Millisecond)
	interrupts <- os.Interrupt
	interrupts <- os.Interrupt
	select {
	case code := <-forced:
		if code != 1 {
			t.Fatalf("expected force exit code 1, got %d", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected force exit hook to be called")
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

func writeScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-agent.sh")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func writeFile(t *testing.T, rel string, content string) {
	t.Helper()
	if err := os.WriteFile(rel, []byte(content), 0o644); err != nil {
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

func envWithScript(script string, extra ...string) map[string]string {
	env := map[string]string{"PATH": os.Getenv("PATH"), "RALPH_CODEX_BINARY": script}
	for i := 0; i+1 < len(extra); i += 2 {
		env[extra[i]] = extra[i+1]
	}
	return env
}

func fixedNow() func() time.Time {
	current := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return func() time.Time {
		current = current.Add(2 * time.Second)
		return current
	}
}

func TestRunBuildsIterationPromptWithContext(t *testing.T) {
	inRepo(t)
	if err := state.WriteContext("Be careful with the config format"); err != nil {
		t.Fatalf("write context: %v", err)
	}
	script := writeScript(t, "#!/bin/sh\nfor last do :; done\nprintf '%s' \"$last\" > prompt.txt\necho '<promise>COMPLETE</promise>'\n")
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"ship the feature"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	promptText, readErr := os.ReadFile("prompt.txt")
	if readErr != nil {
		t.Fatalf("read prompt: %v", readErr)
	}
	got := string(promptText)
	if !strings.Contains(got, "Primary objective:\nship the feature") {
		t.Fatalf("expected base prompt in iteration prompt, got %q", got)
	}
	if !strings.Contains(got, "Additional context:\nBe careful with the config format") {
		t.Fatalf("expected persisted context in iteration prompt, got %q", got)
	}
	if !strings.Contains(got, "Completion promise: COMPLETE") {
		t.Fatalf("expected completion promise in iteration prompt, got %q", got)
	}
}

func TestRunRotatesAgentAndModelAcrossIterations(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\ncount_file=.count\ncount=0\nif [ -f \"$count_file\" ]; then count=$(cat \"$count_file\"); fi\ncount=$((count + 1))\necho \"$count\" > \"$count_file\"\nprintf '%s\n' \"$@\" >> args.log\necho '---' >> args.log\nif [ \"$count\" -eq 2 ]; then echo '<promise>COMPLETE</promise>'; fi\n")
	_, err := RunWithResult(Options{
		Agent:             "codex",
		Rotation:          "codex:alpha,claude-code:beta",
		PromptArgs:        []string{"rotate work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     3,
		MaxIterations:     2,
		NoCommit:          true,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		Env: map[string]string{
			"PATH":                os.Getenv("PATH"),
			"RALPH_CODEX_BINARY":  script,
			"RALPH_CLAUDE_BINARY": script,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	argsLog, readErr := os.ReadFile("args.log")
	if readErr != nil {
		t.Fatalf("read args log: %v", readErr)
	}
	got := string(argsLog)
	if !strings.Contains(got, "--model\nalpha\n") {
		t.Fatalf("expected first rotation model in args log, got %q", got)
	}
	if !strings.Contains(got, "--model\nbeta\n") {
		t.Fatalf("expected second rotation model in args log, got %q", got)
	}
	if !strings.Contains(got, "exec\n") {
		t.Fatalf("expected codex invocation in args log, got %q", got)
	}
	if !strings.Contains(got, "-p\n") {
		t.Fatalf("expected claude-code invocation in args log, got %q", got)
	}
}

func TestRunQuestionToolPersistsPendingQuestion(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho 'tool question: Which config path should I use?'\n")
	var stdout bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"ask for config"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		QuestionsEnabled:  true,
		MinIterations:     1,
		MaxIterations:     2,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script),
	})
	if err == nil {
		t.Fatal("expected question-handling run to pause with an error")
	}
	if !strings.Contains(stdout.String(), "Question requested: Which config path should I use?") {
		t.Fatalf("expected question notice, got %q", stdout.String())
	}
	questions, loadErr := state.LoadQuestions()
	if loadErr != nil {
		t.Fatalf("load questions: %v", loadErr)
	}
	if questions == nil || len(questions.Records) != 1 || !questions.Records[0].Pending {
		t.Fatalf("expected one pending question, got %+v", questions)
	}
}

func TestRunInjectsAnsweredQuestionsIntoContext(t *testing.T) {
	inRepo(t)
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	if err := state.SaveAnswer("Which config path should I use?", ".bu80/custom.json", now); err != nil {
		t.Fatalf("save answer: %v", err)
	}
	script := writeScript(t, "#!/bin/sh\nfor last do :; done\nprintf '%s' \"$last\" > prompt.txt\necho still-running\n")
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"continue the work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		QuestionsEnabled:  true,
		MinIterations:     2,
		MaxIterations:     1,
		NoCommit:          true,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	promptText, readErr := os.ReadFile("prompt.txt")
	if readErr != nil {
		t.Fatalf("read prompt: %v", readErr)
	}
	got := string(promptText)
	if !strings.Contains(got, "Answered questions:\nQ: Which config path should I use?\nA: .bu80/custom.json") {
		t.Fatalf("expected answered-question block in prompt, got %q", got)
	}
	contextText, contextErr := state.ReadContext()
	if contextErr != nil {
		t.Fatalf("read context: %v", contextErr)
	}
	if !strings.Contains(contextText, "Answered questions:\nQ: Which config path should I use?\nA: .bu80/custom.json") {
		t.Fatalf("expected answered-question block in persisted context, got %q", contextText)
	}
	questions, loadErr := state.LoadQuestions()
	if loadErr != nil {
		t.Fatalf("load questions: %v", loadErr)
	}
	if questions != nil {
		t.Fatalf("expected answered question queue to be consumed, got %+v", questions)
	}
}

func TestRunRecordsToolsUsedInHistory(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho 'tool edit: changed file'\necho '{\"tool\":\"question\",\"question\":\"Which path?\"}'\n")
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"collect tools"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		QuestionsEnabled:  false,
		MinIterations:     2,
		MaxIterations:     1,
		NoCommit:          true,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hist, loadErr := state.LoadHistory()
	if loadErr != nil {
		t.Fatalf("load history: %v", loadErr)
	}
	if hist == nil || len(hist.Iterations) != 1 {
		t.Fatalf("expected one history record, got %+v", hist)
	}
	got := hist.Iterations[0].ToolsUsed
	if len(got) != 2 || got[0] != "edit" || got[1] != "question" {
		t.Fatalf("unexpected tools used: %v", got)
	}
}

func TestRunStreamingShowsCompactToolSummary(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho 'tool edit: changed file'\necho 'tool question: Which path?'\necho '<promise>COMPLETE</promise>'\n")
	var stdout bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"stream tools"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stream:            true,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "[bu80] tools: edit=1") || !strings.Contains(out, "question=1") {
		t.Fatalf("expected compact tool summary in output, got %q", out)
	}
	if strings.Contains(out, "tool edit: changed file") {
		t.Fatalf("did not expect raw tool line in compact mode, got %q", out)
	}
}

func TestRunStreamingVerboseToolsPrintsRawLines(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho 'tool edit: changed file'\necho '<promise>COMPLETE</promise>'\n")
	var stdout bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"stream verbose tools"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stream:            true,
		VerboseTools:      true,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "tool edit: changed file") {
		t.Fatalf("expected raw tool line in verbose mode, got %q", out)
	}
	if strings.Contains(out, "[bu80] tools:") {
		t.Fatalf("did not expect compact tool summary in verbose mode, got %q", out)
	}
}

func TestRunStreamingSimplifiesClaudeJSON(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho '{\"message\":{\"content\":[{\"text\":\"Hello from Claude\"}]}}'\necho '<promise>COMPLETE</promise>'\n")
	var stdout bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "claude-code",
		PromptArgs:        []string{"show claude output"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stream:            true,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Env: map[string]string{
			"PATH":                os.Getenv("PATH"),
			"RALPH_CLAUDE_BINARY": script,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Hello from Claude") {
		t.Fatalf("expected simplified Claude text in output, got %q", out)
	}
	if strings.Contains(out, "{\"message\"") {
		t.Fatalf("did not expect raw Claude JSON in output, got %q", out)
	}
}

func TestRunOpenCodeSetsGeneratedConfigEnv(t *testing.T) {
	inRepo(t)
	baseConfig := "agents.json"
	writeFile(t, baseConfig, `{"plugins":["auth-provider","misc-plugin"]}`)
	script := writeScript(t, "#!/bin/sh\nprintf '%s' \"$OPENCODE_CONFIG\" > opencode-config-path.txt\necho '<promise>COMPLETE</promise>'\n")
	_, err := RunWithResult(Options{
		Agent:             "opencode",
		ConfigPath:        baseConfig,
		NoPlugins:         true,
		AllowAll:          true,
		PromptArgs:        []string{"run opencode"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stdout:            &bytes.Buffer{},
		Stderr:            &bytes.Buffer{},
		Env: map[string]string{
			"PATH":                  os.Getenv("PATH"),
			"RALPH_OPENCODE_BINARY": script,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pathBytes, readErr := os.ReadFile("opencode-config-path.txt")
	if readErr != nil {
		t.Fatalf("read OPENCODE_CONFIG path: %v", readErr)
	}
	if strings.TrimSpace(string(pathBytes)) != state.OpenCodeConfigFile {
		t.Fatalf("expected OPENCODE_CONFIG path %q, got %q", state.OpenCodeConfigFile, string(pathBytes))
	}
	data, cfgErr := os.ReadFile(state.OpenCodeConfigFile)
	if cfgErr != nil {
		t.Fatalf("read generated OpenCode config: %v", cfgErr)
	}
	if !strings.Contains(string(data), "auth-provider") || strings.Contains(string(data), "misc-plugin") {
		t.Fatalf("expected filtered plugin config, got %s", string(data))
	}
}

func TestRunDetectsOpenCodePlaceholderPluginError(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho 'Loaded legacy placeholder plugin package'\n")
	var stderr bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "opencode",
		PromptArgs:        []string{"run opencode"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stdout:            &bytes.Buffer{},
		Stderr:            &stderr,
		Env: map[string]string{
			"PATH":                  os.Getenv("PATH"),
			"RALPH_OPENCODE_BINARY": script,
		},
	})
	if err == nil {
		t.Fatal("expected placeholder plugin detection to terminate the run")
	}
	if !strings.Contains(stderr.String(), "legacy placeholder plugin") {
		t.Fatalf("expected guidance in stderr, got %q", stderr.String())
	}
}

func TestRunDetectsMissingModelError(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\necho 'Error: no valid model configured'\n")
	var stderr bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"run codex"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		MinIterations:     1,
		MaxIterations:     2,
		Stdout:            &bytes.Buffer{},
		Stderr:            &stderr,
		Env:               envWithScript(script),
	})
	if err == nil {
		t.Fatal("expected missing model detection to terminate the run")
	}
	if !strings.Contains(stderr.String(), "does not have a valid model configured") {
		t.Fatalf("expected guidance in stderr, got %q", stderr.String())
	}
}

func TestRunQuestionToolReadsInlineAnswerAndContinues(t *testing.T) {
	inRepo(t)
	script := writeScript(t, "#!/bin/sh\ncount_file=.count\ncount=0\nif [ -f \"$count_file\" ]; then count=$(cat \"$count_file\"); fi\ncount=$((count + 1))\necho \"$count\" > \"$count_file\"\nfor last do :; done\nprintf '%s' \"$last\" > prompt.txt\nif [ \"$count\" -eq 1 ]; then echo 'tool question: Which config path should I use?'; exit 0; fi\necho '<promise>COMPLETE</promise>'\n")
	var stdout bytes.Buffer
	_, err := RunWithResult(Options{
		Agent:             "codex",
		PromptArgs:        []string{"continue work"},
		CompletionPromise: "COMPLETE",
		TaskPromise:       state.DefaultTaskPromise,
		QuestionsEnabled:  true,
		MinIterations:     1,
		MaxIterations:     3,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Stdin:             bytes.NewBufferString(".bu80/custom.json\n"),
		Env:               envWithScript(script),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Answer recorded.") {
		t.Fatalf("expected inline answer acknowledgement, got %q", stdout.String())
	}
	promptText, readErr := os.ReadFile("prompt.txt")
	if readErr != nil {
		t.Fatalf("read prompt: %v", readErr)
	}
	if !strings.Contains(string(promptText), "A: .bu80/custom.json") {
		t.Fatalf("expected answered question to be injected into next prompt, got %q", string(promptText))
	}
	questions, loadErr := state.LoadQuestions()
	if loadErr != nil {
		t.Fatalf("load questions: %v", loadErr)
	}
	if questions != nil {
		t.Fatalf("expected answered question queue to be consumed, got %+v", questions)
	}
}
