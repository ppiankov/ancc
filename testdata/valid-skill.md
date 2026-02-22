# mytool

A tool that does something useful.

## Install

```
brew install ppiankov/tap/mytool
```

## Commands

### mytool run

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

### mytool check

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

### mytool init

Initializes configuration.

**Exit codes:**
- 0: created
- 1: already exists

### mytool doctor

Checks tool health and dependencies.

**Flags:**
- `--format json` — output as JSON

**Exit codes:**
- 0: all healthy
- 1: issues found

## What this does NOT do

- Does not modify system files
- Does not require root access

## Parsing examples

```bash
mytool run --format json | jq '.status'
```
