# demotool

A tool that does something useful.

## Install

```
brew install ppiankov/tap/demotool
```

## Commands

### demotool run

Runs the main operation.

**Flags:**
- `--format json` — output as JSON
- `--verbose` — show detailed output

**JSON output:**
```json
{
  "status": "ok",
  "items": []
}
```

**Exit codes:**
- 0: success
- 1: failure

### demotool check

Checks the current state.

**Flags:**
- `--format json` — output as JSON

**JSON output:**
```json
{
  "healthy": true
}
```

**Exit codes:**
- 0: healthy
- 1: unhealthy

### demotool init

Initializes configuration.

**Exit codes:**
- 0: created
- 1: already exists

### demotool doctor

Checks tool health and dependencies.

**Flags:**
- `--format json` — output as JSON

**Exit codes:**
- 0: all healthy
- 1: issues found

## What this does NOT do

- Does not modify system files or manage configurations
- Does not store or persist any user data
- Does not execute external commands or deploy artifacts

## Parsing examples

```bash
demotool run --format json | jq '.status'
```
