package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func setupExportProject(t *testing.T) string {
	t.Helper()
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".clinerules"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("x", 200)
	if err := os.WriteFile(filepath.Join(proj, ".clinerules", "rule.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return proj
}

func TestExportCmd_JSONStructure(t *testing.T) {
	proj := setupExportProject(t)

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"export", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out exportOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}

	if len(out.Agents) == 0 {
		t.Fatal("expected at least one agent in export output")
	}

	// Verify expected fields are present.
	found := false
	for _, a := range out.Agents {
		if a.Name == "cline" {
			found = true
			if a.ConfigDir == "" {
				t.Error("expected config_dir for cline")
			}
			if a.Tokens == 0 {
				t.Error("expected non-zero tokens for cline")
			}
			if len(a.Sources) == 0 {
				t.Error("expected non-empty sources for cline")
			}
			if len(a.SkillFiles) == 0 {
				t.Error("expected non-empty skill_files for cline")
			}
		}
	}
	if !found {
		t.Error("expected cline agent in export output")
	}
}

func TestExportCmd_AgentFilter(t *testing.T) {
	proj := setupExportProject(t)

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"export", "--agent", "cline", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out exportOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}

	if len(out.Agents) != 1 {
		t.Fatalf("expected exactly 1 agent, got %d", len(out.Agents))
	}
	if out.Agents[0].Name != "cline" {
		t.Errorf("expected agent name 'cline', got %q", out.Agents[0].Name)
	}
	if out.Agents[0].ConfigDir == "" {
		t.Error("expected config_dir to be set")
	}
}

func TestExportCmd_AgentFilterNotFound(t *testing.T) {
	proj := setupExportProject(t)

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"export", "--agent", "nonexistent", proj})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent agent filter")
	}
}

func TestExportCmd_YAMLFormat(t *testing.T) {
	proj := setupExportProject(t)

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"export", "--format", "yaml", proj})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out exportOutput
	if err := yaml.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid YAML: %v\nraw: %s", err, buf.String())
	}

	if len(out.Agents) == 0 {
		t.Fatal("expected at least one agent in YAML output")
	}

	found := false
	for _, a := range out.Agents {
		if a.Name == "cline" {
			found = true
			if a.ConfigDir == "" {
				t.Error("expected config_dir for cline in YAML output")
			}
		}
	}
	if !found {
		t.Error("expected cline agent in YAML output")
	}
}

func TestExportCmd_EmptyProject(t *testing.T) {
	// Empty temp dir — no project-level agent configs.
	proj := t.TempDir()

	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"export", "--agent", "nonexistent", proj})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent agent in empty project")
	}
}

func TestExportCmd_NonExistentPath(t *testing.T) {
	// Scan doesn't fail on nonexistent paths - it just returns empty results.
	// However, it still scans the home directory, so we'll get agents from there.
	// The test should verify the command succeeds (doesn't error).
	cmd := newRootCmd("test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"export", "/nonexistent/path/that/does/not/exist"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error for nonexistent path: %v", err)
	}

	var out exportOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}

	// Valid JSON is sufficient — agents may be nil on CI where no home configs exist.
}
