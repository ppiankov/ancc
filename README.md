[![CI](https://github.com/ppiankov/ancc/actions/workflows/ci.yml/badge.svg)](https://github.com/ppiankov/ancc/actions/workflows/ci.yml)
[![Release](https://github.com/ppiankov/ancc/actions/workflows/release.yml/badge.svg)](https://github.com/ppiankov/ancc/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![ANCC](https://img.shields.io/badge/ANCC-compliant-brightgreen)](https://ancc.dev)

# ancc

Static validator for the [Agent-Native CLI Convention](https://ancc.dev).

## Project Status

**Status: Beta** · **v0.7.0** · Pre-1.0

| Milestone | Status |
|-----------|--------|
| SKILL.md parser + validation (30 checks) | Complete |
| CLI with human + JSON output | Complete |
| GitHub repo support + batch scan | Complete |
| Homebrew distribution | Complete |
| Skills scanner (15 agents) | Complete |
| Init, doctor commands | Complete |
| Token counting + budget visualization | Complete |
| Audit (hooks, MCP, skills, environment) | Complete |
| Context window budget overview | Complete |
| Config diff between directories | Complete |
| CI integration (GitHub Action + badge) | Complete |

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

Agents: read [`docs/SKILL.md`](docs/SKILL.md) for install, commands, JSON parsing patterns, and exit codes.

Key pattern for agents: `ancc validate . --format json` returns machine-parseable validation results.

## Usage

```
ancc validate .
ancc validate --format json .
ancc validate --badge .

ancc init
ancc init --name mytool --force

ancc skills .
ancc skills --tokens .
ancc skills --budget 128000 .

ancc audit
ancc audit --agent claude-code

ancc context .
ancc context --agent claude-code --tokens

ancc diff /path/to/project-a /path/to/project-b
ancc diff . ../other-project --tokens

ancc scan ~/dev/
ancc doctor
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
| `json-examples-valid` | JSON examples parse correctly | warn |
| `exit-codes-numeric` | Exit codes are numeric | warn |
| `commands-not-placeholder` | Commands are not placeholder names | warn |
| `install-has-command` | Install section contains a command | warn |

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
  skills/                -- agent config scanner
```

## Supported Agents

ancc detects 15 agents: claude-code, cline, cursor, opencode, codex, qwen, openclaw, windsurf, aider, continue, copilot, kilocode, vibe, goose, antigravity.

Antigravity detection is advisory while agy CLI customization paths continue to settle.

See [`docs/SKILL.md`](docs/SKILL.md) for full config paths per agent.

## CI Integration

Add ANCC validation to any repo with one workflow file:

```yaml
# .github/workflows/ancc.yml
name: ANCC
on: [push, pull_request]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ppiankov/ancc/.github/action@v0.7.0
        with:
          checks: validate
          fail-on-warn: false
```

Or generate a badge: `ancc validate . --badge`

## Known limitations

- Static validation only — does not install or execute the target tool
- GitHub release check requires network access
- SKILL.md section matching is heading-based, not semantic

## License

[MIT](LICENSE)
