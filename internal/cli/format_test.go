package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ppiankov/ancc/internal/validator"
)

func sampleResult() *validator.ValidationResult {
	return &validator.ValidationResult{
		Path:   "/tmp/test",
		Status: validator.OverallFail,
		Checks: []validator.CheckResult{
			{Name: validator.CheckSkillMDExists, Status: validator.StatusPass, Message: "SKILL.md found"},
			{Name: validator.CheckSkillMDInstall, Status: validator.StatusPass, Message: "Install section found"},
			{Name: validator.CheckSkillMDExitCodes, Status: validator.StatusFail, Message: "missing exit codes section"},
			{Name: validator.CheckHasBinaryRelease, Status: validator.StatusWarn, Message: "skipped"},
		},
		Summary: validator.Summary{Total: 4, Pass: 2, Fail: 1, Warn: 1},
	}
}

func TestFormatText_NonVerbose(t *testing.T) {
	buf := new(bytes.Buffer)
	formatText(buf, sampleResult(), false)
	out := buf.String()

	// Non-verbose should hide passing checks.
	if strings.Contains(out, "SKILL.md exists") {
		t.Error("non-verbose should hide passing checks")
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("expected FAIL in output")
	}
	if !strings.Contains(out, "WARN") {
		t.Error("expected WARN in output")
	}
	if !strings.Contains(out, "Result: FAIL") {
		t.Error("expected Result: FAIL summary line")
	}
	if !strings.Contains(out, "2 pass") {
		t.Error("expected pass count in summary")
	}
}

func TestFormatText_Verbose(t *testing.T) {
	buf := new(bytes.Buffer)
	formatText(buf, sampleResult(), true)
	out := buf.String()

	// Verbose should show all checks.
	if !strings.Contains(out, "SKILL.md exists") {
		t.Error("verbose should show passing checks")
	}
	if !strings.Contains(out, "PASS") {
		t.Error("expected PASS in verbose output")
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("expected FAIL in verbose output")
	}
}

func TestFormatText_AllPass(t *testing.T) {
	r := &validator.ValidationResult{
		Path:   "/tmp/test",
		Status: validator.OverallPass,
		Checks: []validator.CheckResult{
			{Name: validator.CheckSkillMDExists, Status: validator.StatusPass, Message: "found"},
		},
		Summary: validator.Summary{Total: 1, Pass: 1},
	}
	buf := new(bytes.Buffer)
	formatText(buf, r, false)
	out := buf.String()

	if !strings.Contains(out, "Result: PASS") {
		t.Errorf("expected Result: PASS, got %q", out)
	}
}

func TestFormatJSON_ValidOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := formatJSON(buf, sampleResult()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed validator.ValidationResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed.Status != validator.OverallFail {
		t.Errorf("status = %q, want %q", parsed.Status, validator.OverallFail)
	}
	if len(parsed.Checks) != 4 {
		t.Errorf("got %d checks, want 4", len(parsed.Checks))
	}
	if parsed.Summary.Fail != 1 {
		t.Errorf("fail = %d, want 1", parsed.Summary.Fail)
	}
}

func TestFormatText_FailMessage(t *testing.T) {
	buf := new(bytes.Buffer)
	formatText(buf, sampleResult(), false)
	out := buf.String()

	if !strings.Contains(out, "missing exit codes section") {
		t.Error("expected failure message in output")
	}
}
