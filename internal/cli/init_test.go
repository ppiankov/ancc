package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ppiankov/ancc/internal/validator"
)

func TestInitCmd_CreatesTemplate(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Created docs/SKILL.md") {
		t.Errorf("expected success message, got %q", out)
	}

	if _, err := os.Stat(filepath.Join(dir, "docs", "SKILL.md")); err != nil {
		t.Fatal("docs/SKILL.md was not created")
	}
}

func TestInitCmd_CustomName(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init", "--name", "mytool"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "docs", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "# mytool") {
		t.Error("expected tool name in heading")
	}
	if !strings.Contains(content, "mytool run") {
		t.Error("expected 'mytool run' command")
	}
}

func TestInitCmd_RefuseOverwrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(origDir) }()

	// Create existing file.
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "SKILL.md"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init"})

	err := cmd.Execute()
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected ExitError code 1, got %v", err)
	}

	// Original content preserved.
	data, _ := os.ReadFile(filepath.Join(dir, "docs", "SKILL.md"))
	if string(data) != "existing" {
		t.Error("original file was overwritten")
	}
}

func TestInitCmd_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(origDir) }()

	// Create existing file.
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "SKILL.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init", "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "docs", "SKILL.md"))
	if string(data) == "old" {
		t.Error("file was not overwritten with --force")
	}
}

func TestInitCmd_ValidatesClean(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init", "--name", "testool"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init error: %v", err)
	}

	// Validate the generated SKILL.md.
	result, err := validator.Validate(dir)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}

	if result.Summary.Fail > 0 {
		for _, c := range result.Checks {
			if c.Status == validator.StatusFail {
				t.Errorf("FAIL: %s — %s", c.Name, c.Message)
			}
		}
		t.Fatalf("generated template has %d failures", result.Summary.Fail)
	}

	// Expect 29 pass + 3 warn (binary-release + failure-modes + changelog-exists).
	if result.Summary.Pass != 29 {
		t.Errorf("pass = %d, want 29", result.Summary.Pass)
	}
	if result.Summary.Warn != 3 {
		t.Errorf("warn = %d, want 3 (binary-release + failure-modes + changelog-exists)", result.Summary.Warn)
	}
}

func TestInitCmd_Help(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"init", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "--name") {
		t.Error("expected --name in help output")
	}
	if !strings.Contains(got, "--force") {
		t.Error("expected --force in help output")
	}
}
