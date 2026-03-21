package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	scaffoldTypeScanner    = "scanner"
	scaffoldTypeDiagnostic = "diagnostic"
)

func newScaffoldCmd() *cobra.Command {
	var toolType string
	var force bool

	cmd := &cobra.Command{
		Use:   "scaffold <name>",
		Short: "Generate a complete ANCC-compliant Go project",
		Long: `Generate a complete, buildable Go project that passes ancc validate on first run.

The generated project includes: cmd/, internal/, Makefile, go.mod, CI workflows,
docs/SKILL.md, README.md, and working test stubs.

Types:
  scanner     — security scanner (spectre pattern: scan + init, spectre/v1 output)
  diagnostic  — health checker (check + init, status output)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if name == "" {
				return fmt.Errorf("tool name is required")
			}

			// Validate type.
			switch toolType {
			case scaffoldTypeScanner, scaffoldTypeDiagnostic:
			default:
				return fmt.Errorf("unknown type %q (use: scanner, diagnostic)", toolType)
			}

			outDir := name
			if !force {
				if _, err := os.Stat(outDir); err == nil {
					return fmt.Errorf("directory %q already exists (use --force to overwrite)", outDir)
				}
			}

			files := generateProject(name, toolType)

			for relPath, content := range files {
				fullPath := filepath.Join(outDir, relPath)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
					return fmt.Errorf("creating directory for %s: %w", relPath, err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", relPath, err)
				}
			}

			// Make entrypoint.sh executable if present.
			entrypoint := filepath.Join(outDir, ".github", "action", "entrypoint.sh")
			if _, err := os.Stat(entrypoint); err == nil {
				_ = os.Chmod(entrypoint, 0o755)
			}

			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "Scaffolded %s (%s) in ./%s/\n", name, toolType, outDir)
			_, _ = fmt.Fprintf(w, "\nNext steps:\n")
			_, _ = fmt.Fprintf(w, "  cd %s\n", outDir)
			_, _ = fmt.Fprintf(w, "  go mod tidy\n")
			_, _ = fmt.Fprintf(w, "  make build\n")
			_, _ = fmt.Fprintf(w, "  make test\n")
			_, _ = fmt.Fprintf(w, "  ancc validate .\n")
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "Genesis loop: search → scaffold → validate → use → compose → govern")
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&toolType, "type", scaffoldTypeScanner, "project type (scanner, diagnostic)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing directory")

	return cmd
}

// generateProject returns a map of relative path → file content.
func generateProject(name, toolType string) map[string]string {
	files := make(map[string]string)

	files["go.mod"] = scaffoldGoMod(name)
	files[filepath.Join("cmd", name, "main.go")] = scaffoldMain(name)
	files[filepath.Join("internal", "commands", "root.go")] = scaffoldRootCmd(name)
	files["Makefile"] = scaffoldMakefile(name)
	files[".gitignore"] = scaffoldGitignore(name)
	files["README.md"] = scaffoldReadme(name)
	files[filepath.Join(".github", "workflows", "ci.yml")] = scaffoldCI(name)
	files[filepath.Join(".github", "workflows", "ancc.yml")] = scaffoldANCCWorkflow()

	switch toolType {
	case scaffoldTypeScanner:
		files[filepath.Join("internal", "commands", "scan.go")] = scaffoldScanCmd(name)
		files[filepath.Join("internal", "commands", "initcmd.go")] = scaffoldInitCmd(name)
		files[filepath.Join("internal", "commands", "scan_test.go")] = scaffoldScanTest(name)
		files["docs/SKILL.md"] = scaffoldScannerSkillMD(name)
	case scaffoldTypeDiagnostic:
		files[filepath.Join("internal", "commands", "check.go")] = scaffoldCheckCmd(name)
		files[filepath.Join("internal", "commands", "initcmd.go")] = scaffoldInitCmd(name)
		files[filepath.Join("internal", "commands", "check_test.go")] = scaffoldCheckTest(name)
		files["docs/SKILL.md"] = scaffoldDiagnosticSkillMD(name)
	}

	return files
}

func scaffoldGoMod(name string) string {
	return fmt.Sprintf(`module github.com/yourorg/%s

go 1.24.0

require github.com/spf13/cobra v1.10.2

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)
`, name)
}

func scaffoldMain(name string) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"github.com/yourorg/%s/internal/commands"
)

var version = "dev"

func main() {
	if err := commands.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`, name)
}

func scaffoldRootCmd(name string) string {
	return fmt.Sprintf(`package commands

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "%s",
	Short: "One-line description of %s",
}

// Execute runs the root command.
func Execute(version string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}
`, name, name)
}

func scaffoldScanCmd(name string) string {
	return fmt.Sprintf(`package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var format string

func init() {
	scanCmd.Flags().StringVar(&format, "format", "text", "output format: text, json")
	rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for findings",
	RunE: func(cmd *cobra.Command, args []string) error {
		result := ScanResult{
			Version: "spectre/v1",
			Scanner: "%s",
			Target:  "target",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Findings: []Finding{},
			Summary: Summary{},
		}

		switch format {
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		default:
			fmt.Println("No findings.")
			return nil
		}
	},
}

// ScanResult is the spectre/v1 output envelope.
type ScanResult struct {
	Version   string    `+"`json:\"version\"`"+`
	Scanner   string    `+"`json:\"scanner\"`"+`
	Target    string    `+"`json:\"target\"`"+`
	Timestamp string    `+"`json:\"timestamp\"`"+`
	Findings  []Finding `+"`json:\"findings\"`"+`
	Summary   Summary   `+"`json:\"summary\"`"+`
}

// Finding is a single scan result.
type Finding struct {
	ID       string `+"`json:\"id\"`"+`
	Severity string `+"`json:\"severity\"`"+`
	Title    string `+"`json:\"title\"`"+`
	Resource string `+"`json:\"resource\"`"+`
	Detail   string `+"`json:\"detail\"`"+`
}

// Summary counts findings by severity.
type Summary struct {
	Total    int `+"`json:\"total\"`"+`
	Critical int `+"`json:\"critical\"`"+`
	High     int `+"`json:\"high\"`"+`
	Medium   int `+"`json:\"medium\"`"+`
	Low      int `+"`json:\"low\"`"+`
}
`, name)
}

func scaffoldInitCmd(name string) string {
	return fmt.Sprintf(`package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := "%s.yaml"
		if _, err := os.Stat(configPath); err == nil {
			fmt.Fprintln(os.Stderr, "config already exists:", configPath)
			os.Exit(1)
		}
		if err := os.WriteFile(configPath, []byte("# %s configuration\n"), 0o644); err != nil {
			return err
		}
		fmt.Println("Created", configPath)
		return nil
	},
}
`, name, name)
}

func scaffoldCheckCmd(_ string) string {
	return `package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var format string

func init() {
	checkCmd.Flags().StringVar(&format, "format", "text", "output format: text, json")
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check health and report status",
	RunE: func(cmd *cobra.Command, args []string) error {
		result := CheckResult{
			Status: "healthy",
			Checks: []Check{
				{Name: "config", Status: "pass", Message: "configuration valid"},
			},
		}

		switch format {
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		default:
			fmt.Println("Status:", result.Status)
			for _, c := range result.Checks {
				fmt.Printf("  %s: %s (%s)\n", c.Name, c.Status, c.Message)
			}
			return nil
		}
	},
}

// CheckResult is the diagnostic output.
type CheckResult struct {
	Status string  ` + "`json:\"status\"`" + `
	Checks []Check ` + "`json:\"checks\"`" + `
}

// Check is a single health check result.
type Check struct {
	Name    string ` + "`json:\"name\"`" + `
	Status  string ` + "`json:\"status\"`" + `
	Message string ` + "`json:\"message\"`" + `
}
`
}

func scaffoldScanTest(name string) string {
	return fmt.Sprintf(`package commands

import (
	"encoding/json"
	"testing"
)

func TestScanResult_JSON(t *testing.T) {
	result := ScanResult{
		Version:  "spectre/v1",
		Scanner:  "%s",
		Target:   "test",
		Findings: []Finding{},
		Summary:  Summary{},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %%v", err)
	}
	var parsed ScanResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %%v", err)
	}
	if parsed.Version != "spectre/v1" {
		t.Errorf("version = %%q, want spectre/v1", parsed.Version)
	}
}
`, name)
}

func scaffoldCheckTest(_ string) string {
	return fmt.Sprintf(`package commands

import (
	"encoding/json"
	"testing"
)

func TestCheckResult_JSON(t *testing.T) {
	result := CheckResult{
		Status: "healthy",
		Checks: []Check{
			{Name: "test", Status: "pass", Message: "ok"},
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %%v", err)
	}
	var parsed CheckResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %%v", err)
	}
	if parsed.Status != "healthy" {
		t.Errorf("status = %%q, want healthy", parsed.Status)
	}
}
`)
}

func scaffoldMakefile(name string) string {
	return fmt.Sprintf(`.PHONY: build test lint fmt clean

build:
	go build -o bin/%s ./cmd/%s

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w cmd/ internal/
	goimports -w cmd/ internal/

clean:
	rm -rf bin/
`, name, name)
}

func scaffoldGitignore(name string) string {
	return fmt.Sprintf(`bin/
%s.yaml
*.log
.DS_Store
`, name)
}

func scaffoldReadme(name string) string {
	return fmt.Sprintf(`[![ANCC](https://img.shields.io/badge/ANCC-compliant-brightgreen)](https://ancc.dev)

# %s

One-line description of %s.

## Install

`+"```"+`
brew install yourorg/tap/%s
`+"```"+`

Or via Go:

`+"```"+`
go install github.com/yourorg/%s/cmd/%s@latest
`+"```"+`

## Usage

`+"```bash"+`
%s scan --format json
%s init
`+"```"+`

## License

MIT
`, name, name, name, name, name, name, name)
}

func scaffoldCI(_ string) string {
	return `name: CI
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: make build
      - run: make test
`
}

func scaffoldANCCWorkflow() string {
	return `name: ANCC
on: [push, pull_request]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ppiankov/ancc/.github/action@v0.7.0
        with:
          checks: validate
`
}

func scaffoldScannerSkillMD(name string) string {
	return fmt.Sprintf(`# %s

One-line description of %s.

## Install

`+"```"+`
brew install yourorg/tap/%s
`+"```"+`

Or via Go:

`+"```"+`
go install github.com/yourorg/%s/cmd/%s@latest
`+"```"+`

## Commands

### %s scan

Scans for findings and reports results.

**Flags:**
- `+"`--format json`"+` — output as JSON (spectre/v1 envelope)

**JSON output:**
`+"```json"+`
{
  "version": "spectre/v1",
  "scanner": "%s",
  "target": "target",
  "findings": [
    {
      "id": "FIND-001",
      "severity": "high",
      "title": "finding description",
      "resource": "resource identifier",
      "detail": "detailed explanation"
    }
  ],
  "summary": {
    "total": 1,
    "critical": 0,
    "high": 1,
    "medium": 0,
    "low": 0
  }
}
`+"```"+`

**Exit codes:**
- 0: scan complete, no findings
- 1: scan complete, findings detected
- 2: scan failed

### %s init

Initialize configuration with sensible defaults.

**Exit codes:**
- 0: config created
- 1: config already exists or error

## Handoffs

- Output: spectre/v1 JSON envelope. Next: aggregator (spectrehub) or enforcement tool.
- Refused questions: how to fix findings, whether to remediate, risk acceptance decisions.

## What this does NOT do

- Does not remediate or modify target systems — scan is read-only
- Does not store findings or manage a findings database
- Does not replace dedicated monitoring — point-in-time audit only

## Failure Modes

- Authentication failure: returns exit code 2. Distrust: all findings. Safe fallback: report scan failure.
- Network timeout: returns exit code 2. Distrust: completeness. Safe fallback: partial results with warning.

## Parsing examples

`+"```bash"+`
%s scan --format json | jq '.summary'
%s scan --format json | jq '.findings[] | select(.severity == "critical")'
`+"```"+`

---

This tool follows the [Agent-Native CLI Convention](https://ancc.dev). Validate with: `+"`ancc validate .`"+`
`, name, name, name, name, name, name, name, name, name, name)
}

func scaffoldDiagnosticSkillMD(name string) string {
	return fmt.Sprintf(`# %s

One-line description of %s.

## Install

`+"```"+`
brew install yourorg/tap/%s
`+"```"+`

Or via Go:

`+"```"+`
go install github.com/yourorg/%s/cmd/%s@latest
`+"```"+`

## Commands

### %s check

Checks health and reports status.

**Flags:**
- `+"`--format json`"+` — output as JSON

**JSON output:**
`+"```json"+`
{
  "status": "healthy",
  "checks": [
    {"name": "config", "status": "pass", "message": "configuration valid"},
    {"name": "connectivity", "status": "pass", "message": "reachable"}
  ]
}
`+"```"+`

**Exit codes:**
- 0: all checks pass
- 1: one or more checks failed
- 2: check could not run

### %s init

Initialize configuration with sensible defaults.

**Exit codes:**
- 0: config created
- 1: config already exists or error

## Handoffs

- Output: structured health JSON. Next: investigation tool for root cause analysis.
- Refused questions: why is it broken, should we fix it, is it safe to act.

## What this does NOT do

- Does not remediate or fix issues — diagnosis only
- Does not store historical health data or manage trends
- Does not replace dedicated monitoring — point-in-time check only

## Failure Modes

- Target unreachable: returns exit code 2. Distrust: all check results. Safe fallback: report unreachable.
- Partial connectivity: returns degraded status. Distrust: affected check results only.

## Parsing examples

`+"```bash"+`
%s check --format json | jq '.status'
%s check --format json | jq '.checks[] | select(.status == "fail")'
`+"```"+`

---

This tool follows the [Agent-Native CLI Convention](https://ancc.dev). Validate with: `+"`ancc validate .`"+`
`, name, name, name, name, name, name, name, name, name)
}
