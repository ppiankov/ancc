package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- Helper utilities ---

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// --- countSkillDirs ---

func TestCountSkillDirs_Empty(t *testing.T) {
	if got := countSkillDirs(t.TempDir()); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestCountSkillDirs_WithSubdirs(t *testing.T) {
	dir := t.TempDir()
	mkdirAll(t, filepath.Join(dir, "skill1"))
	mkdirAll(t, filepath.Join(dir, "skill2"))
	writeTestFile(t, filepath.Join(dir, "readme.txt"), "not a dir")

	if got := countSkillDirs(dir); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestCountSkillDirs_NonExistent(t *testing.T) {
	if got := countSkillDirs("/nonexistent/path"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

// --- countFiles ---

func TestCountFiles_Empty(t *testing.T) {
	if got := countFiles(t.TempDir()); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestCountFiles_WithFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.md"), "a")
	writeTestFile(t, filepath.Join(dir, "b.md"), "b")
	mkdirAll(t, filepath.Join(dir, "subdir"))

	if got := countFiles(dir); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

// --- fileBytes ---

func TestFileBytes_Existing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	writeTestFile(t, path, "hello world") // 11 bytes

	if got := fileBytes(path); got != 11 {
		t.Errorf("got %d, want 11", got)
	}
}

func TestFileBytes_NonExistent(t *testing.T) {
	if got := fileBytes("/nonexistent/file"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

// --- dirBytes ---

func TestDirBytes_WithFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.txt"), "aaaa")   // 4 bytes
	writeTestFile(t, filepath.Join(dir, "b.txt"), "bbbbbb") // 6 bytes
	mkdirAll(t, filepath.Join(dir, "subdir"))               // skipped

	if got := dirBytes(dir); got != 10 {
		t.Errorf("got %d, want 10", got)
	}
}

func TestDirBytes_NonExistent(t *testing.T) {
	if got := dirBytes("/nonexistent"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

// --- dirBytesRecursive ---

func TestDirBytesRecursive_Nested(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.txt"), "aaaa")          // 4
	writeTestFile(t, filepath.Join(dir, "sub", "b.txt"), "bbbbbb") // 6

	if got := dirBytesRecursive(dir); got != 10 {
		t.Errorf("got %d, want 10", got)
	}
}

func TestDirBytesRecursive_NonExistent(t *testing.T) {
	if got := dirBytesRecursive("/nonexistent"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

// --- bytesToTokens ---

func TestBytesToTokens(t *testing.T) {
	tests := []struct{ bytes, want int64 }{
		{0, 0},
		{3, 0},
		{4, 1},
		{100, 25},
		{13600, 3400},
	}
	for _, tt := range tests {
		if got := bytesToTokens(tt.bytes); got != tt.want {
			t.Errorf("bytesToTokens(%d) = %d, want %d", tt.bytes, got, tt.want)
		}
	}
}

// --- parseClaudeSettings ---

func TestParseClaudeSettings_WithHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "echo hi"},
						map[string]interface{}{"type": "command", "command": "echo bye"},
					},
				},
			},
			"PostToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Write",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "lint"},
					},
				},
			},
		},
		"mcpServers": map[string]interface{}{
			"server1": map[string]interface{}{"command": "server1"},
			"server2": map[string]interface{}{"command": "server2"},
		},
	}
	data, _ := json.Marshal(settings)
	writeTestFile(t, path, string(data))

	hooks, mcp, found := parseClaudeSettings(path)
	if !found {
		t.Fatal("expected found=true")
	}
	if hooks != 3 {
		t.Errorf("hooks = %d, want 3", hooks)
	}
	if mcp != 2 {
		t.Errorf("mcp = %d, want 2", mcp)
	}
}

func TestParseClaudeSettings_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeTestFile(t, path, "not json")

	_, _, found := parseClaudeSettings(path)
	if !found {
		t.Error("expected found=true for existing but malformed file")
	}
}

func TestParseClaudeSettings_Missing(t *testing.T) {
	_, _, found := parseClaudeSettings("/nonexistent/settings.json")
	if found {
		t.Error("expected found=false for missing file")
	}
}

// --- parseMCPServers ---

func TestParseMCPServers_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"a": map[string]interface{}{"command": "a"},
			"b": map[string]interface{}{"command": "b"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, path, string(data))

	if got := parseMCPServers(path); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestParseMCPServers_Missing(t *testing.T) {
	if got := parseMCPServers("/nonexistent"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

// --- scanClaudeCode ---

func TestScanClaudeCode_NoConfig(t *testing.T) {
	r := scanClaudeCode(t.TempDir(), t.TempDir())
	if r.Skills != 0 || r.Hooks != 0 || r.MCP != 0 {
		t.Errorf("expected all zeros, got skills=%d hooks=%d mcp=%d", r.Skills, r.Hooks, r.MCP)
	}
}

func TestScanClaudeCode_GlobalAndProject(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	// Global skills.
	mkdirAll(t, filepath.Join(home, ".claude", "skills", "skill1"))
	mkdirAll(t, filepath.Join(home, ".claude", "skills", "skill2"))

	// Global settings with hooks.
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks":   []interface{}{map[string]interface{}{"type": "command"}},
				},
			},
		},
		"mcpServers": map[string]interface{}{
			"srv": map[string]interface{}{"command": "srv"},
		},
	}
	data, _ := json.Marshal(settings)
	writeTestFile(t, filepath.Join(home, ".claude", "settings.json"), string(data))

	// Project skills.
	mkdirAll(t, filepath.Join(proj, ".claude", "skills", "proj-skill"))

	// Project settings.
	writeTestFile(t, filepath.Join(proj, ".claude", "settings.local.json"), "{}")

	r := scanClaudeCode(proj, home)
	if r.Skills != 3 {
		t.Errorf("skills = %d, want 3 (2 global + 1 project)", r.Skills)
	}
	if r.Hooks != 1 {
		t.Errorf("hooks = %d, want 1", r.Hooks)
	}
	if r.MCP != 1 {
		t.Errorf("mcp = %d, want 1", r.MCP)
	}
	if len(r.Sources) != 4 {
		t.Errorf("sources = %d, want 4", len(r.Sources))
	}
}

// --- scanCline ---

func TestScanCline_NoConfig(t *testing.T) {
	r := scanCline(t.TempDir(), "")
	if r.Skills != 0 {
		t.Errorf("skills = %d, want 0", r.Skills)
	}
}

func TestScanCline_WithRules(t *testing.T) {
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, ".clinerules", "rule1.md"), "rule1")
	writeTestFile(t, filepath.Join(proj, ".clinerules", "rule2.md"), "rule2")

	r := scanCline(proj, "")
	if r.Skills != 2 {
		t.Errorf("skills = %d, want 2", r.Skills)
	}
}

// --- scanCursor ---

func TestScanCursor_NoConfig(t *testing.T) {
	r := scanCursor(t.TempDir(), t.TempDir())
	if r.Skills != 0 || r.MCP != 0 {
		t.Errorf("expected zeros, got skills=%d mcp=%d", r.Skills, r.MCP)
	}
}

func TestScanCursor_WithMDC(t *testing.T) {
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, ".cursor", "rules", "a.mdc"), "rule")
	writeTestFile(t, filepath.Join(proj, ".cursor", "rules", "b.mdc"), "rule")
	writeTestFile(t, filepath.Join(proj, ".cursor", "rules", "readme.txt"), "not a rule")

	r := scanCursor(proj, t.TempDir())
	if r.Skills != 2 {
		t.Errorf("skills = %d, want 2 (only .mdc files)", r.Skills)
	}
}

// --- scanOpenCode ---

func TestScanOpenCode_NoConfig(t *testing.T) {
	r := scanOpenCode(t.TempDir(), t.TempDir())
	if r.Skills != 0 || r.MCP != 0 {
		t.Errorf("expected zeros, got skills=%d mcp=%d", r.Skills, r.MCP)
	}
	if !r.Advisory {
		t.Error("expected advisory=true")
	}
}

func TestScanOpenCode_WithConfig(t *testing.T) {
	home := t.TempDir()
	cfg := map[string]interface{}{
		"instructions": []interface{}{"AGENTS.md", "README.md"},
		"mcp": map[string]interface{}{
			"server1": map[string]interface{}{"type": "local", "command": []string{"srv"}},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, filepath.Join(home, ".config", "opencode", "opencode.json"), string(data))

	r := scanOpenCode(t.TempDir(), home)
	if r.Skills != 2 {
		t.Errorf("skills = %d, want 2", r.Skills)
	}
	if r.MCP != 1 {
		t.Errorf("mcp = %d, want 1", r.MCP)
	}
}

// --- scanCodex ---

func TestScanCodex_NoConfig(t *testing.T) {
	r := scanCodex(t.TempDir(), "")
	if r.Skills != 0 {
		t.Errorf("skills = %d, want 0", r.Skills)
	}
	if !r.Advisory {
		t.Error("expected advisory=true")
	}
}

func TestScanCodex_WithAgentsMD(t *testing.T) {
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, "AGENTS.md"), "# Agent instructions")

	r := scanCodex(proj, "")
	if r.Skills != 1 {
		t.Errorf("skills = %d, want 1", r.Skills)
	}
}

func TestScanCodex_WithCodexDir(t *testing.T) {
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, "AGENTS.md"), "# Agent instructions")
	writeTestFile(t, filepath.Join(proj, ".codex", "config.json"), "{}")

	r := scanCodex(proj, "")
	if r.Skills != 2 {
		t.Errorf("skills = %d, want 2 (AGENTS.md + config.json)", r.Skills)
	}
}

// --- scanQwen ---

func TestScanQwen_NoConfig(t *testing.T) {
	r := scanQwen("", t.TempDir())
	if r.MCP != 0 {
		t.Errorf("mcp = %d, want 0", r.MCP)
	}
	if !r.Advisory {
		t.Error("expected advisory=true")
	}
}

func TestScanQwen_WithMCP(t *testing.T) {
	home := t.TempDir()
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"pastewatch": map[string]interface{}{"command": "pastewatch-cli"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, filepath.Join(home, ".qwen", "settings.json"), string(data))

	r := scanQwen("", home)
	if r.MCP != 1 {
		t.Errorf("mcp = %d, want 1", r.MCP)
	}
}

// --- ScanWithHome (integration) ---

func TestScanWithHome_EmptyDir(t *testing.T) {
	result, err := ScanWithHome(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Agents) != 0 {
		t.Errorf("agents = %d, want 0", len(result.Agents))
	}
	if result.Product != nil {
		t.Error("expected nil product")
	}
}

func TestScanWithHome_WithAgents(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	// Claude: 1 global skill.
	mkdirAll(t, filepath.Join(home, ".claude", "skills", "commit"))

	// Cline: 1 rule.
	writeTestFile(t, filepath.Join(proj, ".clinerules", "main.md"), "rules")

	result, err := ScanWithHome(proj, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Agents) != 2 {
		t.Errorf("agents = %d, want 2", len(result.Agents))
	}
}

func TestScanWithHome_WithProduct(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")

	// Use the valid-skill.md fixture as SKILL.md in docs/.
	proj := t.TempDir()
	fixtureData, err := os.ReadFile(filepath.Join(repoRoot, "testdata", "valid-skill.md"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(proj, "docs", "SKILL.md"), string(fixtureData))

	result, err := ScanWithHome(proj, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.Product == nil {
		t.Fatal("expected non-nil product")
	}
	if result.Product.Name == "" {
		t.Error("expected non-empty product name")
	}
}

// --- findANCCProduct ---

func TestFindANCCProduct_DocsLocation(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")

	proj := t.TempDir()
	fixtureData, err := os.ReadFile(filepath.Join(repoRoot, "testdata", "valid-skill.md"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(proj, "docs", "SKILL.md"), string(fixtureData))

	product := findANCCProduct(proj)
	if product == nil {
		t.Fatal("expected non-nil product")
	}
}

func TestFindANCCProduct_RootLocation(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")

	proj := t.TempDir()
	fixtureData, err := os.ReadFile(filepath.Join(repoRoot, "testdata", "valid-skill.md"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(proj, "SKILL.md"), string(fixtureData))

	product := findANCCProduct(proj)
	if product == nil {
		t.Fatal("expected non-nil product")
	}
}

func TestFindANCCProduct_NotFound(t *testing.T) {
	product := findANCCProduct(t.TempDir())
	if product != nil {
		t.Error("expected nil product")
	}
}

// --- Token counting in scanners ---

func TestScanClaudeCode_Tokens(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	// Global settings (small file).
	writeTestFile(t, filepath.Join(home, ".claude", "settings.json"), "{}")
	// Global skill with content.
	content := strings.Repeat("x", 400) // 400 bytes = 100 tokens
	writeTestFile(t, filepath.Join(home, ".claude", "skills", "s1", "prompt.md"), content)
	// Project CLAUDE.md.
	writeTestFile(t, filepath.Join(proj, "CLAUDE.md"), strings.Repeat("y", 200))

	r := scanClaudeCode(proj, home)
	// 2 (settings.json) + 400 (skill file) + 200 (CLAUDE.md) = 602 bytes / 4 = 150
	if r.Tokens != 150 {
		t.Errorf("tokens = %d, want 150", r.Tokens)
	}
}

func TestScanCline_Tokens(t *testing.T) {
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, ".clinerules", "rule.md"), strings.Repeat("a", 80))

	r := scanCline(proj, "")
	if r.Tokens != 20 { // 80 / 4
		t.Errorf("tokens = %d, want 20", r.Tokens)
	}
}

func TestScanCursor_Tokens(t *testing.T) {
	proj := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, filepath.Join(proj, ".cursor", "rules", "a.mdc"), strings.Repeat("r", 40))
	writeTestFile(t, filepath.Join(home, ".cursor", "mcp.json"), `{"mcpServers":{"s":{}}}`)

	r := scanCursor(proj, home)
	expected := bytesToTokens(40 + int64(len(`{"mcpServers":{"s":{}}}`)))
	if r.Tokens != expected {
		t.Errorf("tokens = %d, want %d", r.Tokens, expected)
	}
}

func TestScanCodex_Tokens(t *testing.T) {
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, "AGENTS.md"), strings.Repeat("a", 120))

	r := scanCodex(proj, "")
	if r.Tokens != 30 { // 120 / 4
		t.Errorf("tokens = %d, want 30", r.Tokens)
	}
}

func TestScanWithHome_AgentTokens(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	// Claude config with known size.
	writeTestFile(t, filepath.Join(proj, "CLAUDE.md"), strings.Repeat("x", 400))
	// Cline config.
	writeTestFile(t, filepath.Join(proj, ".clinerules", "rule.md"), strings.Repeat("y", 200))

	result, err := ScanWithHome(proj, home)
	if err != nil {
		t.Fatal(err)
	}

	// Verify individual agents have tokens.
	var claudeTokens, clineTokens int64
	for _, a := range result.Agents {
		switch a.Name {
		case AgentClaudeCode:
			claudeTokens = a.Tokens
		case AgentCline:
			clineTokens = a.Tokens
		}
	}
	if claudeTokens != 100 { // 400 / 4
		t.Errorf("claude tokens = %d, want 100", claudeTokens)
	}
	if clineTokens != 50 { // 200 / 4
		t.Errorf("cline tokens = %d, want 50", clineTokens)
	}
}

func TestScanWithHome_TokensOnlyAgent(t *testing.T) {
	// A project with only CLAUDE.md and no skills/hooks/MCP should still appear
	// because it has tokens > 0.
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, "CLAUDE.md"), strings.Repeat("x", 400))

	result, err := ScanWithHome(proj, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, a := range result.Agents {
		if a.Name == AgentClaudeCode {
			found = true
			if a.Tokens == 0 {
				t.Error("expected non-zero tokens for claude-code")
			}
		}
	}
	if !found {
		t.Error("expected claude-code agent in results (has tokens from CLAUDE.md)")
	}
}
