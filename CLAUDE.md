# ancc

Static validator for the Agent-Native CLI Convention. Checks whether a CLI tool's repo follows ANCC.

## What this is

A Go CLI that reads a repo (local path or GitHub URL) and validates it against the seven ANCC requirements. Focuses on static validation — checking SKILL.md structure, repo contents, and release artifacts. Does not install or execute the target tool.

## What this is NOT

- Not a runtime test harness (does not run the target tool)
- Not a linter for code quality
- Not a registry or index of ANCC tools
- Not a SKILL.md generator (it validates, not creates)
- Not a framework or library

## Architecture

```
cmd/ancc/main.go        -- entry point, delegates to internal/cli
internal/
  cli/                   -- Cobra command setup, flag parsing, output formatting
  validator/             -- orchestrates all checks, produces results
  skillmd/               -- SKILL.md parser and section validator
```

## Design decisions

- Static validation only (v1). Runtime validation is a future concern
- SKILL.md is the primary validation target — if SKILL.md is correct and complete, the tool is likely compliant
- Checks are independent and produce individual pass/fail/warn results
- Output: human-readable default, `--format json` for machine consumption
- Exit codes: 0 = all pass, 1 = any fail, 2 = partial (warnings only)
- The tool itself must be ANCC-compliant (eats its own cooking)

## Checks (v1)

1. **skill-md-exists** — SKILL.md exists at repo root
2. **skill-md-install** — has ## Install section
3. **skill-md-commands** — has ## Commands section with at least one subcommand
4. **skill-md-flags** — commands document flags including `--format json`
5. **skill-md-json-output** — commands show JSON output schema
6. **skill-md-exit-codes** — has exit codes documented
7. **skill-md-not-do** — has "What this does NOT do" section
8. **skill-md-parsing** — has parsing examples
9. **has-binary-release** — GitHub releases contain binary assets (GitHub repos only)
10. **has-init-command** — SKILL.md documents an init command
11. **has-doctor-command** — SKILL.md documents a doctor command (warn, not fail — recommended extension)

## Style reference

Follow conventions from chainwatch, noisepan, entropia:
- Cobra CLI with root + subcommands
- `--format json` flag on all output commands
- Structured JSON output with consistent schema
- Exit codes: 0 success, 1 failure, 2 partial
- LDFLAGS version injection

## Testing

- Unit tests for SKILL.md parser against fixture files in testdata/
- Unit tests for each individual check
- Integration test: run `ancc validate` against the ancc repo itself (self-validation)
- Test fixtures: valid SKILL.md, minimal SKILL.md, missing sections, malformed SKILL.md
