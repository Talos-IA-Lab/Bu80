# Bu80

## Autonomous agent loop for coding workflows

Bu80 is a Go CLI that runs supported coding agents in an iterative loop until they explicitly signal completion, hit an abort condition, request user input, or exhaust an iteration limit. It is designed to keep a coding task moving forward while preserving enough local state to resume, inspect progress, and manage task-oriented work across multiple iterations.

The current implementation is not just a scaffold. The repository already contains the CLI entrypoint, loop runtime, persisted state helpers, agent registry, prompt generation, task/context commands, status rendering, git integration, and a test suite covering the core behaviors.

<img width="821" height="1981" alt="image" src="https://github.com/user-attachments/assets/c70d4fe7-8d18-4ebb-aa8d-28a2c74db791" />

## What Bu80 does

Bu80 wraps an external agent CLI such as Codex, Claude Code, OpenCode, or Copilot CLI and repeatedly invokes it with a generated prompt. After each iteration it:

- captures output and optionally streams it live
- detects completion, abort, task-handoff, and question signals
- records iteration history and changed files
- auto-commits work by default when git changes exist
- persists loop state under `.bu80/` so status and resume work across runs

The loop stops when one of these conditions occurs:

- the agent emits the configured completion promise
- the agent emits the configured abort promise
- a pending question needs user input
- `--max-iterations` is reached
- the process is interrupted

## Supported agents

Built-in agent definitions currently include:

- `codex`
- `claude-code`
- `opencode`
- `copilot`

Each built-in agent has a default executable name and an environment-variable override:

- `BU80_CODEX_BINARY`
- `BU80_CLAUDE_BINARY`
- `BU80_OPENCODE_BINARY`
- `BU80_COPILOT_BINARY`

Bu80 can also rotate between agent/model pairs with `--rotation`, using entries in `agent:model` format.

## Core loop behavior

A normal run resolves a prompt from one of these sources, in order:

1. `--prompt-file` / `--file` / `-f`
2. a single positional argument that points to an existing file
3. positional arguments joined as inline prompt text
4. the persisted prompt from an active loop state

For each iteration, Bu80 builds a prompt that includes the main objective, iteration metadata, the configured promises, and any saved context. In tasks mode it also includes the current contents of `.bu80/bu80-tasks.md` and instructs the agent to complete one task at a time.

Completion detection is strict: the last non-empty output line must be exactly `<promise> PROMISE_TEXT </promise>` after ANSI cleanup. The default completion promise is `COMPLETE`. In tasks mode the default per-task handoff promise is `READY_FOR_NEXT_TASK`.

If the agent outputs a detectable question and questions are enabled, Bu80 can prompt on stdin for an answer, store it, and merge answered questions back into the saved context for later iterations.

## Tasks mode

Tasks mode is enabled with `--tasks` or `-t`.

Task state lives in `.bu80/bu80-tasks.md` and uses Markdown checkbox-style markers:

```md
# Bu80 Tasks
- [ ] Todo task
- [/] In-progress task
- [x] Completed task
  - [ ] Optional subtask
```

Behavior in tasks mode:

- the agent is asked to pick one incomplete task at a time
- `[/]` marks in-progress work and `[x]` marks completed work
- the task promise signals that the current task is done but more tasks remain
- the completion promise only ends the loop when every parsed task and subtask is complete

Top-level task management is available through:

- `--list-tasks`
- `--add-task "..."`
- `--remove-task N`

## Persisted state

Bu80 keeps runtime files under `.bu80/` in the working repository:

- `.bu80/bu80-loop.state.json`: active loop metadata such as iteration counters, promises, model, agent, rotation, and prompt text
- `.bu80/bu80-history.json`: per-iteration history, durations, files modified, tool names seen in output, exit codes, and struggle indicators
- `.bu80/bu80-context.md`: appended contextual notes and answered-question blocks
- `.bu80/bu80-tasks.md`: task list for tasks mode
- `.bu80/bu80-questions.json`: pending and answered question records
- `.bu80/bu80-opencode.config.json`: generated OpenCode config when plugin filtering or allow-all permissions are requested

`--status` reads these files and prints a concise snapshot of the active loop, recent iterations, struggle indicators, pending context, pending questions, and task progress.

## Configuration and environment

Bu80 supports an agent config file path via `--config`. The default path is `~/.config/bu80/agents.json`.

`--init-config` writes a default config containing the built-in agent types.

For OpenCode runs, Bu80 can generate an adjusted config when:

- `--no-plugins` is set, which filters plugins down to auth-related entries
- `--allow-all` is set, which writes permissive tool permissions into the generated OpenCode config

Additional runtime flags include:

- `--agent`
- `--model`
- `--rotation`
- `--min-iterations`
- `--max-iterations`
- `--completion-promise`
- `--abort-promise`
- `--task-promise`
- `--prompt-template`
- `--stream` / `--no-stream`
- `--verbose-tools`
- `--questions` / `--no-questions`
- `--no-commit`
- `--version`
- `--help`

Arguments after `--` are passed through to the underlying agent invocation.

## Output, history, and interrupts

When streaming is enabled, Bu80 forwards agent output live, emits a periodic `[bu80] still working...` heartbeat during long silent stretches, and can summarize detected tool usage unless `--verbose-tools` is enabled.

History records include:

- iteration number, agent, and model
- start/end timestamps and duration
- exit code
- modified files detected by git snapshot diffing
- completion detection status
- extracted error-like lines
- tool names parsed from the output stream

By default, Bu80 auto-commits changed files after each iteration with a message like `Bu80 iteration N: work in progress`. Use `--no-commit` to disable that behavior.

A first interrupt stops the current agent process and clears active loop state. A second interrupt forces process exit.

## Repository layout

Current top-level layout:

- `cmd/bu80/`: CLI entrypoint
- `internal/agent/`: built-in agent definitions, rotation parsing, argv construction
- `internal/cli/`: flag parsing and command dispatch
- `internal/config/`: config initialization and OpenCode env/config generation
- `internal/contextcmd/`: add and clear context helpers
- `internal/gitutil/`: git snapshotting, modified-file detection, auto-commit
- `internal/history/`: iteration history types
- `internal/loop/`: main iterative runtime, streaming, questions, interrupt handling
- `internal/output/`: promise detection, tool extraction, streaming display helpers
- `internal/prompt/`: prompt source resolution and prompt/template rendering
- `internal/state/`: persisted `.bu80/` files and question/context/task helpers
- `internal/statuscmd/`: status snapshot rendering
- `internal/taskcmd/`: task list/add/remove commands
- `internal/tasks/`: Markdown task parser and formatter
- `scripts/run-tests.sh`: convenience wrapper for `go test ./...`

## Build, test, and development

The repository includes a `Makefile` and test script.

```bash
make fmt
make build
make test
./scripts/run-tests.sh
```

The Make targets prefer a local Go toolchain under `.tools/go/bin/` when present, and otherwise fall back to the system `go` / `gofmt`.

## Example commands

Run a single loop with an inline prompt:

```bash
bu80 "Implement the failing parser tests and stop only when they pass"
```

Run with a prompt file and explicit model:

```bash
bu80 --agent codex --model gpt-5.4 --prompt-file prompts/task.md
```

Use tasks mode:

```bash
bu80 --tasks "Work through the repository task list"
```

Inspect current state:

```bash
bu80 --status
```

Append context for the next iteration:

```bash
bu80 --add-context "Prefer minimal API surface changes and keep the patch backward compatible."
```

Initialize the default agent config:

```bash
bu80 --init-config
```

## Current status

Implemented today:

- Go CLI entrypoint and flag parsing
- iterative loop runtime with resume support
- prompt-file, inline prompt, and prompt-template support
- completion, abort, and task-promise detection
- tasks mode and Markdown task parsing
- question detection and answer persistence
- persisted loop/history/context/task/question state under `.bu80/`
- status reporting from saved state
- agent rotation across agent/model pairs
- git-based modified-file tracking and auto-commit
- streaming output, tool summaries, and heartbeat messages
- config initialization and OpenCode config generation
- package-level tests across the main subsystems

## License

This repository is licensed under the terms in [LICENSE](LICENSE).
