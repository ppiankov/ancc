# Changelog

## [Unreleased]

### Added

- Antigravity advisory scanner support in `ancc skills`, bringing detected agents from 14 to 15
- Invalid or rejected Antigravity scan locations are surfaced in text, JSON, and export output, including `--agent` filtering

### Fixed

- Antigravity skill directories now require a `SKILL.md` marker before being counted as skills or tokens

## [0.8.0] - 2026-03-26

### Added

- `changelog-exists` validator check (warn) — CHANGELOG.md must exist at repo root
- `changelog-version-entry` validator check (fail) — latest git tag must have matching entry
- `doctor-provenance` validator check (warn) — doctor output should include version and source.repo
- Scaffold generates CHANGELOG.md template with initial version entry
- Scaffold generates `.github/workflows/release.yml` with CHANGELOG verification gate
- Scaffold Makefile includes `release` target that refuses to tag without CHANGELOG entry
- Total checks: 30 → 33

## [0.7.1] - 2026-03-22

### Added

- `ancc scaffold` command for full Go project generation
- GitHub Action exits 0 for warnings, 1 for failures only

## [0.7.0] - 2026-03-20

### Added

- 30 validation checks (11 structural, 4 semantic, 5 not-do/scope, 5 spec/policy, 5 quality)
- 14 agent scanners
- GitHub Action for CI validation
- `ancc scan` for batch multi-repo validation
- `ancc context`, `ancc diff`, `ancc export` subcommands

## [0.6.0] - 2026-03-15

### Added

- Vibe and Goose agent support
- Expanded Aider and Copilot scanners

## [0.5.0] - 2026-03-10

### Added

- `ancc context`, `ancc diff`, `ancc export` subcommands
- `--budget` flag for skills context visualization

## [0.4.0] - 2026-03-05

### Added

- Semantic quality checks for SKILL.md validation
- JSON example validation, exit code numeric checks

## [0.3.0] - 2026-02-28

### Added

- OpenClaw agent support
- Environment security checks in `ancc audit`

## [0.2.0] - 2026-02-22

### Added

- Skill scanning, doctor command, agent detection
- 14 agent scanners

## [0.1.0] - 2026-02-15

### Added

- Initial release — ANCC validator with structural checks
- Cobra CLI with validate and init subcommands
