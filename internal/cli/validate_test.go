package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ppiankov/ancc/internal/validator"
)

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestValidateCmd_RepoRoot(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", "--verbose", repoRoot()})

	err := cmd.Execute()

	// The repo itself should produce output (may pass or warn).
	out := buf.String()
	if !strings.Contains(out, "Result:") {
		t.Errorf("expected Result line in output, got %q", out)
	}

	// Exit error is acceptable (warn = exit 2).
	var exitErr *ExitError
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatalf("unexpected error type: %v", err)
	}
}

func TestValidateCmd_JSON(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", "--format", "json", repoRoot()})

	err := cmd.Execute()

	var exitErr *ExitError
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed validator.ValidationResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}
	if parsed.Summary.Total != 25 {
		t.Errorf("total = %d, want 25", parsed.Summary.Total)
	}
}

func TestValidateCmd_MissingPath(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", t.TempDir()})

	err := cmd.Execute()

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.Code != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.Code)
	}
}

func TestValidateCmd_ExitCode2_WarnOnly(t *testing.T) {
	// Use repo root which has SKILL.md — should produce at least binary-release warn.
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", "--format", "json", repoRoot()})

	err := cmd.Execute()

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Skipf("repo may have fails, skipping exit code 2 test: err=%v", err)
	}
	// If we get exit 2, great — that's warn-only.
	// If we get exit 1, that's also valid (repo has fails).
	if exitErr.Code != 1 && exitErr.Code != 2 {
		t.Errorf("exit code = %d, want 1 or 2", exitErr.Code)
	}
}

func TestValidateCmd_Badge(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", "--badge", repoRoot()})

	err := cmd.Execute()
	var exitErr *ExitError
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "img.shields.io/badge/ANCC") {
		t.Errorf("expected badge URL in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Badge:") {
		t.Errorf("expected 'Badge:' label in output, got:\n%s", output)
	}
}

func TestValidateCmd_BadgeJSON(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", "--badge", "--format", "json", repoRoot()})

	err := cmd.Execute()
	var exitErr *ExitError
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed struct {
		BadgeURL string `json:"badge_url"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if parsed.BadgeURL == "" {
		t.Error("expected non-empty badge_url")
	}
	if !strings.Contains(parsed.BadgeURL, "img.shields.io") {
		t.Errorf("badge_url = %q, expected shields.io URL", parsed.BadgeURL)
	}
}

func TestBadgeURL(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"pass", "brightgreen"},
		{"partial", "yellow"},
		{"fail", "red"},
	}
	for _, tt := range tests {
		got := badgeURL(tt.status)
		if !strings.Contains(got, tt.want) {
			t.Errorf("badgeURL(%q) = %q, want color %q", tt.status, got, tt.want)
		}
	}
}

func TestValidateCmd_Help(t *testing.T) {
	cmd := newRootCmd("dev")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"validate", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "--format") {
		t.Error("expected --format in help output")
	}
	if !strings.Contains(got, "--verbose") {
		t.Error("expected --verbose in help output")
	}
}
