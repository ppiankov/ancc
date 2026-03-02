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

// noopReadDir always returns not-exist, preventing directory checks from producing warnings.
func noopReadDir(_ string) ([]os.DirEntry, error) {
	return nil, os.ErrNotExist
}

// noopStat always returns not-exist, preventing file checks from producing warnings.
func noopStat(_ string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func testAuditEnv(found ...string) *auditEnv {
	return &auditEnv{
		lookPath: mockLookPath(found...),
		homeDir:  "/mock/home",
		readDir:  noopReadDir,
		stat:     noopStat,
		goos:     "darwin",
	}
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
	// Environment checks: 7 sensitive + 7 credential + 5 history + 6 credential-file = 25 (darwin).
	if len(result.Environment) != 25 {
		t.Errorf("environment = %d, want 25", len(result.Environment))
	}
	if result.Summary.Total != 25 {
		t.Errorf("total = %d, want 25", result.Summary.Total)
	}
	if result.Summary.OK != 25 {
		t.Errorf("ok = %d, want 25", result.Summary.OK)
	}
}

// --- auditBudget ---

func TestAuditBudget_Warning(t *testing.T) {
	// 11% of 128K = 14,080 tokens → should trigger config-budget-warning.
	result := &ScanResult{
		Agents: []AgentResult{
			{Name: AgentCline, Tokens: 14_080, Sources: []string{".clinerules/"}},
		},
	}
	entries := auditBudget(result, "/mock/home", "/mock/proj")
	var found bool
	for _, e := range entries {
		if e.Name == "config-budget-warning" && e.Status == AuditWarn {
			found = true
			data := e.Data.(map[string]interface{})
			if data["agent"] != AgentCline {
				t.Errorf("data.agent = %v, want %q", data["agent"], AgentCline)
			}
		}
	}
	if !found {
		t.Error("expected config-budget-warning entry")
	}
}

func TestAuditBudget_Critical(t *testing.T) {
	// 21% of 128K = 26,880 tokens → should trigger config-budget-critical.
	result := &ScanResult{
		Agents: []AgentResult{
			{Name: AgentCline, Tokens: 26_880, Sources: []string{".clinerules/"}},
		},
	}
	entries := auditBudget(result, "/mock/home", "/mock/proj")
	var found bool
	for _, e := range entries {
		if e.Name == "config-budget-critical" && e.Status == AuditWarn {
			found = true
		}
	}
	if !found {
		t.Error("expected config-budget-critical entry")
	}
}

func TestAuditBudget_Under10Percent(t *testing.T) {
	// 5% of 128K = 6,400 tokens → no budget entries.
	result := &ScanResult{
		Agents: []AgentResult{
			{Name: AgentCline, Tokens: 6_400, Sources: []string{".clinerules/"}},
		},
	}
	entries := auditBudget(result, "/mock/home", "/mock/proj")
	for _, e := range entries {
		if e.Name == "config-budget-warning" || e.Name == "config-budget-critical" {
			t.Errorf("unexpected budget entry: %s", e.Name)
		}
	}
}

func TestAuditBudget_LargeSkill(t *testing.T) {
	dir := t.TempDir()
	// Create a large file at .clinerules/big.md.
	rulesDir := filepath.Join(dir, ".clinerules")
	mkdirAll(t, rulesDir)
	content := make([]byte, 12000) // 12000 bytes = 3000 tokens > 2000 threshold
	writeTestFile(t, filepath.Join(rulesDir, "big.md"), string(content))

	result := &ScanResult{
		Agents: []AgentResult{
			{Name: AgentCline, Tokens: 3000, Sources: []string{".clinerules/"}},
		},
	}
	entries := auditBudget(result, "/mock/home", dir)
	var found bool
	for _, e := range entries {
		if e.Name == "large-skill-warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected large-skill-warning entry")
	}
}

func TestAuditBudget_EmptySkill(t *testing.T) {
	dir := t.TempDir()
	// Create a tiny file at .clinerules/stub.md.
	rulesDir := filepath.Join(dir, ".clinerules")
	mkdirAll(t, rulesDir)
	writeTestFile(t, filepath.Join(rulesDir, "stub.md"), "hi") // 2 bytes = 0 tokens

	result := &ScanResult{
		Agents: []AgentResult{
			{Name: AgentCline, Tokens: 1, Sources: []string{".clinerules/"}},
		},
	}
	entries := auditBudget(result, "/mock/home", dir)
	// 2 bytes / 4 = 0 tokens — sourceTokens returns 0, which is not > 0 so empty check skips.
	// Need at least 1 token (4 bytes) but under 50.
	for _, e := range entries {
		if e.Name == "empty-token-skills" {
			t.Error("should not flag 0-token source as empty (0 is not > 0)")
		}
	}
}

func TestAuditBudget_EmptySkill_SmallFile(t *testing.T) {
	dir := t.TempDir()
	// Create a small file: 40 bytes = 10 tokens (>0, <50).
	rulesDir := filepath.Join(dir, ".clinerules")
	mkdirAll(t, rulesDir)
	content := make([]byte, 40)
	writeTestFile(t, filepath.Join(rulesDir, "stub.md"), string(content))

	result := &ScanResult{
		Agents: []AgentResult{
			{Name: AgentCline, Tokens: 10, Sources: []string{".clinerules/"}},
		},
	}
	entries := auditBudget(result, "/mock/home", dir)
	var found bool
	for _, e := range entries {
		if e.Name == "empty-token-skills" {
			found = true
		}
	}
	if !found {
		t.Error("expected empty-token-skills entry for 10-token source")
	}
}

func TestExpandSourcePath(t *testing.T) {
	got := expandSourcePath("~/.claude/skills/", "/home/user", "/proj")
	want := filepath.Join("/home/user", ".claude", "skills")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	got = expandSourcePath(".clinerules/", "/home/user", "/proj")
	want = filepath.Join("/proj", ".clinerules/")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatBudgetTokens(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{100, "100"},
		{1000, "1,000"},
		{18230, "18,230"},
		{165000, "165,000"},
	}
	for _, tt := range tests {
		if got := formatBudgetTokens(tt.input); got != tt.want {
			t.Errorf("formatBudgetTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatK(t *testing.T) {
	if got := formatK(165000); got != "165" {
		t.Errorf("formatK(165000) = %q, want %q", got, "165")
	}
	if got := formatK(128000); got != "128" {
		t.Errorf("formatK(128000) = %q, want %q", got, "128")
	}
}

// --- auditWindsurf ---

func TestAuditWindsurf_WithRules(t *testing.T) {
	proj := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, filepath.Join(proj, ".windsurf", "rules", "rule.md"), "project rule")
	writeTestFile(t, filepath.Join(home, ".windsurf", "rules", "global.md"), "global rule")

	env := testAuditEnv()
	a := auditWindsurf(proj, home, env)
	if len(a.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(a.Entries))
	}
	for _, e := range a.Entries {
		if e.Status != AuditOK {
			t.Errorf("entry %s: status = %s, want ok", e.Name, e.Status)
		}
	}
}

func TestAuditWindsurf_WithMCP(t *testing.T) {
	proj := t.TempDir()
	home := t.TempDir()
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"srv": map[string]interface{}{"command": "my-mcp-srv"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"), string(data))

	env := testAuditEnv("my-mcp-srv")
	a := auditWindsurf(proj, home, env)

	var found bool
	for _, e := range a.Entries {
		if e.Category == "mcp" && e.Name == "srv" && e.Status == AuditOK {
			found = true
		}
	}
	if !found {
		t.Error("expected MCP audit entry for srv")
	}
}

func TestAuditWindsurf_NoConfig(t *testing.T) {
	env := testAuditEnv()
	a := auditWindsurf(t.TempDir(), t.TempDir(), env)
	if len(a.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(a.Entries))
	}
}

// --- auditAider ---

func TestAuditAider_WithConfig(t *testing.T) {
	proj := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, filepath.Join(proj, ".aider.conf.yml"), "model: gpt-4")
	writeTestFile(t, filepath.Join(home, ".aider.conf.yml"), "model: claude")

	env := testAuditEnv()
	a := auditAider(proj, home, env)
	if len(a.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(a.Entries))
	}
	for _, e := range a.Entries {
		if e.Status != AuditOK {
			t.Errorf("entry %s: status = %s, want ok", e.Name, e.Status)
		}
	}
}

func TestAuditAider_NoConfig(t *testing.T) {
	env := testAuditEnv()
	a := auditAider(t.TempDir(), t.TempDir(), env)
	if len(a.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(a.Entries))
	}
}

// --- auditContinue ---

func TestAuditContinue_WithConfig(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".continue", "config.yaml"), "models: []")
	writeTestFile(t, filepath.Join(proj, ".continuerc.json"), "{}")

	env := testAuditEnv()
	a := auditContinue(proj, home, env)
	if len(a.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(a.Entries))
	}
	for _, e := range a.Entries {
		if e.Status != AuditOK {
			t.Errorf("entry %s: status = %s, want ok", e.Name, e.Status)
		}
	}
}

func TestAuditContinue_DeprecatedJSON(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".continue", "config.json"), "{}")

	env := testAuditEnv()
	a := auditContinue(t.TempDir(), home, env)
	if len(a.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(a.Entries))
	}
	if a.Entries[0].Status != AuditWarn {
		t.Errorf("status = %s, want warn (deprecated)", a.Entries[0].Status)
	}
}

func TestAuditContinue_NoConfig(t *testing.T) {
	env := testAuditEnv()
	a := auditContinue(t.TempDir(), t.TempDir(), env)
	if len(a.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(a.Entries))
	}
}

// --- auditCopilot ---

func TestAuditCopilot_WithInstructions(t *testing.T) {
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, ".github", "copilot-instructions.md"), "# Instructions\nUse Go.")

	env := testAuditEnv()
	a := auditCopilot(proj, "", env)
	if len(a.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(a.Entries))
	}
	if a.Entries[0].Status != AuditOK {
		t.Errorf("status = %s, want ok", a.Entries[0].Status)
	}
	if a.Entries[0].Name != "copilot-instructions.md" {
		t.Errorf("name = %q, want %q", a.Entries[0].Name, "copilot-instructions.md")
	}
}

func TestAuditCopilot_NoConfig(t *testing.T) {
	env := testAuditEnv()
	a := auditCopilot(t.TempDir(), "", env)
	if len(a.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(a.Entries))
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

	// 1 hook ok + 25 environment ok = 26 ok.
	if result.Summary.OK != 26 {
		t.Errorf("ok = %d, want 26", result.Summary.OK)
	}
	if result.Summary.Errors != 1 {
		t.Errorf("errors = %d, want 1", result.Summary.Errors)
	}
}
