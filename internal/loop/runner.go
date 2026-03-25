package loop

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"bu80/internal/agent"
	"bu80/internal/config"
	"bu80/internal/gitutil"
	"bu80/internal/history"
	"bu80/internal/output"
	"bu80/internal/prompt"
	"bu80/internal/state"
	"bu80/internal/tasks"
)

var (
	ErrInterrupted     = errors.New("interrupted")
	ErrForcedInterrupt = errors.New("force interrupted")
	defaultForceExit   = func(code int) { os.Exit(code) }
)

type Options struct {
	Agent             string
	Model             string
	PromptFile        string
	PromptArgs        []string
	PromptTemplate    string
	CompletionPromise string
	AbortPromise      string
	TaskPromise       string
	Rotation          string
	ConfigPath        string
	NoPlugins         bool
	QuestionsEnabled  bool
	VerboseTools      bool
	TasksMode         bool
	MinIterations     int
	MaxIterations     int
	AllowAll          bool
	NoCommit          bool
	ExtraArgs         []string
	Stream            bool
	HeartbeatInterval time.Duration
	Stdout            io.Writer
	Stderr            io.Writer
	Stdin             io.Reader
	Now               func() time.Time
	Env               map[string]string
	Interrupts        <-chan os.Signal
	ForceExit         func(int)
}

type Result struct {
	Completed bool
	Aborted   bool
	Reason    string
}

type interruptMonitor struct {
	mu              sync.Mutex
	stopCh          chan struct{}
	stopOnce        sync.Once
	interruptedFlag bool
	forcedFlag      bool
}

func Run(opts Options) error {
	_, err := RunWithResult(opts)
	return err
}

func RunWithResult(opts Options) (Result, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Env == nil {
		opts.Env = currentEnv()
	}
	if opts.HeartbeatInterval <= 0 {
		opts.HeartbeatInterval = defaultHeartbeatInterval(opts.Env)
	}
	if opts.ForceExit == nil {
		opts.ForceExit = defaultForceExit
	}
	if opts.Interrupts == nil {
		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigCh)
		opts.Interrupts = sigCh
	}

	defs := agent.Builtins()
	if _, ok := defs[opts.Agent]; !ok {
		return Result{}, fmt.Errorf("unknown agent: %s", opts.Agent)
	}

	rotationEntries, err := agent.ParseRotation(opts.Rotation, defs)
	if err != nil {
		return Result{}, err
	}

	resumeState, err := state.LoadLoopState()
	if err != nil {
		return Result{}, err
	}
	promptText, err := prompt.ResolveSource(prompt.SourceOptions{
		PromptFile: opts.PromptFile,
		PromptArgs: opts.PromptArgs,
		Resume:     resumeState,
	})
	if err != nil {
		return Result{}, err
	}
	templateText, err := resolvePromptTemplate(opts.PromptTemplate, resumeState)
	if err != nil {
		return Result{}, err
	}

	loopState := buildInitialState(opts, promptText, templateText, agent.FormatRotation(rotationEntries), resumeState, opts.Now())
	if err := state.SaveLoopState(loopState); err != nil {
		return Result{}, err
	}

	hist, err := state.LoadHistory()
	if err != nil {
		return Result{}, err
	}
	if hist == nil {
		hist = &history.History{}
	}

	for {
		if loopState.MaxIterations > 0 && loopState.Iteration > loopState.MaxIterations {
			if err := state.ClearLoopState(); err != nil {
				return Result{}, err
			}
			if err := state.ClearQuestions(); err != nil {
				return Result{}, err
			}
			fmt.Fprintf(opts.Stdout, "Reached max iterations: %d\n", loopState.MaxIterations)
			return Result{Reason: "max-iterations"}, nil
		}

		currentAgent, def, command, err := resolveIterationAgent(loopState, defs, opts.Env)
		if err != nil {
			return Result{}, err
		}
		iterationPrompt, err := buildIterationPrompt(loopState)
		if err != nil {
			return Result{}, err
		}

		iterationStart := opts.Now()
		args, err := agent.BuildArgs(def, agent.InvocationOptions{
			Prompt:    iterationPrompt,
			Model:     currentAgent.Model,
			ExtraArgs: opts.ExtraArgs,
			AllowAll:  opts.AllowAll,
		})
		if err != nil {
			return Result{}, err
		}

		beforeSnapshot, snapErr := gitutil.CaptureSnapshot()
		if snapErr != nil {
			return Result{}, snapErr
		}

		cmd := exec.Command(command, args...)
		runEnv, err := buildRunEnv(opts, currentAgent.Agent)
		if err != nil {
			return Result{}, err
		}
		cmd.Env = envSlice(runEnv)
		outputText, exitCode, runErr := runCommand(cmd, opts, currentAgent.Agent)
		if errors.Is(runErr, ErrInterrupted) {
			if err := clearRunState(false, false); err != nil {
				return Result{}, err
			}
			fmt.Fprintln(opts.Stdout, "Interrupted.")
			return Result{Reason: "interrupted"}, nil
		}
		if errors.Is(runErr, ErrForcedInterrupt) {
			if err := clearRunState(false, false); err != nil {
				return Result{}, err
			}
			return Result{Reason: "force-interrupted"}, runErr
		}
		if runErr != nil {
			fmt.Fprintf(opts.Stderr, "Agent exited with code %d\n", exitCode)
		}

		if currentAgent.Agent == "opencode" && output.DetectOpenCodePlaceholderPlugin(outputText) {
			if err := clearRunState(false, false); err != nil {
				return Result{}, err
			}
			fmt.Fprintln(opts.Stderr, "OpenCode loaded a legacy placeholder plugin. Remove it and retry.")
			return Result{Reason: "opencode-placeholder-plugin"}, errors.New("opencode placeholder plugin detected")
		}
		if output.DetectMissingModel(outputText) {
			if err := clearRunState(false, false); err != nil {
				return Result{}, err
			}
			fmt.Fprintln(opts.Stderr, "The selected agent does not have a valid model configured. Set a model and retry.")
			return Result{Reason: "missing-model"}, errors.New("missing model configuration")
		}

		completionDetected := output.DetectPromise(outputText, loopState.CompletionPromise)
		taskDetected := loopState.TasksMode && output.DetectPromise(outputText, loopState.TaskPromise)
		abortDetected := output.DetectPromise(outputText, loopState.AbortPromise)
		if loopState.Iteration < loopState.MinIterations {
			completionDetected = false
		}
		if loopState.TasksMode && completionDetected {
			tasksRaw, readErr := state.ReadTasks()
			if readErr != nil {
				return Result{}, readErr
			}
			if !tasks.AllComplete(tasksRaw) {
				completionDetected = false
			}
		}

		modifiedFiles, modErr := gitutil.DetectModifiedFiles(beforeSnapshot)
		if modErr != nil {
			return Result{}, modErr
		}
		if !opts.NoCommit {
			if commitErr := gitutil.AutoCommit(loopState.Iteration); commitErr != nil {
				fmt.Fprintf(opts.Stderr, "Auto-commit failed: %v\n", commitErr)
			}
		}

		iterationEnd := opts.Now()
		record := history.IterationRecord{
			Iteration:          loopState.Iteration,
			StartedAt:          iterationStart,
			EndedAt:            iterationEnd,
			DurationMs:         iterationEnd.Sub(iterationStart).Milliseconds(),
			Agent:              currentAgent.Agent,
			Model:              currentAgent.Model,
			FilesModified:      modifiedFiles,
			ExitCode:           exitCode,
			CompletionDetected: completionDetected,
			Errors:             extractErrors(outputText),
			ToolsUsed:          output.ParseTools(outputText),
		}
		appendHistory(hist, record)
		if err := state.SaveHistory(*hist); err != nil {
			return Result{}, err
		}

		if opts.QuestionsEnabled {
			questionText := detectQuestion(outputText)
			if questionText != "" {
				fmt.Fprintf(opts.Stdout, "Question requested: %s\n", questionText)
				answer, answered := promptForAnswer(opts.Stdin, opts.Stdout, questionText)
				if answered {
					if err := state.SaveAnswer(questionText, answer, opts.Now()); err != nil {
						return Result{}, err
					}
					fmt.Fprintln(opts.Stdout, "Answer recorded.")
					advanceLoopState(&loopState)
					if err := state.SaveLoopState(loopState); err != nil {
						return Result{}, err
					}
					continue
				}
				if err := state.AddPendingQuestion(questionText, opts.Now()); err != nil {
					return Result{}, err
				}
				return Result{Reason: "question"}, errors.New("question requires user input")
			}
		}

		if abortDetected {
			if err := clearRunState(true, true); err != nil {
				return Result{}, err
			}
			fmt.Fprintln(opts.Stdout, "Abort promise detected.")
			return Result{Aborted: true, Reason: "abort"}, errors.New("abort promise detected")
		}
		if completionDetected {
			if err := clearRunState(true, true); err != nil {
				return Result{}, err
			}
			fmt.Fprintln(opts.Stdout, "Completion detected.")
			return Result{Completed: true, Reason: "complete"}, nil
		}
		if taskDetected {
			fmt.Fprintln(opts.Stdout, "Task promise detected.")
		}

		advanceLoopState(&loopState)
		if err := state.SaveLoopState(loopState); err != nil {
			return Result{}, err
		}
	}
}

func promptForAnswer(r io.Reader, w io.Writer, question string) (string, bool) {
	if r == nil {
		return "", false
	}
	_, _ = fmt.Fprintf(w, "Answer for '%s': ", question)
	reader := bufio.NewReader(r)
	answer, err := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if err != nil && answer == "" {
		return "", false
	}
	if answer == "" {
		return "", false
	}
	return answer, true
}

func runCommand(cmd *exec.Cmd, opts Options, agentName string) (string, int, error) {
	if !opts.Stream {
		return runCommandBuffered(cmd, opts)
	}
	return runCommandStreaming(cmd, opts, agentName)
}

func runCommandBuffered(cmd *exec.Cmd, opts Options) (string, int, error) {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return "", 1, err
	}

	monitor := startInterruptMonitor(cmd, opts, nil)
	runErr := cmd.Wait()
	monitor.stop()

	outputText := stdoutBuf.String() + stderrBuf.String()
	if outputText != "" {
		_, _ = io.WriteString(opts.Stdout, outputText)
		if !strings.HasSuffix(outputText, "\n") {
			_, _ = io.WriteString(opts.Stdout, "\n")
		}
	}
	if monitor.forced() {
		return outputText, 1, ErrForcedInterrupt
	}
	if monitor.interrupted() {
		return outputText, exitCodeFromResult(runErr), ErrInterrupted
	}
	if runErr != nil {
		return outputText, exitCodeOf(runErr), runErr
	}
	return outputText, 0, nil
}

func runCommandStreaming(cmd *exec.Cmd, opts Options, agentName string) (string, int, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", 1, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", 1, err
	}
	if err := cmd.Start(); err != nil {
		return "", 1, err
	}

	var buf bytes.Buffer
	var mu sync.Mutex
	toolCounts := map[string]int{}
	lastSummary := ""
	lastOutput := time.Now()
	setLastOutput := func() {
		mu.Lock()
		lastOutput = time.Now()
		mu.Unlock()
	}
	getLastOutput := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return lastOutput
	}
	appendOutput := func(text string) {
		mu.Lock()
		buf.WriteString(text)
		mu.Unlock()
	}
	emitToolSummary := func() {
		mu.Lock()
		summary := output.FormatToolSummary(toolCounts)
		if summary == "" || summary == lastSummary {
			mu.Unlock()
			return
		}
		lastSummary = summary
		mu.Unlock()
		_, _ = fmt.Fprintln(opts.Stdout, summary)
	}

	heartbeatStop := make(chan struct{})
	monitor := startInterruptMonitor(cmd, opts, func() { closeChannel(heartbeatStop) })

	var wg sync.WaitGroup
	streamPipe := func(r io.Reader, w io.Writer) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			appendOutput(line + "\n")
			setLastOutput()
			toolName := output.ToolName(line)
			if toolName != "" {
				mu.Lock()
				toolCounts[toolName]++
				mu.Unlock()
				if !opts.VerboseTools {
					emitToolSummary()
					continue
				}
			}
			displayLine := output.SimplifyDisplayLine(agentName, line)
			if strings.TrimSpace(displayLine) == "" {
				continue
			}
			_, _ = fmt.Fprintln(w, displayLine)
		}
	}

	wg.Add(2)
	go streamPipe(stdoutPipe, opts.Stdout)
	go streamPipe(stderrPipe, opts.Stderr)
	go runHeartbeat(opts, getLastOutput, heartbeatStop)

	runErr := cmd.Wait()
	wg.Wait()
	closeChannel(heartbeatStop)
	monitor.stop()

	mu.Lock()
	outputText := buf.String()
	mu.Unlock()
	if monitor.forced() {
		return outputText, 1, ErrForcedInterrupt
	}
	if monitor.interrupted() {
		return outputText, exitCodeFromResult(runErr), ErrInterrupted
	}
	if runErr != nil {
		return outputText, exitCodeOf(runErr), runErr
	}
	return outputText, 0, nil
}

func runHeartbeat(opts Options, getLastOutput func() time.Time, stop <-chan struct{}) {
	ticker := time.NewTicker(opts.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if time.Since(getLastOutput()) >= opts.HeartbeatInterval {
				_, _ = fmt.Fprintln(opts.Stdout, "[bu80] still working...")
			}
		case <-stop:
			return
		}
	}
}

func startInterruptMonitor(cmd *exec.Cmd, opts Options, onFirstInterrupt func()) *interruptMonitor {
	monitor := &interruptMonitor{stopCh: make(chan struct{})}
	if opts.Interrupts == nil {
		return monitor
	}
	go func() {
		count := 0
		for {
			select {
			case <-monitor.stopCh:
				return
			case _, ok := <-opts.Interrupts:
				if !ok {
					return
				}
				count++
				if count == 1 {
					monitor.mu.Lock()
					monitor.interruptedFlag = true
					monitor.mu.Unlock()
					if onFirstInterrupt != nil {
						onFirstInterrupt()
					}
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
					}
				} else {
					monitor.mu.Lock()
					monitor.forcedFlag = true
					monitor.mu.Unlock()
					if opts.ForceExit != nil {
						opts.ForceExit(1)
					}
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
					}
					return
				}
			}
		}
	}()
	return monitor
}

func (m *interruptMonitor) stop() {
	m.stopOnce.Do(func() { close(m.stopCh) })
}

func (m *interruptMonitor) interrupted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.interruptedFlag
}

func (m *interruptMonitor) forced() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.forcedFlag
}

func defaultHeartbeatInterval(env map[string]string) time.Duration {
	if env["NODE_ENV"] == "test" {
		return time.Second
	}
	return 10 * time.Second
}

func buildInitialState(opts Options, promptText string, templateText string, rotation []string, resume *state.LoopState, now time.Time) state.LoopState {
	if resume != nil && resume.Active && strings.TrimSpace(opts.PromptFile) == "" && len(opts.PromptArgs) == 0 && strings.TrimSpace(opts.Rotation) == "" && strings.TrimSpace(opts.PromptTemplate) == "" {
		return *resume
	}
	return state.LoopState{
		Active:            true,
		Iteration:         1,
		MinIterations:     opts.MinIterations,
		MaxIterations:     opts.MaxIterations,
		CompletionPromise: opts.CompletionPromise,
		AbortPromise:      opts.AbortPromise,
		TasksMode:         opts.TasksMode,
		TaskPromise:       opts.TaskPromise,
		Prompt:            promptText,
		PromptTemplate:    templateText,
		StartedAt:         now,
		Model:             opts.Model,
		Agent:             opts.Agent,
		Rotation:          rotation,
		RotationIndex:     0,
	}
}

func resolvePromptTemplate(path string, resume *state.LoopState) (string, error) {
	if strings.TrimSpace(path) != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if resume != nil && resume.Active && strings.TrimSpace(resume.PromptTemplate) != "" {
		return resume.PromptTemplate, nil
	}
	return "", nil
}

func buildIterationPrompt(loopState state.LoopState) (string, error) {
	contextText, err := state.ReadContext()
	if err != nil {
		return "", err
	}
	answered, err := state.ConsumeAnsweredQuestions()
	if err != nil {
		return "", err
	}
	if len(answered) > 0 {
		contextText = state.MergeContext(contextText, state.FormatAnswersBlock(answered))
		if err := state.WriteContext(contextText); err != nil {
			return "", err
		}
	}
	tasksText, err := state.ReadTasks()
	if err != nil {
		return "", err
	}
	data := prompt.Data{
		Iteration:         loopState.Iteration,
		MaxIterations:     loopState.MaxIterations,
		MinIterations:     loopState.MinIterations,
		Prompt:            loopState.Prompt,
		CompletionPromise: loopState.CompletionPromise,
		AbortPromise:      loopState.AbortPromise,
		TaskPromise:       loopState.TaskPromise,
		Context:           contextText,
		Tasks:             tasksText,
	}
	return prompt.Build(data, loopState.TasksMode, loopState.PromptTemplate), nil
}

func resolveIterationAgent(loopState state.LoopState, defs map[string]agent.Definition, env map[string]string) (agent.RotationEntry, agent.Definition, string, error) {
	current := agent.CurrentRotationEntry(loopState.Rotation, loopState.RotationIndex, loopState.Agent, loopState.Model)
	def, ok := defs[current.Agent]
	if !ok {
		return agent.RotationEntry{}, agent.Definition{}, "", fmt.Errorf("unknown agent: %s", current.Agent)
	}
	command := agent.ResolveCommand(def, env)
	if _, err := exec.LookPath(command); err != nil {
		return agent.RotationEntry{}, agent.Definition{}, "", fmt.Errorf("agent executable not found: %s", command)
	}
	return current, def, command, nil
}

func advanceLoopState(loopState *state.LoopState) {
	loopState.Iteration++
	if len(loopState.Rotation) > 0 {
		loopState.RotationIndex = (loopState.RotationIndex + 1) % len(loopState.Rotation)
	}
}

func buildRunEnv(opts Options, agentName string) (map[string]string, error) {
	if agentName == "opencode" {
		return config.BuildOpenCodeEnv(opts.Env, opts.ConfigPath, opts.NoPlugins, opts.AllowAll)
	}
	return currentEnvClone(opts.Env), nil
}

func currentEnvClone(env map[string]string) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = v
	}
	return out
}

func appendHistory(hist *history.History, record history.IterationRecord) {
	hist.Iterations = append(hist.Iterations, record)
	hist.TotalDurationMs += record.DurationMs
	if record.DurationMs < 30000 {
		hist.StruggleIndicators.ShortIterations++
	}
	if len(record.Errors) > 0 {
		hist.StruggleIndicators.RepeatedErrors++
	}
	if len(record.FilesModified) == 0 {
		hist.StruggleIndicators.NoProgressIters++
	}
}

func clearRunState(clearHistory bool, clearContext bool) error {
	if err := state.ClearLoopState(); err != nil {
		return err
	}
	if clearHistory {
		if err := state.ClearHistory(); err != nil {
			return err
		}
	}
	if clearContext {
		if err := state.ClearContext(); err != nil {
			return err
		}
	}
	if err := state.ClearQuestions(); err != nil {
		return err
	}
	return nil
}

func currentEnv() map[string]string {
	result := make(map[string]string)
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func envSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func extractErrors(outputText string) []string {
	lines := strings.Split(outputText, "\n")
	errs := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
			errs = append(errs, trimmed)
		}
	}
	return errs
}

func exitCodeFromResult(err error) int {
	if err == nil {
		return 0
	}
	return exitCodeOf(err)
}

func exitCodeOf(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func closeChannel(ch chan struct{}) {
	defer func() { _ = recover() }()
	close(ch)
}
