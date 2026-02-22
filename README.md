# ancc

Static validator for the [Agent-Native CLI Convention](https://ancc.dev).

## What it is

A CLI that checks whether a tool's repo follows ANCC — the six requirements that make CLI tools agent-native.

## What it is NOT

- Not a runtime test harness
- Not a linter for code quality
- Not a registry or index
- Not a framework

## Install

```
brew install ppiankov/tap/ancc
```

Or via Go:

```
go install github.com/ppiankov/ancc/cmd/ancc@latest
```

## Usage

```
ancc validate .
ancc validate /path/to/repo
ancc validate --format json .
ancc validate --verbose .
```

## Checks

| Check | What it validates |
|-------|------------------|
| `skill-md-exists` | SKILL.md present at repo root |
| `skill-md-install` | Install section documented |
| `skill-md-commands` | Commands section with subcommands |
| `skill-md-flags` | Flags including `--format json` |
| `skill-md-json-output` | JSON output schema shown |
| `skill-md-exit-codes` | Exit codes documented |
| `skill-md-not-do` | "What this does NOT do" section |
| `skill-md-parsing` | Parsing examples provided |
| `has-init-command` | Init command documented |
| `has-doctor-command` | Doctor command documented (warn) |
| `has-binary-release` | Binary release assets (warn) |

## Exit codes

- `0` — all checks pass
- `1` — one or more checks fail
- `2` — warnings only, no failures

## Architecture

```
cmd/ancc/main.go        -- entry point
internal/
  cli/                   -- Cobra commands, output formatting
  validator/             -- check orchestration, results
  skillmd/               -- SKILL.md parser
```

## Known limitations

- Static validation only — does not install or execute the target tool
- GitHub release check requires network access
- SKILL.md section matching is heading-based, not semantic

## License

MIT
