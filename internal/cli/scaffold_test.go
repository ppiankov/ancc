package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCmd_Scanner(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-scanner")

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"scaffold", "my-scanner", "--type", "scanner"})

	// Change to temp dir so scaffold creates inside it.
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(orig) }()

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify key files exist.
	expectedFiles := []string{
		"go.mod",
		"Makefile",
		"README.md",
		".gitignore",
		"docs/SKILL.md",
		"cmd/my-scanner/main.go",
		"internal/commands/root.go",
		"internal/commands/scan.go",
		"internal/commands/initcmd.go",
		"internal/commands/scan_test.go",
		".github/workflows/ci.yml",
		".github/workflows/ancc.yml",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(outDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s not found", f)
		}
	}

	// Verify SKILL.md has ANCC breadcrumb.
	skill, err := os.ReadFile(filepath.Join(outDir, "docs", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	if !strings.Contains(string(skill), "Agent-Native CLI Convention") {
		t.Error("SKILL.md missing ANCC breadcrumb")
	}

	// Verify output message.
	output := buf.String()
	if !strings.Contains(output, "Scaffolded my-scanner") {
		t.Errorf("expected scaffold confirmation in output: %s", output)
	}
	if !strings.Contains(output, "Genesis loop") {
		t.Error("expected genesis loop hint in output")
	}
}

func TestScaffoldCmd_Diagnostic(t *testing.T) {
	dir := t.TempDir()

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"scaffold", "my-checker", "--type", "diagnostic"})

	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(orig) }()

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify diagnostic-specific files.
	outDir := filepath.Join(dir, "my-checker")
	checkGo := filepath.Join(outDir, "internal", "commands", "check.go")
	if _, err := os.Stat(checkGo); os.IsNotExist(err) {
		t.Error("expected check.go for diagnostic type")
	}

	// Should NOT have scan.go.
	scanGo := filepath.Join(outDir, "internal", "commands", "scan.go")
	if _, err := os.Stat(scanGo); err == nil {
		t.Error("diagnostic type should not have scan.go")
	}
}

func TestScaffoldCmd_DirectoryExists(t *testing.T) {
	dir := t.TempDir()
	_ = os.Mkdir(filepath.Join(dir, "existing"), 0o755)

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"scaffold", "existing"})

	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(orig) }()

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for existing directory")
	}
}

func TestScaffoldCmd_InvalidType(t *testing.T) {
	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"scaffold", "test-tool", "--type", "invalid"})

	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(orig) }()

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestScaffoldCmd_NoArgs(t *testing.T) {
	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"scaffold"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing name argument")
	}
}
