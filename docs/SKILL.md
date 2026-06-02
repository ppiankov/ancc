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
- `--badge` — include shields.io badge URL in output

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
    "total": 15,
    "pass": 14,
    "fail": 0,
    "warn": 1
  },
  "badge_url": "https://img.shields.io/badge/ANCC-pass-brightgreen"
}
```

Note: `badge_url` field is only present when `--badge` is set.
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

Scans for agent configurations in a directory. Detects 15 agents: Claude Code, Cline, Cursor, OpenCode, Codex, Qwen, OpenClaw, Windsurf, Aider, Continue, Copilot, Kilocode, Vibe, Goose, and Antigravity.

**Flags:**
- `--format json` — output as JSON (default: human-readable)
- `--tokens` — show estimated token counts per agent
- `--budget <N>` — context window size in tokens; shows percentage consumed per agent (implies `--tokens`)

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
      "tokens": 15496,
      "sources": ["~/.claude/settings.json", "~/.claude/skills/"],
      "enforcement": "unverified",
      "advisory": false,
      "budget_pct": 7.7
    },
    {
      "name": "antigravity",
      "skills": 3,
      "hooks": 0,
      "mcp": 0,
      "tokens": 1200,
      "sources": ["~/.gemini/GEMINI.md"],
      "enforcement": "advisory",
      "evidence": [
        {
          "kind": "real_tool_result",
          "note": "trustedWorkspaces does not confine reads to workspace"
        },
        {
          "kind": "unfakeable_output",
          "note": "outside-workspace /tmp read returned a UUID-verified probe payload"
        },
        {
          "kind": "agent_self_report",
          "note": "YES/NO self-report probes are unreliable"
        }
      ],
      "warning": "agent self-reports are not valid evidence for security probes",
      "advisory": true
    }
  ],
  "invalid_locations": [
    {
      "agent": "antigravity",
      "path": "./.antigravitycli/skills/draft",
      "reason": "missing required file SKILL.md"
    }
  ],
  "product": {
    "path": "/path/to/project/docs/SKILL.md",
    "name": "ancc"
  }
}
```

Note: `budget_pct` field is only present when `--budget` is set. `enforcement` reports `enforcing`, `advisory`, or `unverified`; advisory text output includes the evidence standard without changing exit codes. `evidence.kind` values such as `real_tool_result` and `unfakeable_output` can support advisory/enforcing posture; `vendor_docs` and `agent_self_report` cannot. Antigravity skill directories must contain `SKILL.md`; candidates without that marker are reported in `invalid_locations` and are not counted as skills, sources, or tokens.

**Exit codes:**
- 0: scan completed

### ancc audit

Deep inspection of agent configurations. Goes beyond counting to verify that hooks, MCP servers, and skills are valid and functional.

**Flags:**
- `--format json` — output as JSON (default: human-readable)
- `--agent <name>` — audit only this agent

**Checks performed:**
- **Hooks** — does each hook command/script exist? Resolves `~/` paths and PATH lookups
- **MCP servers** — does each server command binary exist in PATH or at its specified path?
- **Skills** — is each skill directory non-empty? Reports file count per skill
- **Environment** — probes sensitive directories, credential directories, shell history files, and credential files for accessibility. Platform-aware (macOS gets Movies+Library, Linux/Windows get Videos). Reports `ok` if blocked or not present, `warn` if accessible. Skipped when `--agent` filter is active.
  - Sensitive dirs: ~/Documents, ~/Downloads, ~/Desktop, ~/Pictures, ~/Music, + platform-specific
  - Credential dirs: ~/.ssh, ~/.aws, ~/.gnupg, ~/.docker, ~/.kube, ~/.azure, ~/.gcloud
  - History files: ~/.bash_history, ~/.zsh_history, ~/.sh_history, ~/.python_history, ~/.node_repl_history
  - Credential files: ~/.netrc, ~/.git-credentials, ~/.npmrc, ~/.pypirc, ~/.gem/credentials, ~/.cargo/credentials.toml

**JSON output:**
```json
{
  "path": "/path/to/project",
  "agents": [
    {
      "name": "claude-code",
      "entries": [
        {
          "category": "hook",
          "name": "PreToolUse/Bash",
          "status": "ok",
          "message": "~/.claude/hooks/bash-guard.sh (found)"
        },
        {
          "category": "mcp",
          "name": "pastewatch",
          "status": "error",
          "message": "pastewatch-cli (not found in PATH)",
          "path": "~/.claude/settings.json"
        }
      ]
    }
  ],
  "environment": [
    {
      "category": "sensitive-dir",
      "name": "~/Documents",
      "status": "ok",
      "message": "blocked (access denied)"
    },
    {
      "category": "credential-dir",
      "name": "~/.ssh",
      "status": "warn",
      "message": "accessible (contains credentials, agents can read)"
    },
    {
      "category": "history-file",
      "name": "~/.zsh_history",
      "status": "warn",
      "message": "accessible (may contain accidentally typed secrets)"
    },
    {
      "category": "credential-file",
      "name": "~/.npmrc",
      "status": "ok",
      "message": "not present"
    }
  ],
  "summary": {"total": 6, "ok": 3, "warn": 2, "errors": 1}
}
```

**Exit codes:**
- 0: all checks pass
- 1: one or more errors found
- 2: warnings only, no errors

### ancc doctor

Checks ancc's own health and reports companion tools.

**Flags:**
- `--format json` — output as JSON (default: human-readable)

**JSON output:**
```json
{
  "status": "ok | warn | error",
  "version": "0.7.1",
  "source": {
    "repo": "github.com/ppiankov/ancc"
  },
  "checks": [
    {"name": "ancc-version", "status": "ok", "message": "0.7.1"},
    {"name": "go-available", "status": "ok", "message": "go version go1.24.0 darwin/arm64"},
    {"name": "github-api", "status": "warn", "message": "GITHUB_TOKEN not set"},
    {"name": "homebrew", "status": "ok", "message": "brew found"}
  ],
  "agents": [
    {
      "name": "antigravity",
      "enforcement": "advisory",
      "enforcement_evidence": "trustedWorkspaces does not confine reads to workspace; outside-workspace /tmp read returned a UUID-verified probe payload",
      "evidence": [
        {
          "kind": "real_tool_result",
          "note": "trustedWorkspaces does not confine reads to workspace"
        },
        {
          "kind": "unfakeable_output",
          "note": "outside-workspace /tmp read returned a UUID-verified probe payload"
        },
        {
          "kind": "agent_self_report",
          "note": "YES/NO self-report probes are unreliable"
        }
      ],
      "warning": "agent self-reports are not valid evidence for security probes"
    }
  ]
}
```

Text output also shows each detected agent's posture. Advisory posture is informational, teaches the valid/invalid evidence standard, and does not make `doctor` fail.

**Exit codes:**
- 0: all healthy or warnings only
- 1: critical issue found

### ancc scan

Batch validates all repos in a directory. Walks the directory tree, finds git repos, and runs ANCC validation on each.

**Flags:**
- `--format json` — output as JSON (default: human-readable)
- `--depth <N>` — maximum directory depth to search (default: 2)

**JSON output:**
```json
{
  "path": "/path/to/parent",
  "repos": [
    {
      "name": "ancc",
      "path": "/path/to/parent/ancc",
      "status": "partial",
      "summary": {
        "total": 15,
        "pass": 14,
        "fail": 0,
        "warn": 1
      }
    },
    {
      "name": "noisepan",
      "path": "/path/to/parent/noisepan",
      "status": "missing"
    }
  ],
  "summary": {
    "total": 2,
    "pass": 0,
    "fail": 0,
    "partial": 1,
    "missing": 1
  }
}
```

**Exit codes:**
- 0: all repos pass (or all missing)
- 1: one or more repos fail
- 2: warnings only across all repos

### ancc context

Shows per-agent token budget breakdown. Displays how much of each agent's context window is consumed by configuration.

**Flags:**
- `--format json` — output as JSON (default: human-readable)
- `--agent <name>` — show only this agent
- `--window <tokens>` — override default context window size

**JSON output:**
```json
{
  "agents": [
    {
      "name": "claude-code",
      "config_tokens": 15569,
      "context_window": 165000,
      "available_tokens": 149431,
      "config_percent": 9.4,
      "skills": 26,
      "hooks": 8,
      "mcp": 0
    }
  ]
}
```

**Exit codes:**
- 0: completed

### ancc diff

Compares agent configurations between two directories. Shows structural differences in skills, hooks, MCP counts, and sources.

**Flags:**
- `--format json` — output as JSON (default: human-readable)
- `--agent <name>` — compare only this agent
- `--tokens` — show token counts

**JSON output:**
```json
{
  "path_a": "/path/to/project-a",
  "path_b": "/path/to/project-b",
  "agents": [
    {
      "name": "claude-code",
      "status": "changed",
      "skills": {"a": 3, "b": 5},
      "hooks": {"a": 1, "b": 1},
      "mcp": {"a": 0, "b": 1},
      "tokens": {"a": 450, "b": 780},
      "sources_added": [".claude/skills/"],
      "sources_common": ["CLAUDE.md"]
    }
  ],
  "summary": {"total": 3, "added": 1, "removed": 0, "changed": 1, "identical": 1}
}
```

**Exit codes:**
- 0: configurations are identical
- 1: differences found

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

# Audit all agents — check hooks, MCP, skills validity
ancc audit

# Audit single agent
ancc audit --agent claude-code

# Get audit errors as JSON
ancc audit --format json | jq '.agents[].entries[] | select(.status == "error")'

# Check environment security — accessible sensitive directories
ancc audit --format json | jq '.environment[] | select(.status == "warn")'

# Batch validate all repos under a directory
ancc scan ~/dev/

# Scan with JSON output — list failing repos
ancc scan ~/dev/ --format json | jq '.repos[] | select(.status == "fail") | .name'

# Scan with limited depth
ancc scan ~/dev/ --depth 1

# Get per-agent budget percentages
ancc skills --budget 200000 --format json | jq '.agents[] | {name, tokens, budget_pct}'

# Get badge URL for a repo
ancc validate . --badge --format json | jq '.badge_url'

# Compare agent configs between two projects
ancc diff /path/to/project-a /path/to/project-b --format json | jq '.summary'

# List changed agents between directories
ancc diff . ../other --format json | jq '.agents[] | select(.status == "changed") | .name'

# Show per-agent context window usage
ancc context --format json | jq '.agents[] | {name, config_percent}'

# Filter context for a single agent
ancc context --agent claude-code --format json

# Check doctor status
ancc doctor --format json | jq '.status'
```

## Supported agents

| Agent | Config paths scanned | Advisory |
|-------|---------------------|----------|
| claude-code | `~/.claude/settings.json`, `~/.claude/skills/`, `~/.claude/CLAUDE.md`, `.claude/skills/`, `.claude/settings.local.json`, `CLAUDE.md`, `CLAUDE.local.md` | No |
| cline | `~/.cline/skills/`, `.clinerules/` | No |
| cursor | `.cursor/rules/*.mdc`, `~/.cursor/mcp.json` | No |
| opencode | `~/.config/opencode/opencode.json`, `~/.config/opencode/commands/`, `~/.config/opencode/skills/`, `opencode.json` (project) | Yes |
| codex | `~/.codex/AGENTS.md`, `~/.codex/skills/`, `~/.codex/config.toml`, `AGENTS.md`, `.codex/` | Yes |
| qwen | `~/.qwen/skills/`, `~/.qwen/settings.json` | Yes |
| openclaw | `~/.openclaw/skills/`, `~/.openclaw/openclaw.json`, `~/.openclaw/config/mcporter.json` | Yes |
| windsurf | `.windsurfrules`, `.windsurf/rules/`, `~/.windsurf/rules/`, `~/.codeium/windsurf/mcp_config.json` | No |
| aider | `~/.aider.conf.yml`, `~/.aider/skills/`, `.aider.conf.yml`, `CONVENTIONS.md` | Yes |
| continue | `~/.continue/config.yaml`, `~/.continue/config.json`, `.continuerc.json` | Yes |
| copilot | `~/.copilot/skills/`, `.github/copilot-instructions.md`, `AGENTS.md` | No |
| kilocode | `~/.kilocode/skills/`, `~/.config/kilo/opencode.json`, `~/.config/kilo/kilo.jsonc`, `.kilocode/rules/`, `.kilocode/skills/`, `kilo.jsonc` | No |
| vibe | `~/.vibe/skills/`, `AGENTS.md` (project) | Yes |
| goose | `~/.config/goose/config.yaml`, `~/.config/goose/skills/`, `.goosehints` | Yes |
| antigravity | `~/.gemini/GEMINI.md`, `~/.gemini/antigravity-cli/skills/`, `~/.gemini/antigravity-cli/global_workflows/`, `~/.gemini/antigravity-cli/workflows/`, `AGENTS.md`, `.antigravitycli/skills/`, `.antigravitycli/workflows/` | Yes |

Advisory agents are detected but not considered primary — their config paths are labeled accordingly in output. Use the `enforcement` field plus structured `evidence` for current posture; the legacy `advisory` field is retained for compatibility. Antigravity skill directories must contain `SKILL.md`; invalid candidates are reported separately instead of being counted.
