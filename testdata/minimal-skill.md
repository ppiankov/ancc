# minimal

A minimal tool.

## Install

```
go install github.com/example/minimal@latest
```

## Commands

### minimal run

Runs the thing.

**Flags:**
- `--format json` — JSON output

**JSON output:**
```json
{"ok": true}
```

**Exit codes:**
- 0: success
- 1: failure

## What this does NOT do

- Nothing extra

## Parsing examples

```bash
minimal run --format json | jq '.ok'
```
