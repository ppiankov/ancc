package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ppiankov/ancc/internal/validator"
)

func makeFakeRepo(t *testing.T, parent, name string) {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestScanCmd_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"scan", dir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "No repos found") {
		t.Errorf("output = %q, want 'No repos found'", buf.String())
	}
}

func TestScanCmd_JSON(t *testing.T) {
	dir := t.TempDir()
	makeFakeRepo(t, dir, "testrepo")

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"scan", "--format", "json", dir})

	err := cmd.Execute()

	// May have ExitError due to missing SKILL.md.
	var exitErr *ExitError
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatalf("unexpected error: %v", err)
	}

	var result validator.ScanResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if result.Summary.Total != 1 {
		t.Errorf("total = %d, want 1", result.Summary.Total)
	}
	if result.Repos[0].Status != "missing" {
		t.Errorf("status = %q, want %q", result.Repos[0].Status, "missing")
	}
}

func TestScanCmd_DepthFlag(t *testing.T) {
	dir := t.TempDir()
	// Create repo at depth 3.
	deep := filepath.Join(dir, "a", "b", "deep")
	if err := os.MkdirAll(filepath.Join(deep, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"scan", "--depth", "1", dir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "No repos found") {
		t.Errorf("depth=1 should not find repo at depth 3; output = %q", buf.String())
	}
}

func TestScanCmd_ExitCode(t *testing.T) {
	dir := t.TempDir()
	makeFakeRepo(t, dir, "noskill")

	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"scan", dir})

	err := cmd.Execute()

	// Missing repos don't cause exit code 1 — only fails do.
	if err != nil {
		t.Errorf("expected no error for missing-only scan, got: %v", err)
	}
}
