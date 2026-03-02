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

func TestCountSkillDirs_Symlinks(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir()
	mkdirAll(t, filepath.Join(target, "real-skill"))

	// Create symlink from dir/linked-skill -> target/real-skill
	if err := os.Symlink(filepath.Join(target, "real-skill"), filepath.Join(dir, "linked-skill")); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	// Also a real dir.
	mkdirAll(t, filepath.Join(dir, "real-dir"))

	if got := countSkillDirs(dir); got != 2 {
		t.Errorf("got %d, want 2 (1 real dir + 1 symlinked dir)", got)
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

func TestDirBytesRecursive_Symlinks(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir()
	writeTestFile(t, filepath.Join(target, "skill-dir", "prompt.md"), strings.Repeat("x", 100))

	// Symlink dir/linked -> target/skill-dir
	if err := os.Symlink(filepath.Join(target, "skill-dir"), filepath.Join(dir, "linked")); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	if got := dirBytesRecursive(dir); got != 100 {
		t.Errorf("got %d, want 100", got)
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

func TestScanCline_HomeSkills(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	mkdirAll(t, filepath.Join(home, ".cline", "skills", "commit"))
	mkdirAll(t, filepath.Join(home, ".cline", "skills", "review"))

	r := scanCline(proj, home)
	if r.Skills != 2 {
		t.Errorf("skills = %d, want 2", r.Skills)
	}
}

func TestScanCline_HomeSymlinkedSkills(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	target := t.TempDir()

	mkdirAll(t, filepath.Join(target, "commit"))
	if err := os.Symlink(filepath.Join(target, "commit"), filepath.Join(home, ".cline", "skills", "commit")); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	r := scanCline(proj, home)
	if r.Skills != 1 {
		t.Errorf("skills = %d, want 1 (symlinked)", r.Skills)
	}
}

func TestScanCline_HomeAndProject(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	mkdirAll(t, filepath.Join(home, ".cline", "skills", "commit"))
	writeTestFile(t, filepath.Join(proj, ".clinerules", "rule.md"), "rule")

	r := scanCline(proj, home)
	if r.Skills != 2 { // 1 home skill + 1 project rule
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

// --- parseCodexTOMLMCP ---

func TestParseCodexTOMLMCP_WithServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `model = "gpt-5.3-codex"

[mcp_servers.pastewatch]
command = "pastewatch-cli"
args = ["mcp"]
enabled = true

[mcp_servers.filesystem]
command = "fs-server"
enabled = true
`
	writeTestFile(t, path, content)

	if got := parseCodexTOMLMCP(path); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestParseCodexTOMLMCP_NoServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeTestFile(t, path, `model = "gpt-5.3-codex"`)

	if got := parseCodexTOMLMCP(path); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestParseCodexTOMLMCP_Missing(t *testing.T) {
	if got := parseCodexTOMLMCP("/nonexistent/config.toml"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestParseCodexTOMLMCP_IgnoresArrayTables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[mcp_servers.real]
command = "srv"

[[mcp_servers.fake]]
command = "not-a-table"
`
	writeTestFile(t, path, content)

	if got := parseCodexTOMLMCP(path); got != 1 {
		t.Errorf("got %d, want 1 (array table should be ignored)", got)
	}
}

// --- parseOpenCodeJSON ---

func TestParseOpenCodeJSON_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	cfg := map[string]interface{}{
		"instructions": []interface{}{"AGENTS.md", "README.md"},
		"mcp": map[string]interface{}{
			"server1": map[string]interface{}{"type": "local"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, path, string(data))

	instructions, mcp, b, found := parseOpenCodeJSON(path)
	if !found {
		t.Fatal("expected found=true")
	}
	if instructions != 2 {
		t.Errorf("instructions = %d, want 2", instructions)
	}
	if mcp != 1 {
		t.Errorf("mcp = %d, want 1", mcp)
	}
	if b == 0 {
		t.Error("expected non-zero bytes")
	}
}

func TestParseOpenCodeJSON_Missing(t *testing.T) {
	_, _, _, found := parseOpenCodeJSON("/nonexistent")
	if found {
		t.Error("expected found=false")
	}
}

func TestParseOpenCodeJSON_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	writeTestFile(t, path, "not json")

	_, _, b, found := parseOpenCodeJSON(path)
	if !found {
		t.Error("expected found=true for existing file")
	}
	if b == 0 {
		t.Error("expected non-zero bytes for existing file")
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

func TestScanOpenCode_WithHomeConfig(t *testing.T) {
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

func TestScanOpenCode_WithProjectConfig(t *testing.T) {
	proj := t.TempDir()
	cfg := map[string]interface{}{
		"instructions": []interface{}{"AGENTS.md"},
		"mcp": map[string]interface{}{
			"pastewatch": map[string]interface{}{"type": "local"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, filepath.Join(proj, "opencode.json"), string(data))

	r := scanOpenCode(proj, t.TempDir())
	if r.Skills != 1 {
		t.Errorf("skills = %d, want 1", r.Skills)
	}
	if r.MCP != 1 {
		t.Errorf("mcp = %d, want 1", r.MCP)
	}
}

func TestScanOpenCode_HomeAndProject(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	homeCfg := map[string]interface{}{
		"instructions": []interface{}{"AGENTS.md"},
		"mcp": map[string]interface{}{
			"pastewatch": map[string]interface{}{"type": "local"},
		},
	}
	data, _ := json.Marshal(homeCfg)
	writeTestFile(t, filepath.Join(home, ".config", "opencode", "opencode.json"), string(data))

	projCfg := map[string]interface{}{
		"instructions": []interface{}{"README.md"},
	}
	data, _ = json.Marshal(projCfg)
	writeTestFile(t, filepath.Join(proj, "opencode.json"), string(data))

	r := scanOpenCode(proj, home)
	if r.Skills != 2 { // 1 home + 1 project
		t.Errorf("skills = %d, want 2", r.Skills)
	}
	if r.MCP != 1 { // only from home
		t.Errorf("mcp = %d, want 1", r.MCP)
	}
	if len(r.Sources) != 2 {
		t.Errorf("sources = %d, want 2", len(r.Sources))
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

func TestScanCodex_HomeAgentsMD(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".codex", "AGENTS.md"), "# Global agent instructions")

	r := scanCodex(proj, home)
	if r.Skills != 1 {
		t.Errorf("skills = %d, want 1 (home AGENTS.md)", r.Skills)
	}
}

func TestScanCodex_HomeSkills(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	mkdirAll(t, filepath.Join(home, ".codex", "skills", "commit"))
	mkdirAll(t, filepath.Join(home, ".codex", "skills", "review"))
	mkdirAll(t, filepath.Join(home, ".codex", "skills", "ship"))

	r := scanCodex(proj, home)
	if r.Skills != 3 {
		t.Errorf("skills = %d, want 3 (home skill dirs)", r.Skills)
	}
}

func TestScanCodex_HomeMCP(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	content := `model = "gpt-5.3-codex"

[mcp_servers.pastewatch]
command = "pastewatch-cli"
enabled = true

[mcp_servers.filesystem]
command = "fs-server"
`
	writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), content)

	r := scanCodex(proj, home)
	if r.MCP != 2 {
		t.Errorf("mcp = %d, want 2", r.MCP)
	}
}

func TestScanCodex_HomeAndProject(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()

	// Home: AGENTS.md + 2 skills + 1 MCP
	writeTestFile(t, filepath.Join(home, ".codex", "AGENTS.md"), "# Global agents")
	mkdirAll(t, filepath.Join(home, ".codex", "skills", "commit"))
	mkdirAll(t, filepath.Join(home, ".codex", "skills", "review"))
	writeTestFile(t, filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.pw]\ncommand = \"pw\"\n")

	// Project: AGENTS.md + 1 .codex file
	writeTestFile(t, filepath.Join(proj, "AGENTS.md"), "# Project agents")
	writeTestFile(t, filepath.Join(proj, ".codex", "config.toml"), "doc = true")

	r := scanCodex(proj, home)
	// 1 home AGENTS.md + 2 home skills + 1 project AGENTS.md + 1 .codex file = 5
	if r.Skills != 5 {
		t.Errorf("skills = %d, want 5", r.Skills)
	}
	if r.MCP != 1 {
		t.Errorf("mcp = %d, want 1", r.MCP)
	}
}

func TestScanCodex_HomeSymlinkedSkills(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	target := t.TempDir()

	mkdirAll(t, filepath.Join(target, "commit"))
	skillsDir := filepath.Join(home, ".codex", "skills")
	mkdirAll(t, skillsDir)
	if err := os.Symlink(filepath.Join(target, "commit"), filepath.Join(skillsDir, "commit")); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	r := scanCodex(proj, home)
	if r.Skills != 1 {
		t.Errorf("skills = %d, want 1 (symlinked)", r.Skills)
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

func TestScanQwen_HomeSkills(t *testing.T) {
	home := t.TempDir()
	mkdirAll(t, filepath.Join(home, ".qwen", "skills", "commit"))
	mkdirAll(t, filepath.Join(home, ".qwen", "skills", "review"))

	r := scanQwen("", home)
	if r.Skills != 2 {
		t.Errorf("skills = %d, want 2", r.Skills)
	}
}

func TestScanQwen_HomeSkillsAndMCP(t *testing.T) {
	home := t.TempDir()
	mkdirAll(t, filepath.Join(home, ".qwen", "skills", "commit"))
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"srv": map[string]interface{}{"command": "srv"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, filepath.Join(home, ".qwen", "settings.json"), string(data))

	r := scanQwen("", home)
	if r.Skills != 1 {
		t.Errorf("skills = %d, want 1", r.Skills)
	}
	if r.MCP != 1 {
		t.Errorf("mcp = %d, want 1", r.MCP)
	}
}

// --- scanOpenClaw ---

func TestScanOpenClaw_NoConfig(t *testing.T) {
	r := scanOpenClaw("", t.TempDir())
	if r.Skills != 0 || r.MCP != 0 {
		t.Errorf("expected zeros, got skills=%d mcp=%d", r.Skills, r.MCP)
	}
	if !r.Advisory {
		t.Error("expected advisory=true")
	}
}

func TestScanOpenClaw_EmptyHome(t *testing.T) {
	r := scanOpenClaw("", "")
	if r.Skills != 0 || r.MCP != 0 {
		t.Errorf("expected zeros, got skills=%d mcp=%d", r.Skills, r.MCP)
	}
}

func TestScanOpenClaw_WithSkills(t *testing.T) {
	home := t.TempDir()
	mkdirAll(t, filepath.Join(home, ".openclaw", "skills", "chainwatch"))
	mkdirAll(t, filepath.Join(home, ".openclaw", "skills", "safety"))

	r := scanOpenClaw("", home)
	if r.Skills != 2 {
		t.Errorf("skills = %d, want 2", r.Skills)
	}
}

func TestScanOpenClaw_WithMCP(t *testing.T) {
	home := t.TempDir()
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"chainwatch": map[string]interface{}{"command": "chainwatch-mcp"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, filepath.Join(home, ".openclaw", "openclaw.json"), string(data))

	r := scanOpenClaw("", home)
	if r.MCP != 1 {
		t.Errorf("mcp = %d, want 1", r.MCP)
	}
}

func TestScanOpenClaw_WithMcporter(t *testing.T) {
	home := t.TempDir()
	cfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"chainwatch": map[string]interface{}{"command": "chainwatch-mcp"},
			"filesystem": map[string]interface{}{"command": "fs-server"},
		},
	}
	data, _ := json.Marshal(cfg)
	writeTestFile(t, filepath.Join(home, ".openclaw", "config", "mcporter.json"), string(data))

	r := scanOpenClaw("", home)
	if r.MCP != 2 {
		t.Errorf("mcp = %d, want 2", r.MCP)
	}
}

func TestScanOpenClaw_SkillsAndMCP(t *testing.T) {
	home := t.TempDir()
	mkdirAll(t, filepath.Join(home, ".openclaw", "skills", "safety"))

	mainCfg := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"chainwatch": map[string]interface{}{"command": "chainwatch-mcp"},
		},
	}
	data, _ := json.Marshal(mainCfg)
	writeTestFile(t, filepath.Join(home, ".openclaw", "openclaw.json"), string(data))

	mcporter := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"extra": map[string]interface{}{"command": "extra-mcp"},
		},
	}
	data, _ = json.Marshal(mcporter)
	writeTestFile(t, filepath.Join(home, ".openclaw", "config", "mcporter.json"), string(data))

	r := scanOpenClaw("", home)
	if r.Skills != 1 {
		t.Errorf("skills = %d, want 1", r.Skills)
	}
	if r.MCP != 2 { // 1 from openclaw.json + 1 from mcporter.json
		t.Errorf("mcp = %d, want 2", r.MCP)
	}
	if len(r.Sources) != 3 {
		t.Errorf("sources = %d, want 3", len(r.Sources))
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
	home := t.TempDir()
	proj := t.TempDir()
	writeTestFile(t, filepath.Join(proj, "AGENTS.md"), strings.Repeat("a", 120))
	writeTestFile(t, filepath.Join(home, ".codex", "AGENTS.md"), strings.Repeat("b", 80))

	r := scanCodex(proj, home)
	// 120 (project AGENTS.md) + 80 (home AGENTS.md) = 200 bytes / 4 = 50
	if r.Tokens != 50 {
		t.Errorf("tokens = %d, want 50", r.Tokens)
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
