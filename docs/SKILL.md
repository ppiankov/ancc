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

### ancc skills

Scans for agent configurations in a directory. Detects Claude Code, Cline, Cursor, OpenCode, Codex, and Qwen setups.

**Flags:**
- `--format json` — output as JSON (default: human-readable)
- `--tokens` — show estimated token counts per agent

**JSON output:**
```json
{
  "path": "/path/to/project",
  "agents": [
    {
      "name": "claude-code",
      "skills": 26,
      "hooks": 8,
      "mcp": 0,
      "tokens": 3400,
      "sources": ["~/.claude/settings.json", "~/.claude/skills/"],
      "advisory": false
    }
  ],
  "product": {
    "path": "/path/to/project/docs/SKILL.md",
    "name": "mytool"
  }
}
```

**Exit codes:**
- 0: scan completed

### ancc doctor

Checks ancc's own health and reports companion tools.

**Flags:**
- `--format json` — output as JSON (default: human-readable)

**JSON output:**
```json
{
  "status": "ok | warn | error",
  "checks": [
    {"name": "ancc-version", "status": "ok", "message": "0.2.0"},
    {"name": "go-available", "status": "ok", "message": "go version go1.24.0 darwin/arm64"},
    {"name": "github-api", "status": "warn", "message": "GITHUB_TOKEN not set"},
    {"name": "homebrew", "status": "ok", "message": "brew found"}
  ]
}
```

**Exit codes:**
- 0: all healthy or warnings only
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

# List detected agents
ancc skills . --format json | jq '.agents[].name'

# Get per-agent token estimates
ancc skills --tokens . --format json | jq '.agents[] | {name, tokens}'

# Check doctor status
ancc doctor --format json | jq '.status'
```
