[![CI](https://github.com/ppiankov/ancc/actions/workflows/ci.yml/badge.svg)](https://github.com/ppiankov/ancc/actions/workflows/ci.yml)
[![Release](https://github.com/ppiankov/ancc/actions/workflows/release.yml/badge.svg)](https://github.com/ppiankov/ancc/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

# ancc

Static validator for the [Agent-Native CLI Convention](https://ancc.dev).

## Project Status

**Status: Beta** · **v0.1.0** · Pre-1.0

| Milestone | Status |
|-----------|--------|
| SKILL.md parser | Complete |
| Validation checks (11 checks) | Complete |
| CLI with human + JSON output | Complete |
| GitHub repo support | Complete |
| Self-validation test | Complete |
| Homebrew distribution | Complete |
| Init command (SKILL.md generator) | Planned |
| Doctor command | Planned |

Pre-1.0: check names and JSON output structure may change between minor versions.

---

## What it is

A CLI that checks whether a tool's repo follows ANCC — the six requirements that make CLI tools agent-native.

## What it is NOT

- Not a runtime test harness (does not run the target tool)
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

### Agent Integration

ancc is itself ANCC-compliant. Single binary, deterministic output, structured JSON, bounded jobs.

Agents: read [`SKILL.md`](SKILL.md) for install, commands, JSON parsing patterns, and exit codes.

Key pattern for agents: `ancc validate . --format json` returns machine-parseable validation results.

## Usage

```
ancc validate .
ancc validate /path/to/repo
ancc validate --format json .
ancc validate --verbose .
```

## Checks

| Check | What it validates | Severity |
|-------|------------------|----------|
| `skill-md-exists` | SKILL.md present at repo root | fail |
| `skill-md-install` | Install section documented | fail |
| `skill-md-commands` | Commands section with subcommands | fail |
| `skill-md-flags` | Flags including `--format json` | fail |
| `skill-md-json-output` | JSON output schema shown | fail |
| `skill-md-exit-codes` | Exit codes documented | fail |
| `skill-md-not-do` | "What this does NOT do" section | fail |
| `skill-md-parsing` | Parsing examples provided | fail |
| `has-init-command` | Init command documented | fail |
| `has-doctor-command` | Doctor command documented | warn |
| `has-binary-release` | Binary release assets | warn |

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

[MIT](LICENSE)
