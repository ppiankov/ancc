# Work Orders — ancc

## Roadmap

- [ ] WO-001: Scaffold Go project
- [ ] WO-002: SKILL.md parser
- [ ] WO-003: Validation checks
- [ ] WO-004: CLI with output formatting
- [ ] WO-005: GitHub repo support
- [ ] WO-006: Self-validation test
- [ ] WO-007: Release pipeline
- [ ] WO-008: Add to ancc.dev site

---

### WO-001: Scaffold Go project

**Status:** `[ ]` planned
**Priority:** high — foundation for everything

### Summary

Initialize Go module, Cobra CLI skeleton, Makefile, CI workflow, and project metadata files.

### Scope

| File | Change |
|------|--------|
| `go.mod` | Create: module github.com/ppiankov/ancc |
| `cmd/ancc/main.go` | Create: entry point delegating to internal/cli |
| `internal/cli/root.go` | Create: Cobra root command with version flag |
| `internal/cli/validate.go` | Create: validate subcommand stub |
| `Makefile` | Create: build, test, lint, install targets |
| `.gitignore` | Create: Go standard ignores |
| `.github/workflows/ci.yml` | Create: Go CI (test, lint, race detection) |
| `LICENSE` | Create: MIT |
| `CHANGELOG.md` | Create: initial entry |
| `CONTRIBUTING.md` | Create: standard contribution guide |

### Acceptance criteria

- [ ] `go build ./cmd/ancc` produces a binary
- [ ] `ancc --version` prints version
- [ ] `ancc validate --help` shows usage
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] CI workflow runs on push

---

### WO-002: SKILL.md parser

**Status:** `[ ]` planned — depends on WO-001
**Priority:** high — core logic

### Summary

Build a parser that reads SKILL.md and extracts structured sections. This is the foundation for all validation checks.

### Scope

| File | Change |
|------|--------|
| `internal/skillmd/parser.go` | Create: parse SKILL.md into structured sections |
| `internal/skillmd/parser_test.go` | Create: tests with fixture files |
| `internal/skillmd/sections.go` | Create: section types and constants |
| `testdata/valid-skill.md` | Create: complete valid SKILL.md |
| `testdata/minimal-skill.md` | Create: bare minimum SKILL.md |
| `testdata/missing-sections.md` | Create: SKILL.md with gaps |
| `testdata/malformed-skill.md` | Create: broken SKILL.md |

### Design

```go
type SkillFile struct {
    Name        string
    Description string
    Sections    map[string]*Section
    Commands    []Command
}

type Section struct {
    Heading string
    Level   int
    Content string
    Raw     string
}

type Command struct {
    Name       string
    Desc       string
    Flags      []Flag
    JSONOutput string
    ExitCodes  []ExitCode
}
```

### Acceptance criteria

- [ ] Parses valid SKILL.md into structured representation
- [ ] Identifies all required sections by heading
- [ ] Extracts command definitions with flags and output schemas
- [ ] Handles missing sections gracefully (returns nil, not panic)
- [ ] All test fixtures pass
- [ ] 90%+ coverage on parser

---

### WO-003: Validation checks

**Status:** `[ ]` planned — depends on WO-002
**Priority:** high — the actual value

### Summary

Implement each validation check as an independent function. Each check takes a parsed SkillFile (and optional repo context) and returns a result.

### Scope

| File | Change |
|------|--------|
| `internal/validator/validator.go` | Create: orchestrator that runs all checks |
| `internal/validator/checks.go` | Create: individual check functions |
| `internal/validator/result.go` | Create: result types |
| `internal/validator/validator_test.go` | Create: tests for each check |

### Design

```go
type CheckResult struct {
    Name    string `json:"name"`
    Status  string `json:"status"` // "pass", "fail", "warn"
    Message string `json:"message"`
}

type ValidationResult struct {
    Path    string        `json:"path"`
    Status  string        `json:"status"` // "pass", "fail", "partial"
    Checks  []CheckResult `json:"checks"`
    Summary Summary       `json:"summary"`
}

type Summary struct {
    Total int `json:"total"`
    Pass  int `json:"pass"`
    Fail  int `json:"fail"`
    Warn  int `json:"warn"`
}
```

### Checks

1. `skill-md-exists` — SKILL.md file present at root
2. `skill-md-install` — has ## Install section
3. `skill-md-commands` — has ## Commands with at least one documented command
4. `skill-md-flags` — documents `--format json` flag
5. `skill-md-json-output` — shows JSON output schema
6. `skill-md-exit-codes` — documents exit codes
7. `skill-md-not-do` — has "What this does NOT do" section
8. `skill-md-parsing` — has parsing examples section
9. `has-init-command` — documents an init command
10. `has-doctor-command` — documents a doctor command (warn, not fail — recommended)
11. `has-binary-release` — GitHub releases contain binaries (warn if not checkable)

### Acceptance criteria

- [ ] Each check is independently testable
- [ ] Valid SKILL.md passes all checks
- [ ] Missing section SKILL.md fails appropriate checks
- [ ] Orchestrator aggregates results correctly
- [ ] Summary counts are accurate

---

### WO-004: CLI with output formatting

**Status:** `[ ]` planned — depends on WO-003
**Priority:** high — the user interface

### Summary

Wire validation logic into the Cobra CLI. Support human-readable and JSON output. Set exit codes based on results.

### Scope

| File | Change |
|------|--------|
| `internal/cli/validate.go` | Update: wire validator, format output |
| `internal/cli/format.go` | Create: human-readable and JSON formatters |
| `internal/cli/format_test.go` | Create: formatter tests |

### Usage

```
ancc validate .                     # validate current directory
ancc validate /path/to/repo         # validate local repo
ancc validate --format json .       # JSON output
ancc validate --verbose .           # show all checks, not just failures
```

### Human output

```
ancc validate .

  SKILL.md exists ..................... PASS
  Install section .................... PASS
  Commands section ................... PASS
  Flags documented ................... PASS
  JSON output schema ................. PASS
  Exit codes documented .............. FAIL  missing exit codes section
  What this does NOT do .............. PASS
  Parsing examples ................... PASS
  Init command ....................... PASS
  Binary release ..................... WARN  not a GitHub repo, skipped

  Result: FAIL (8 pass, 1 fail, 1 warn)
```

### Acceptance criteria

- [ ] `ancc validate .` produces human-readable output
- [ ] `ancc validate --format json .` produces valid JSON
- [ ] Exit code 0 when all pass
- [ ] Exit code 1 when any fail
- [ ] Exit code 2 when warnings only (no fails)
- [ ] `--verbose` shows all checks including passing ones

---

### WO-005: GitHub repo support

**Status:** `[ ]` planned — depends on WO-004
**Priority:** medium — nice to have for remote validation

### Summary

Support validating a GitHub repo by URL. Fetch SKILL.md via GitHub API, check releases for binary assets.

### Scope

| File | Change |
|------|--------|
| `internal/validator/github.go` | Create: GitHub API client for SKILL.md + releases |
| `internal/validator/github_test.go` | Create: tests with mocked responses |

### Usage

```
ancc validate github.com/ppiankov/chainwatch
ancc validate https://github.com/ppiankov/noisepan
```

### Acceptance criteria

- [ ] Fetches SKILL.md from GitHub repo
- [ ] Checks releases for binary assets
- [ ] Works without authentication for public repos
- [ ] Supports optional GITHUB_TOKEN for rate limits
- [ ] Graceful error on private/nonexistent repos

---

### WO-006: Self-validation test

**Status:** `[ ]` planned — depends on WO-004
**Priority:** medium — proves the tool works

### Summary

Add an integration test that runs `ancc validate .` against the ancc repo itself. This proves the tool eats its own cooking.

### Scope

| File | Change |
|------|--------|
| `internal/validator/self_test.go` | Create: self-validation integration test |

### Acceptance criteria

- [ ] `ancc validate .` passes all checks on the ancc repo
- [ ] Test runs in CI
- [ ] Any regression in SKILL.md breaks the test

---

### WO-007: Release pipeline

**Status:** `[ ]` planned — depends on WO-006
**Priority:** medium — needed for distribution

### Summary

Set up GoReleaser and Homebrew tap publishing.

### Scope

| File | Change |
|------|--------|
| `.goreleaser.yml` | Create: multi-platform build config |
| `.github/workflows/release.yml` | Create: tag-triggered release |

### Acceptance criteria

- [ ] `git tag v0.1.0 && git push --tags` triggers release
- [ ] Binaries for darwin/linux amd64/arm64
- [ ] Homebrew formula published to ppiankov/homebrew-tap
- [ ] `brew install ppiankov/tap/ancc` works

---

### WO-008: Add to ancc.dev site

**Status:** `[ ]` planned — depends on WO-007
**Priority:** low — after first release

### Summary

Add ancc to the tools list on ancc.dev and mention validation in the convention section.

### Scope

| File | Change |
|------|--------|
| `../ancc-site/index.html` | Update: add ancc to tools list |

### Acceptance criteria

- [ ] ancc listed on ancc.dev with description
- [ ] Links to GitHub repo
