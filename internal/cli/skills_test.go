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

func TestSkillsCmd_TokensFlag(t *testing.T) {
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("x", 400) // 400 bytes
	if err := os.WriteFile(filepath.Join(proj, ".clinerules", "rule.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"skills", "--tokens", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Tokens") {
		t.Error("expected Tokens column header in output")
	}
	if strings.Contains(output, "Total context tax") {
		t.Error("total line should not appear — agents load independently")
	}
}

func TestSkillsCmd_NoTokensFlag(t *testing.T) {
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, ".clinerules", "rule.md"), []byte("rule"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"skills", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "Tokens") {
		t.Error("Tokens column should be hidden without --tokens flag")
	}
}

func TestSkillsCmd_JSONIncludesTokens(t *testing.T) {
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("x", 80)
	if err := os.WriteFile(filepath.Join(proj, ".clinerules", "rule.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"skills", "--format", "json", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result skills.ScanResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}

	var clineAgent *skills.AgentResult
	for i := range result.Agents {
		if result.Agents[i].Name == "cline" {
			clineAgent = &result.Agents[i]
			break
		}
	}
	if clineAgent == nil {
		t.Fatal("expected cline agent in results")
	}
	if clineAgent.Tokens == 0 {
		t.Error("expected non-zero cline tokens")
	}
}

func TestSkillsCmd_BudgetFlag(t *testing.T) {
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("x", 400)
	if err := os.WriteFile(filepath.Join(proj, ".clinerules", "rule.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"skills", "--budget", "200000", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Budget") {
		t.Error("expected Budget column header in output")
	}
	if !strings.Contains(output, "Tokens") {
		t.Error("--budget should imply --tokens")
	}
	if !strings.Contains(output, "%") {
		t.Error("expected percentage in output")
	}
}

func TestSkillsCmd_BudgetJSON(t *testing.T) {
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("x", 800)
	if err := os.WriteFile(filepath.Join(proj, ".clinerules", "rule.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"skills", "--budget", "200000", "--format", "json", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify budget_pct is in JSON output.
	if !strings.Contains(buf.String(), "budget_pct") {
		t.Errorf("expected budget_pct in JSON output; got: %s", buf.String())
	}

	// Parse and verify structure.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	var agents []map[string]interface{}
	if err := json.Unmarshal(raw["agents"], &agents); err != nil {
		t.Fatalf("invalid agents JSON: %v", err)
	}
	if len(agents) == 0 {
		t.Fatal("expected at least one agent")
	}
	if _, ok := agents[0]["budget_pct"]; !ok {
		t.Error("expected budget_pct field in agent JSON")
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "~0"},
		{5, "~5"},
		{100, "~100"},
		{999, "~999"},
		{1000, "~1,000"},
		{3400, "~3,400"},
		{12500, "~12,500"},
		{1000000, "~1,000,000"},
	}
	for _, tt := range tests {
		if got := formatTokenCount(tt.input); got != tt.want {
			t.Errorf("formatTokenCount(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
