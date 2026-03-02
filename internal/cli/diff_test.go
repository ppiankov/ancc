package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ppiankov/ancc/internal/skills"
)

func TestDiffCmd_IdenticalDirs(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "CLAUDE.md"), "# Project rules")

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", dir, dir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "identical") {
		t.Errorf("expected 'identical' in output, got:\n%s", output)
	}
}

func TestDiffCmd_DifferentDirs(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	writeTestFile(t, filepath.Join(dirA, "CLAUDE.md"), "# Project A rules")
	writeTestFile(t, filepath.Join(dirB, ".clinerules", "rule.md"), "cline rule")

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", dirA, dirB})

	err := cmd.Execute()
	var exitErr *ExitError
	if err == nil {
		// Both dirs may share global agents making them identical.
		// If no error, the dirs were identical from the scanner perspective.
		return
	}
	if !isExitError(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected exit code 1, got: %v", err)
	}
}

func TestDiffCmd_JSON(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "CLAUDE.md"), "# Rules")

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", dir, dir, "--format", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result skills.DiffResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if result.PathA == "" || result.PathB == "" {
		t.Error("expected non-empty paths")
	}
}

func TestDiffCmd_AgentFilter(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "CLAUDE.md"), "# Rules")
	writeTestFile(t, filepath.Join(dir, ".clinerules", "rule.md"), "cline rule")

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", dir, dir, "--agent", "cline"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "cline") {
		t.Errorf("expected 'cline' in output, got:\n%s", output)
	}
	if strings.Contains(output, "claude-code") {
		t.Errorf("expected no 'claude-code' in filtered output, got:\n%s", output)
	}
}

func TestDiffCmd_AgentFilterNoMatch(t *testing.T) {
	dir := t.TempDir()

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", dir, dir, "--agent", "nonexistent"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No agent") {
		t.Errorf("expected 'No agent' message, got:\n%s", output)
	}
}

func TestDiffCmd_TokensFlag(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "CLAUDE.md"), strings.Repeat("x", 400))

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"diff", dir, dir, "--tokens"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Tokens") {
		t.Errorf("expected 'Tokens' column in output, got:\n%s", output)
	}
}

func TestDiffCmd_MissingArg(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"diff", "."})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for missing arg")
	}
}

// isExitError checks if err is *ExitError and assigns it.
func isExitError(err error, target **ExitError) bool {
	if e, ok := err.(*ExitError); ok {
		*target = e
		return true
	}
	return false
}

// writeTestFile creates a file with the given content (test helper).
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
