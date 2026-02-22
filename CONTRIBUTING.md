# Contributing to ancc

## Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/welcome/install/)

## Development

```bash
git clone https://github.com/ppiankov/ancc.git
cd ancc
make build
make test
make lint
```

## Making changes

1. Fork the repo and create a feature branch
2. Write tests for new functionality
3. Run `make test` and `make lint` before committing
4. Use conventional commits: `feat:`, `fix:`, `test:`, `refactor:`, `docs:`, `chore:`
5. Open a pull request against `main`

## Project layout

```
cmd/ancc/main.go        -- entry point
internal/
  cli/                   -- Cobra command setup, flags, output formatting
  validator/             -- validation orchestration and results
  skillmd/               -- SKILL.md parser and section constants
```

## Testing

Tests live alongside source files. Run with race detection:

```bash
make test
```

Coverage target: >85%.

## Code style

- Named exports, early returns
- Comments explain "why" not "what"
- No magic numbers
