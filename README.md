# Bu80
## Autonomus Agnostic Agent

<img width="821" height="1981" alt="image" src="https://github.com/user-attachments/assets/c70d4fe7-8d18-4ebb-aa8d-28a2c74db791" />

## Development

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
