# Contributing

Thanks for contributing to Bu80.

## Before you change code

- Read [README.md](README.md) for the current CLI behavior and workflow.
- Review [AGENTS.md](AGENTS.md) for repository conventions.
- Keep changes aligned with the existing Go package layout and naming patterns.

## Development workflow

1. Create a focused branch for your change.
2. Make the smallest change that solves the problem.
3. Add or update tests for behavior changes.
4. Run formatting and tests before opening a pull request.

## Local verification

Use the existing project commands:

```bash
make fmt
make build
make test
./scripts/run-tests.sh
```

If your change affects CLI behavior, update [README.md](README.md) in the same pull request.

## Coding expectations

- Follow standard Go conventions and keep package names lowercase.
- Prefer small, composable functions over large command handlers.
- Keep user-facing output stable unless the change explicitly requires it.
- Avoid unrelated refactors in the same pull request.

## Tests

Add or update `_test.go` coverage near the package you changed. Priority areas include:

- loop stop conditions
- promise detection
- task parsing and task-mode behavior
- persisted state and history handling
- CLI command behavior

## Commit messages

Use short, capitalized, imperative commit subjects, for example:

- `Add security policy`
- `Fix task parser edge case`
- `Update README for tasks mode`

## Pull requests

A good pull request includes:

- a clear summary of the change
- the motivation or bug being addressed
- notes on any CLI or workflow impact
- verification performed

If you are not sure whether a change fits the project direction, open an issue first.
