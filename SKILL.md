# ancc

Static validator for the Agent-Native CLI Convention.

## Install

```
brew install ppiankov/tap/ancc
```

Or via Go:

```
go install github.com/ppiankov/ancc/cmd/ancc@latest
```

## Commands

### ancc validate

Validates a CLI tool's repo against the ANCC convention. Checks SKILL.md structure, required sections, and repo contents.

**Flags:**
- `--format json` — output as JSON (default: human-readable)
- `--verbose` — show all checks including passing ones (default: failures and warnings only)

**JSON output:**
```json
{
  "path": "/path/to/repo",
  "status": "pass | fail | partial",
  "checks": [
    {
      "name": "skill-md-exists",
      "status": "pass | fail | warn",
      "message": "SKILL.md found at repo root"
    }
  ],
  "summary": {
    "total": 11,
    "pass": 10,
    "fail": 0,
    "warn": 1
  }
}
```

**Exit codes:**
- 0: all checks pass
- 1: one or more checks fail
- 2: warnings only, no failures

### ancc init

Creates a template SKILL.md in the current directory with all required sections.

**Flags:**
- `--name` — tool name (default: directory name)
- `--force` — overwrite existing SKILL.md

**Exit codes:**
- 0: SKILL.md created successfully
- 1: SKILL.md already exists (without --force) or write error

### ancc doctor

Checks ancc's own health and reports companion tools.

**Flags:**
- `--format json` — output as JSON (default: human-readable)

**JSON output:**
```json
{
  "status": "ok | warn | error",
  "checks": [
    {"name": "go-version", "status": "ok", "message": "go 1.23 found"},
    {"name": "github-api", "status": "ok", "message": "GitHub API reachable"}
  ]
}
```

**Exit codes:**
- 0: all healthy
- 1: critical issue found

### ancc version

Prints the version of ancc.

## What this does NOT do

- Does not install or execute the target tool
- Does not modify any files in the target repo (except `ancc init` which creates SKILL.md)
- Does not require network access for local repos
- Does not validate code quality or test coverage
- Does not publish or register tools anywhere

## Parsing examples

```bash
# Check if a repo passes validation
ancc validate /path/to/repo --format json | jq '.status'

# List failing checks
ancc validate . --format json | jq '.checks[] | select(.status == "fail")'

# Get summary counts
ancc validate . --format json | jq '.summary'

# Use in CI — exit code does the work
ancc validate . || echo "ANCC validation failed"
```
