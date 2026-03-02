package skills

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// mockLookPath returns a function that succeeds for listed binaries and fails for others.
func mockLookPath(found ...string) func(string) (string, error) {
	set := make(map[string]bool)
	for _, f := range found {
		set[f] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
}

// noopReadDir always returns not-exist, preventing environment checks from producing entries.
func noopReadDir(_ string) ([]os.DirEntry, error) {
	return nil, os.ErrNotExist
}

func testAuditEnv(found ...string) *auditEnv {
	return &auditEnv{lookPath: mockLookPath(found...), homeDir: "/mock/home", readDir: noopReadDir}
}

// --- expandTilde ---

func TestExpandTilde(t *testing.T) {
	if got := expandTilde("~/scripts/hook.sh", "/home/user"); got != "/home/user/scripts/hook.sh" {
		t.Errorf("got %q", got)
	}
	if got := expandTilde("/usr/bin/cmd", "/home/user"); got != "/usr/bin/cmd" {
		t.Errorf("got %q (should not change absolute)", got)
	}
	if got := expandTilde("cmd", "/home/user"); got != "cmd" {
		t.Errorf("got %q (should not change bare cmd)", got)
	}
	if got := expandTilde("~/scripts/hook.sh", ""); got != "~/scripts/hook.sh" {
		t.Errorf("got %q (should not expand without homeDir)", got)
	}
}

// --- extractCommand ---

func TestExtractCommand_String(t *testing.T) {
	raw := json.RawMessage(`"pastewatch-cli"`)
	if got := extractCommand(raw); got != "pastewatch-cli" {
		t.Errorf("got %q, want %q", got, "pastewatch-cli")
	}
}

func TestExtractCommand_Array(t *testing.T) {
	raw := json.RawMessage(`["pastewatch-cli", "mcp"]`)
	if got := extractCommand(raw); got != "pastewatch-cli" {
		t.Errorf("got %q, want %q", got, "pastewatch-cli")
	}
}

func TestExtractCommand_Nil(t *testing.T) {
	if got := extractCommand(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// --- parseMCPDetails ---

func TestParseMCPDetails_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"pastewatch": map[string]interface{}{"command": "pastewatch-cli"},
			"filesystem": map[string]interface{}{"command": "fs-server"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, path, string(data))

	details := parseMCPDetails(path, "mcpServers")
	if len(details) != 2 {
		t.Fatalf("got %d details, want 2", len(details))
	}
	// Sorted alphabetically.
	if details[0].name != "filesystem" || details[0].command != "fs-server" {
		t.Errorf("details[0] = %+v", details[0])
	}
	if details[1].name != "pastewatch" || details[1].command != "pastewatch-cli" {
		t.Errorf("details[1] = %+v", details[1])
	}
}

func TestParseMCPDetails_OpenCodeFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	cfg := map[string]interface{}{
		"mcp": map[string]interface{}{
			"pastewatch": map[string]interface{}{
				"type":    "local",
				"command": []string{"pastewatch-cli", "mcp"},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, path, string(data))

	details := parseMCPDetails(path, "mcp")
	if len(details) != 1 {
		t.Fatalf("got %d details, want 1", len(details))
	}
	if details[0].command != "pastewatch-cli" {
		t.Errorf("command = %q, want %q", details[0].command, "pastewatch-cli")
	}
}

func TestParseMCPDetails_Missing(t *testing.T) {
	details := parseMCPDetails("/nonexistent", "mcpServers")
	if details != nil {
		t.Errorf("expected nil, got %v", details)
	}
}

// --- parseCodexMCPDetails ---

func TestParseCodexMCPDetails_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `model = "gpt-5"

[mcp_servers.pastewatch]
command = "pastewatch-cli"
args = ["mcp"]

[mcp_servers.fs]
command = "fs-server"
`
	writeTestFile(t, path, content)

	details := parseCodexMCPDetails(path)
	if len(details) != 2 {
		t.Fatalf("got %d details, want 2", len(details))
	}
	if details[0].name != "pastewatch" || details[0].command != "pastewatch-cli" {
		t.Errorf("details[0] = %+v", details[0])
	}
	if details[1].name != "fs" || details[1].command != "fs-server" {
		t.Errorf("details[1] = %+v", details[1])
	}
}

func TestParseCodexMCPDetails_Missing(t *testing.T) {
	details := parseCodexMCPDetails("/nonexistent")
	if details != nil {
		t.Errorf("expected nil, got %v", details)
	}
}

// --- parseClaudeHookDetails ---

func TestParseClaudeHookDetails_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "bash-guard.sh"},
					},
				},
				map[string]interface{}{
					"matcher": "Write",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "pastewatch-guard"},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(settings)
	writeTestFile(t, path, string(data))

	details := parseClaudeHookDetails(path)
	if len(details) != 2 {
		t.Fatalf("got %d details, want 2", len(details))
	}
	// Sorted: PreToolUse/Bash before PreToolUse/Write.
	if details[0].event != "PreToolUse" || details[0].matcher != "Bash" {
		t.Errorf("details[0] = %+v", details[0])
	}
	if details[1].event != "PreToolUse" || details[1].matcher != "Write" {
		t.Errorf("details[1] = %+v", details[1])
	}
}

func TestParseClaudeHookDetails_Missing(t *testing.T) {
	details := parseClaudeHookDetails("/nonexistent")
	if details != nil {
		t.Errorf("expected nil, got %v", details)
	}
}

// --- auditMCPEntries ---

func TestAuditMCPEntries_FoundAndMissing(t *testing.T) {
	details := []mcpDetail{
		{name: "pastewatch", command: "pastewatch-cli"},
		{name: "broken", command: "nonexistent-binary"},
		{name: "nocmd"},
	}

	env := testAuditEnv("pastewatch-cli")
	entries := auditMCPEntries(details, "test-source", env)

	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if entries[0].Status != AuditOK {
		t.Errorf("pastewatch: status = %s, want ok", entries[0].Status)
	}
	if entries[1].Status != AuditError {
		t.Errorf("broken: status = %s, want error", entries[1].Status)
	}
	if entries[2].Status != AuditWarn {
		t.Errorf("nocmd: status = %s, want warn", entries[2].Status)
	}
}

// --- auditHookEntries ---

func TestAuditHookEntries_FoundAndMissing(t *testing.T) {
	details := []hookDetail{
		{event: "PreToolUse", matcher: "Bash", command: "bash-guard.sh"},
		{event: "PostToolUse", matcher: "Edit", command: "missing-hook"},
	}

	env := testAuditEnv("bash-guard.sh")
	entries := auditHookEntries(details, env)

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Status != AuditOK || entries[0].Name != "PreToolUse/Bash" {
		t.Errorf("entry[0] = %+v", entries[0])
	}
	if entries[1].Status != AuditError || entries[1].Name != "PostToolUse/Edit" {
		t.Errorf("entry[1] = %+v", entries[1])
	}
}

func TestAuditHookEntries_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "my-hook.sh")
	writeTestFile(t, script, "#!/bin/sh\nexit 0\n")

	details := []hookDetail{
		{event: "PreToolUse", matcher: "Bash", command: script},
		{event: "PreToolUse", matcher: "Write", command: "/nonexistent/hook.sh"},
	}

	env := testAuditEnv() // no PATH lookups needed for absolute paths
	entries := auditHookEntries(details, env)

	if entries[0].Status != AuditOK {
		t.Errorf("existing absolute path: status = %s, want ok", entries[0].Status)
	}
	if entries[1].Status != AuditError {
		t.Errorf("missing absolute path: status = %s, want error", entries[1].Status)
	}
}

func TestAuditHookEntries_TildeExpansion(t *testing.T) {
	home := t.TempDir()
	script := filepath.Join(home, "hooks", "guard.sh")
	writeTestFile(t, script, "#!/bin/sh\nexit 0\n")

	details := []hookDetail{
		{event: "PreToolUse", matcher: "Bash", command: "~/hooks/guard.sh"},
	}

	env := &auditEnv{lookPath: mockLookPath(), homeDir: home}
	entries := auditHookEntries(details, env)

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Status != AuditOK {
		t.Errorf("tilde-expanded hook: status = %s, want ok (path: %s)", entries[0].Status, script)
	}
}

// --- auditSkillDirs ---

func TestAuditSkillDirs_MixedState(t *testing.T) {
	dir := t.TempDir()

	// Non-empty skill.
	mkdirAll(t, filepath.Join(dir, "commit"))
	writeTestFile(t, filepath.Join(dir, "commit", "prompt.md"), "do the commit")

	// Empty skill.
	mkdirAll(t, filepath.Join(dir, "empty-skill"))

	entries := auditSkillDirs(dir, "test-source")
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	// Sort by name to ensure stable order.
	var commitEntry, emptyEntry AuditEntry
	for _, e := range entries {
		switch e.Name {
		case "commit":
			commitEntry = e
		case "empty-skill":
			emptyEntry = e
		}
	}

	if commitEntry.Status != AuditOK {
		t.Errorf("commit: status = %s, want ok", commitEntry.Status)
	}
	if emptyEntry.Status != AuditWarn {
		t.Errorf("empty-skill: status = %s, want warn", emptyEntry.Status)
	}
}

func TestAuditSkillDirs_NonExistent(t *testing.T) {
	entries := auditSkillDirs("/nonexistent", "test")
	if entries != nil {
		t.Errorf("expected nil, got %v", entries)
	}
}

func TestAuditSkillDirs_Symlinks(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir()
	mkdirAll(t, filepath.Join(target, "real-skill"))
	writeTestFile(t, filepath.Join(target, "real-skill", "prompt.md"), "content")

	skillsDir := filepath.Join(dir, "skills")
	mkdirAll(t, skillsDir)
	if err := os.Symlink(filepath.Join(target, "real-skill"), filepath.Join(skillsDir, "linked-skill")); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	entries := auditSkillDirs(skillsDir, "test")
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Status != AuditOK {
		t.Errorf("status = %s, want ok", entries[0].Status)
	}
}

// --- auditSkillFiles ---

func TestAuditSkillFiles_MixedState(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "rule.md"), "good rule")
	writeTestFile(t, filepath.Join(dir, "empty.md"), "")

	entries := auditSkillFiles(dir, "test")
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	var goodEntry, emptyEntry AuditEntry
	for _, e := range entries {
		switch e.Name {
		case "rule.md":
			goodEntry = e
		case "empty.md":
			emptyEntry = e
		}
	}

	if goodEntry.Status != AuditOK {
		t.Errorf("rule.md: status = %s, want ok", goodEntry.Status)
	}
	if emptyEntry.Status != AuditWarn {
		t.Errorf("empty.md: status = %s, want warn", emptyEntry.Status)
	}
}

// --- Per-agent audit functions ---

func TestAuditClaudeCode_WithConfig(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	// Settings with hook and MCP.
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": "bash-guard"}},
				},
			},
		},
		"mcpServers": map[string]interface{}{
			"pastewatch": map[string]interface{}{"command": "pastewatch-cli"},
		},
	}
	data, _ := json.Marshal(settings)
	writeTestFile(t, filepath.Join(home, ".claude", "settings.json"), string(data))

	// Global skill.
	mkdirAll(t, filepath.Join(home, ".claude", "skills", "commit"))
	writeTestFile(t, filepath.Join(home, ".claude", "skills", "commit", "prompt.md"), "content")

	env := testAuditEnv("bash-guard", "pastewatch-cli")
	a := auditClaudeCode(proj, home, env)

	if len(a.Entries) != 3 { // 1 hook + 1 MCP + 1 skill
		t.Errorf("entries = %d, want 3", len(a.Entries))
	}
	for _, e := range a.Entries {
		if e.Status != AuditOK {
			t.Errorf("entry %s/%s: status = %s, want ok", e.Category, e.Name, e.Status)
		}
	}
}

func TestAuditClaudeCode_NoConfig(t *testing.T) {
	env := testAuditEnv()
	a := auditClaudeCode(t.TempDir(), t.TempDir(), env)
	if len(a.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(a.Entries))
	}
}

func TestAuditCline_WithSkills(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	mkdirAll(t, filepath.Join(home, ".cline", "skills", "commit"))
	writeTestFile(t, filepath.Join(home, ".cline", "skills", "commit", "prompt.md"), "content")

	env := testAuditEnv()
	a := auditCline(proj, home, env)
	if len(a.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(a.Entries))
	}
}

func TestAuditCodex_HomeMCP(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.pw]\ncommand = \"pastewatch-cli\"\n")

	env := testAuditEnv("pastewatch-cli")
	a := auditCodex(proj, home, env)

	var found bool
	for _, e := range a.Entries {
		if e.Category == "mcp" && e.Name == "pw" && e.Status == AuditOK {
			found = true
		}
	}
	if !found {
		t.Error("expected MCP audit entry for pastewatch")
	}
}

// --- AuditWithHome (integration) ---

func TestAuditWithHome_EmptyDir(t *testing.T) {
	env := testAuditEnv()
	result, err := AuditWithHome(t.TempDir(), t.TempDir(), env)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Agents) != 0 {
		t.Errorf("agents = %d, want 0", len(result.Agents))
	}
	// Environment checks produce 10 entries (all "not present" = ok).
	if len(result.Environment) != 10 {
		t.Errorf("environment = %d, want 10", len(result.Environment))
	}
	if result.Summary.Total != 10 {
		t.Errorf("total = %d, want 10", result.Summary.Total)
	}
	if result.Summary.OK != 10 {
		t.Errorf("ok = %d, want 10", result.Summary.OK)
	}
}

func TestAuditWithHome_SummaryCount(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	// Claude: 1 hook (found) + 1 MCP (missing).
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": "bash-guard"}},
				},
			},
		},
		"mcpServers": map[string]interface{}{
			"broken": map[string]interface{}{"command": "nonexistent-mcp"},
		},
	}
	data, _ := json.Marshal(settings)
	writeTestFile(t, filepath.Join(home, ".claude", "settings.json"), string(data))

	env := testAuditEnv("bash-guard")
	result, err := AuditWithHome(proj, home, env)
	if err != nil {
		t.Fatal(err)
	}

	// 1 hook ok + 10 environment ok = 11 ok.
	if result.Summary.OK != 11 {
		t.Errorf("ok = %d, want 11", result.Summary.OK)
	}
	if result.Summary.Errors != 1 {
		t.Errorf("errors = %d, want 1", result.Summary.Errors)
	}
}
