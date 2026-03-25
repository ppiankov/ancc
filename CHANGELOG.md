# Changelog

## [Unreleased]

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
