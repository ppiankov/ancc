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

func TestContextCmd_TextOutput(t *testing.T) {
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
	cmd.SetArgs([]string{"context", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	for _, col := range []string{"Agent", "Config", "Window", "Available", "Config %"} {
		if !strings.Contains(output, col) {
			t.Errorf("expected %q column header in output", col)
		}
	}
	if !strings.Contains(output, "cline") {
		t.Error("expected cline agent in output")
	}
	if !strings.Contains(output, "128,000") {
		t.Error("expected default window size 128,000 in output")
	}
}

func TestContextCmd_NoAgents(t *testing.T) {
	// Use buildContextAgents directly with empty scan result to test the
	// empty-agents path. The CLI scanner picks up home-dir agents, so
	// we can't get an empty result through the CLI in a real environment.
	result := &skills.ScanResult{}
	agents := buildContextAgents(result, "", 0)
	if len(agents) != 0 {
		t.Errorf("expected 0 agents from empty scan, got %d", len(agents))
	}

	// Verify text formatter handles empty list.
	buf := new(bytes.Buffer)
	formatContextText(buf, nil)
	if !strings.Contains(buf.String(), "No agent configurations found.") {
		t.Error("expected 'No agent configurations found.' message")
	}
}

func TestContextCmd_JSONOutput(t *testing.T) {
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
	cmd.SetArgs([]string{"context", "--format", "json", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result contextResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}

	if len(result.Agents) == 0 {
		t.Fatal("expected at least one agent")
	}

	// Find cline agent by name (order depends on home-dir agents).
	var cline *contextAgent
	for i := range result.Agents {
		if result.Agents[i].Name == "cline" {
			cline = &result.Agents[i]
			break
		}
	}
	if cline == nil {
		t.Fatal("expected cline agent in results")
	}
	if cline.ConfigTokens == 0 {
		t.Error("expected non-zero config_tokens")
	}
	if cline.ContextWindow != 128_000 {
		t.Errorf("expected context_window 128000, got %d", cline.ContextWindow)
	}
	if cline.AvailableTokens <= 0 {
		t.Error("expected positive available_tokens")
	}
	if cline.ConfigPercent <= 0 {
		t.Error("expected positive config_percent")
	}
}

func TestContextCmd_AgentFilter(t *testing.T) {
	proj := t.TempDir()
	// Set up cline config.
	if err := os.MkdirAll(filepath.Join(proj, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, ".clinerules", "rule.md"), []byte("rule"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"context", "--agent", "cline", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "cline") {
		t.Error("expected cline in filtered output")
	}
}

func TestContextCmd_AgentFilterNoMatch(t *testing.T) {
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
	cmd.SetArgs([]string{"context", "--agent", "nonexistent", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "No agent configurations found.") {
		t.Error("expected no agents found for nonexistent filter")
	}
}

func TestContextCmd_WindowOverride(t *testing.T) {
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
	cmd.SetArgs([]string{"context", "--window", "200000", "--format", "json", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result contextResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result.Agents) == 0 {
		t.Fatal("expected at least one agent")
	}
	if result.Agents[0].ContextWindow != 200_000 {
		t.Errorf("expected context_window 200000 with override, got %d", result.Agents[0].ContextWindow)
	}
}

func TestContextCmd_AvailableNonNegative(t *testing.T) {
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write enough content to exceed a tiny window.
	content := strings.Repeat("x", 4000)
	if err := os.WriteFile(filepath.Join(proj, ".clinerules", "rule.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"context", "--window", "100", "--format", "json", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result contextResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result.Agents) == 0 {
		t.Fatal("expected at least one agent")
	}
	if result.Agents[0].AvailableTokens < 0 {
		t.Errorf("available_tokens should not be negative, got %d", result.Agents[0].AvailableTokens)
	}
}
