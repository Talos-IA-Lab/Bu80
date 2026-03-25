# Bu80
## Autonomus Agnostic Agent

This repository now contains an in-progress Go reimplementation scaffold for Bu80.

## Development

A local Go toolchain is installed at `./.tools/go/bin/go` for this workspace.

- `make test`
- `make build`
- `make fmt`
- `./scripts/run-tests.sh`

## Current Status

Implemented so far:
- Go module and package skeleton
- CLI flag parsing and validation scaffold
- Completion promise detection helpers with tests
- Task parsing and completion gating helpers with tests
- Prompt source resolution and template rendering with tests
- Agent command resolution, environment overrides, and argv builders with tests
- File-backed state/history/context/questions helpers
- `--status` rendering from persisted `.bu80/` files
- `--list-tasks`, `--add-task`, and `--remove-task`
- `--add-context` and `--clear-context`

Implementation is still in progress. The behavioral contract remains in `GO_REIMPLEMENTATION_SPEC.md`.
