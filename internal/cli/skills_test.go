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

func TestFormatSkillsTextShowsInvalidLocations(t *testing.T) {
	result := &skills.ScanResult{
		InvalidLocations: []skills.InvalidLocation{
			{
				Agent:  skills.AgentAntigravity,
				Path:   "./.antigravitycli/skills/scratch",
				Reason: "missing required file SKILL.md",
			},
		},
	}

	buf := new(bytes.Buffer)
	formatSkillsText(buf, result, false, 0)
	output := buf.String()

	for _, want := range []string{
		"No agent configurations found.",
		"Invalid locations:",
		skills.AgentAntigravity,
		"./.antigravitycli/skills/scratch",
		"missing required file SKILL.md",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q; got: %s", want, output)
		}
	}
}

func TestFormatSkillsTextShowsEnforcementPosture(t *testing.T) {
	result := &skills.ScanResult{
		Agents: []skills.AgentResult{
			{
				Name:        skills.AgentAntigravity,
				Enforcement: skills.EnforcementAdvisory,
				Warning:     skills.SecurityProbeSelfReportWarning,
				Sources:     []string{"AGENTS.md"},
			},
			{
				Name:        skills.AgentCline,
				Enforcement: skills.EnforcementUnverified,
				Sources:     []string{".clinerules/"},
			},
			{
				Name:        skills.AgentAider,
				Enforcement: skills.EnforcementUnverified,
				Autonomy: []skills.AutonomyCapability{
					{
						Mode:       "--yes-always",
						Disables:   "confirmation prompts",
						SourceKind: skills.AutonomySourceVendorDocs,
						Source:     "Aider options reference",
					},
				},
				Sources: []string{".aider.conf.yml"},
			},
			{
				Name:        skills.AgentCodex,
				Enforcement: skills.EnforcementEnforcing,
				Evidence: []skills.EvidenceItem{
					{Kind: skills.EvidenceRealToolResult, Note: "policy blocked a real write attempt"},
				},
				Sources: []string{"~/.codex/AGENTS.md"},
			},
		},
	}

	buf := new(bytes.Buffer)
	formatSkillsText(buf, result, false, 0)
	output := buf.String()

	for _, want := range []string{
		"Posture",
		skills.AgentAntigravity,
		string(skills.EnforcementAdvisory),
		"Antigravity's workspace trust is not an enforcement boundary.",
		"agent self-reports as proof of access or enforcement",
		"valid:   real OS result, real tool error, unfakeable payload",
		"invalid: vendor docs, agent says \"YES\", model explanation",
		"Autonomy:",
		"Mode",
		"Source",
		"--yes-always",
		"vendor_docs",
		"confirmation prompts",
		"Aider options reference",
		skills.AgentCline,
		string(skills.EnforcementUnverified),
		skills.AgentAider,
		skills.AgentCodex,
		string(skills.EnforcementEnforcing),
		"policy blocked a real write attempt",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q; got: %s", want, output)
		}
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, skills.AgentCline) && strings.Contains(line, "warning") {
			t.Errorf("unverified cline line should not include warning: %q", line)
		}
	}
}

func TestFormatSkillsJSONOmitsPlainUnverifiedDetails(t *testing.T) {
	agent := skills.AgentResult{
		Name:        skills.AgentCodex,
		Enforcement: skills.EnforcementAdvisory,
		Warning:     skills.SecurityProbeSelfReportWarning,
		Autonomy: []skills.AutonomyCapability{
			{
				Mode:       "--full-auto",
				Disables:   "edit and command approval prompts within the selected sandbox mode",
				SourceKind: skills.AutonomySourceVendorDocs,
				Source:     "OpenAI Codex CLI approval modes docs",
			},
		},
		Evidence: []skills.EvidenceItem{
			{Kind: skills.EvidenceVendorDocs, Note: "vendor docs claim secure mode"},
			{Kind: skills.EvidenceAgentSelfReport, Note: "agent said YES"},
		},
	}
	agent.NormalizeEnforcement()

	result := &skills.ScanResult{
		Path:   "/tmp/project",
		Agents: []skills.AgentResult{agent},
	}

	buf := new(bytes.Buffer)
	if err := formatSkillsJSON(buf, result, 0); err != nil {
		t.Fatalf("formatSkillsJSON returned error: %v", err)
	}

	var raw struct {
		Agents []struct {
			Name        string                      `json:"name"`
			Enforcement skills.Enforcement          `json:"enforcement"`
			Autonomy    []skills.AutonomyCapability `json:"autonomy"`
			Evidence    json.RawMessage             `json:"evidence"`
			Warning     string                      `json:"warning"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if len(raw.Agents) != 1 {
		t.Fatalf("agents = %d, want 1", len(raw.Agents))
	}
	if raw.Agents[0].Enforcement != skills.EnforcementUnverified {
		t.Fatalf("enforcement = %q, want %q", raw.Agents[0].Enforcement, skills.EnforcementUnverified)
	}
	if len(raw.Agents[0].Evidence) != 0 || raw.Agents[0].Warning != "" {
		t.Fatalf("plain unverified JSON should omit evidence/warning, got raw=%s warning=%q",
			raw.Agents[0].Evidence, raw.Agents[0].Warning)
	}
	if len(raw.Agents[0].Autonomy) != 1 || raw.Agents[0].Autonomy[0].Mode != "--full-auto" {
		t.Fatalf("unverified JSON should retain autonomy, got %+v", raw.Agents[0].Autonomy)
	}
}

func TestFormatSkillsJSONIncludesEnforcementPosture(t *testing.T) {
	evidence := []skills.EvidenceItem{
		{Kind: skills.EvidenceRealToolResult, Note: "trustedWorkspaces does not confine reads to workspace"},
	}
	result := &skills.ScanResult{
		Path: "/tmp/project",
		Agents: []skills.AgentResult{
			{
				Name:        skills.AgentAntigravity,
				Enforcement: skills.EnforcementAdvisory,
				Evidence:    evidence,
				Warning:     skills.SecurityProbeSelfReportWarning,
			},
		},
	}

	buf := new(bytes.Buffer)
	if err := formatSkillsJSON(buf, result, 0); err != nil {
		t.Fatalf("formatSkillsJSON returned error: %v", err)
	}

	var raw struct {
		Agents []struct {
			Name        string                `json:"name"`
			Enforcement skills.Enforcement    `json:"enforcement"`
			Evidence    []skills.EvidenceItem `json:"evidence"`
			Warning     string                `json:"warning"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if len(raw.Agents) != 1 {
		t.Fatalf("agents = %d, want 1", len(raw.Agents))
	}
	if raw.Agents[0].Enforcement != skills.EnforcementAdvisory {
		t.Fatalf("enforcement = %q, want %q", raw.Agents[0].Enforcement, skills.EnforcementAdvisory)
	}
	if len(raw.Agents[0].Evidence) != 1 || raw.Agents[0].Evidence[0] != evidence[0] {
		t.Fatalf("evidence = %+v, want %+v", raw.Agents[0].Evidence, evidence)
	}
	if raw.Agents[0].Warning != skills.SecurityProbeSelfReportWarning {
		t.Fatalf("warning = %q, want %q", raw.Agents[0].Warning, skills.SecurityProbeSelfReportWarning)
	}
}

func TestFormatSkillsJSONWithBudgetIncludesInvalidLocations(t *testing.T) {
	result := &skills.ScanResult{
		Path: "/tmp/project",
		InvalidLocations: []skills.InvalidLocation{
			{
				Agent:  skills.AgentAntigravity,
				Path:   "~/.gemini/antigravity-cli/skills/scratch",
				Reason: "missing required file SKILL.md",
			},
		},
	}

	buf := new(bytes.Buffer)
	if err := formatSkillsJSON(buf, result, 200000); err != nil {
		t.Fatalf("formatSkillsJSON returned error: %v", err)
	}

	var raw struct {
		InvalidLocations []skills.InvalidLocation `json:"invalid_locations"`
	}
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if len(raw.InvalidLocations) != 1 {
		t.Fatalf("expected 1 invalid location, got %d", len(raw.InvalidLocations))
	}
	if raw.InvalidLocations[0].Path != "~/.gemini/antigravity-cli/skills/scratch" {
		t.Errorf("unexpected invalid location: %+v", raw.InvalidLocations[0])
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
