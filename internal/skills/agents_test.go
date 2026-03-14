package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// mockFileSystem sets up a temporary directory with a given structure for testing.
type mockFileSystem struct {
	files map[string]string // path -> content
	dirs  []string          // list of directories to create
}

func (mfs *mockFileSystem) setup(t *testing.T) string {
	tempDir := t.TempDir()
	for _, dir := range mfs.dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create mock directory %s: %v", dir, err)
		}
	}
	for path, content := range mfs.files {
		fullPath := filepath.Join(tempDir, path)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create mock file %s: %v", path, err)
		}
	}
	return tempDir
}

func TestScanAgentPaths(t *testing.T) {
	mfs := &mockFileSystem{
		files: map[string]string{
			"home/.claude/settings.json":    `{"hooks": {"onSubmit": [{"hooks": [{}]}]}, "mcpServers": {"server1": {}}}`, // 1 hook, 1 mcp
			"home/.claude/CLAUDE.md":        "claude home",
			"home/.cline/skills/skill1/a":   "a",
			"home/.cline/skills/skill2/b":   "b",
			"project/.clinerules/rule1":     "rule1 content",
			"project/CLAUDE.md":             "claude project",
			"project/.claude/skills/ps/c":   "c",
			"project/AGENTS.md":             "agents",
			"home/.codex/config.toml":       `[mcp_servers.server1]`,
		},
		dirs: []string{
			"home/.claude/skills/homeskill",
			"home/.cline/skills/skill1",
			"home/.cline/skills/skill2",
			"project/.claude/skills/ps",
			"project/.clinerules",
		},
	}

	tempRoot := mfs.setup(t)
	homeDir := filepath.Join(tempRoot, "home")
	projectDir := filepath.Join(tempRoot, "project")

	testCases := []struct {
		name     string
		spec     agentPathSpec
		expected AgentResult
	}{
		{
			name: "ClaudeCode",
			spec: agentPathSpec{
				Name: AgentClaudeCode,
				Home: []pathSpec{
					{
						Path: ".claude/settings.json", SourcePrefix: "~/",
						Parse: func(path string, r *AgentResult) (bool, int64) {
							hooks, mcp, found := parseClaudeSettings(path)
							r.Hooks += hooks
							r.MCP += mcp
							return found, fileBytes(path)
						},
					},
					{Path: ".claude/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true},
					{Path: ".claude/CLAUDE.md", SourcePrefix: "~/", Type: pathTypeFile},
				},
				Project: []pathSpec{
					{Path: ".claude/skills", SourcePrefix: "./", Type: pathTypeDirSkills, RecursiveSize: true},
					{Path: "CLAUDE.md", SourcePrefix: "./", Type: pathTypeFile},
					{Path: ".claude/settings.local.json", SourcePrefix: "./", Type: pathTypeFile},
					{Path: "CLAUDE.local.md", SourcePrefix: "./", Type: pathTypeFile},
				},
			},
			expected: AgentResult{
				Name:    AgentClaudeCode,
				Skills:  4, // 1 home skill dir + 1 project skill dir + 1 home CLAUDE.md + 1 project CLAUDE.md
				Hooks:   1,
				MCP:     1,
				Sources: []string{"./.claude/skills/", "./CLAUDE.md", "~/.claude/CLAUDE.md", "~/.claude/settings.json", "~/.claude/skills/"},
				Tokens:  bytesToTokens(83 + 11 + 1 + 14 + 1),
			},
		},
		{
			name: "Cline",
			spec: agentPathSpec{
				Name: AgentCline,
				Home: []pathSpec{
					{Path: ".cline/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true},
				},
				Project: []pathSpec{
					{Path: ".clinerules", SourcePrefix: "./", Type: pathTypeDirFiles},
				},
			},
			expected: AgentResult{
				Name:    AgentCline,
				Skills:  3, // 2 home skill dirs, 1 project rule file
				Sources: []string{"./.clinerules/", "~/.cline/skills/"},
				Tokens:  bytesToTokens(2 + 13),
			},
		},
		{
			name: "Codex MCP",
			spec: agentPathSpec{
				Name: AgentCodex,
				Home: []pathSpec{
					{
						Path: ".codex/config.toml", SourcePrefix: "~/",
						Parse: func(path string, r *AgentResult) (bool, int64) {
							mcp := parseCodexTOMLMCP(path)
							if mcp > 0 {
								r.MCP += mcp
								return true, fileBytes(path)
							}
							return false, 0
						},
					},
				},
			},
			expected: AgentResult{
				Name:    AgentCodex,
				MCP:     1,
				Sources: []string{"~/.codex/config.toml"},
				Tokens:  bytesToTokens(21),
			},
		},
		{
			name: "No config dir",
			spec: agentPathSpec{
				Name: "NoOpAgent",
				Home: []pathSpec{
					{Path: ".nonexistent/config.json", SourcePrefix: "~/", Type: pathTypeFile},
				},
			},
			expected: AgentResult{
				Name: "NoOpAgent",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := scanAgentPaths(projectDir, homeDir, tc.spec)

			// Sort sources for stable comparison
			sort.Strings(result.Sources)
			sort.Strings(tc.expected.Sources)

			if result.Name != tc.expected.Name {
				t.Errorf("Expected name %s, got %s", tc.expected.Name, result.Name)
			}
			if result.Skills != tc.expected.Skills {
				t.Errorf("Expected %d skills, got %d", tc.expected.Skills, result.Skills)
			}
			if result.Hooks != tc.expected.Hooks {
				t.Errorf("Expected %d hooks, got %d", tc.expected.Hooks, result.Hooks)
			}
			if result.MCP != tc.expected.MCP {
				t.Errorf("Expected %d MCP, got %d", tc.expected.MCP, result.MCP)
			}
			if !reflect.DeepEqual(result.Sources, tc.expected.Sources) {
				t.Errorf("Expected sources %v, got %v", tc.expected.Sources, result.Sources)
			}
			if result.Tokens != tc.expected.Tokens {
				t.Errorf("Expected %d tokens, got %d", tc.expected.Tokens, result.Tokens)
			}
		})
	}
}

func TestScanAgentFunctions(t *testing.T) {
	mfs := &mockFileSystem{
		files: map[string]string{
			// Claude
			"home/.claude/settings.json": `{"hooks": {"onSubmit": [{"hooks": [{}]}]}, "mcpServers": {"server1": {}}}`,
			"home/.claude/CLAUDE.md":     "claude home",
			"project/CLAUDE.md":          "claude project",
			"project/.claude/settings.local.json": "{}",
			"project/CLAUDE.local.md":    "local",
			// Cline
			"home/.cline/skills/skill1/a": "a",
			"project/.clinerules/rule1":   "rule1 content",
			// Cursor
			"project/.cursor/rules/rule.mdc": "cursor rule",
			"home/.cursor/mcp.json":          `{"mcpServers": {"s1":{}}}`,
			// OpenCode
			"home/.config/opencode/opencode.json": `{"instructions": [{}], "mcp": {"m1":{}}}`,
			"project/opencode.json":               `{"instructions": [{},{}], "mcp": {"m2":{}}}`,
			// Codex
			"home/.codex/AGENTS.md":      "agents",
			"home/.codex/config.toml":    "[mcp_servers.s1]",
			"project/AGENTS.md":          "p-agents",
			"project/.codex/skill.py":    "print(1)",
			// Qwen
			"home/.qwen/skills/qskill/a": "a",
			"home/.qwen/settings.json":   `{"mcpServers": {"s1":{}}}`,
			// OpenClaw
			"home/.openclaw/skills/oskill/a":   "a",
			"home/.openclaw/openclaw.json":     `{"mcpServers": {"s1":{}}}`,
			"home/.openclaw/config/mcporter.json": `{"mcpServers": {"s2":{}}}`,
			// Windsurf
			"project/.windsurfrules":          ".windsurfrules",
			"project/.windsurf/rules/ws_rule": "ws_rule",
			"home/.windsurf/rules/ws_home":    "ws_home",
			"home/.codeium/windsurf/mcp_config.json": `{"mcpServers": {"s1":{}}}`,
			// Aider
			"project/.aider.conf.yml": "aider",
			"home/.aider.conf.yml":    "h-aider",
			// Continue
			"home/.continue/config.yaml": "continue yaml",
			"home/.continue/config.json": "continue json",
			"project/.continuerc.json":   "continue rc",
			// Copilot
			"project/.github/copilot-instructions.md": "copilot",
		},
		dirs: []string{
			"home/.claude/skills/homeskill",
			"project/.claude/skills/ps",
			"home/.cline/skills/skill1",
			"project/.clinerules",
			"project/.cursor/rules",
			"home/.codex/skills/codexskill",
			"home/.qwen/skills/qskill",
			"home/.openclaw/skills/oskill",
			"project/.windsurf/rules",
			"home/.windsurf/rules",
		},
	}
	tempRoot := mfs.setup(t)
	homeDir := filepath.Join(tempRoot, "home")
	projectDir := filepath.Join(tempRoot, "project")

	scanners := []struct {
		name      string
		refactored func(string, string) AgentResult
		original  func(string, string) AgentResult
	}{
		{"scanClaudeCode", scanClaudeCode, scanClaudeCode_original},
		{"scanCline", scanCline, scanCline_original},
		{"scanCursor", scanCursor, scanCursor_original},
		{"scanOpenCode", scanOpenCode, scanOpenCode_original},
		{"scanCodex", scanCodex, scanCodex_original},
		{"scanQwen", scanQwen, scanQwen_original},
		{"scanOpenClaw", scanOpenClaw, scanOpenClaw_original},
		{"scanWindsurf", scanWindsurf, scanWindsurf_original},
		{"scanAider", scanAider, scanAider_original},
		{"scanContinue", scanContinue, scanContinue_original},
		{"scanCopilot", scanCopilot, scanCopilot_original},
	}

	for _, s := range scanners {
		t.Run("Compare_"+s.name, func(t *testing.T) {
			originalResult := s.original(projectDir, homeDir)
			refactoredResult := s.refactored(projectDir, homeDir)

			sort.Strings(originalResult.Sources)
			sort.Strings(refactoredResult.Sources)

			if !reflect.DeepEqual(originalResult, refactoredResult) {
				t.Errorf("Mismatch for %s:\nOriginal: %+v\nRefactored: %+v\n", s.name, originalResult, refactoredResult)
			}
		})
	}
}

// --- Original scan functions for comparison ---

func scanClaudeCode_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentClaudeCode}
	var bytes int64

	if homeDir != "" {
		globalSkillsDir := resolvePath(filepath.Join(homeDir, ".claude", "skills"))
		skillCount := countSkillDirs(globalSkillsDir)
		if skillCount > 0 {
			r.Skills += skillCount
			r.Sources = append(r.Sources, "~/.claude/skills/")
		}

		globalSettings := resolvePath(filepath.Join(homeDir, ".claude", "settings.json"))
		hooks, mcp, found := parseClaudeSettings(globalSettings)
		if found {
			r.Hooks += hooks
			r.MCP += mcp
			r.Sources = append(r.Sources, "~/.claude/settings.json")
		}

		bytes += fileBytes(globalSettings)
		bytes += dirBytesRecursive(globalSkillsDir)
		if fb := fileBytes(resolvePath(filepath.Join(homeDir, ".claude", "CLAUDE.md"))); fb > 0 {
			r.Skills++
			r.Sources = append(r.Sources, "~/.claude/CLAUDE.md")
			bytes += fb
		}
	}

	projectSkillsDir := resolvePath(filepath.Join(projectDir, ".claude", "skills"))
	projSkillCount := countSkillDirs(projectSkillsDir)
	if projSkillCount > 0 {
		r.Skills += projSkillCount
		r.Sources = append(r.Sources, "./.claude/skills/")
	}

	localSettings := resolvePath(filepath.Join(projectDir, ".claude", "settings.local.json"))
	if _, err := os.Stat(localSettings); err == nil {
		r.Skills++ // Count as a skill
		r.Sources = append(r.Sources, "./.claude/settings.local.json")
	}

	bytes += fileBytes(localSettings)
	bytes += dirBytesRecursive(projectSkillsDir)
	if fb := fileBytes(resolvePath(filepath.Join(projectDir, "CLAUDE.md"))); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, "./CLAUDE.md")
		bytes += fb
	}
	if fb := fileBytes(resolvePath(filepath.Join(projectDir, "CLAUDE.local.md"))); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, "./CLAUDE.local.md")
		bytes += fb
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanCline_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentCline}
	var bytes int64

	if homeDir != "" {
		homeSkillsDir := resolvePath(filepath.Join(homeDir, ".cline", "skills"))
		homeCount := countSkillDirs(homeSkillsDir)
		if homeCount > 0 {
			r.Skills += homeCount
			r.Sources = append(r.Sources, "~/.cline/skills/")
		}
		bytes += dirBytesRecursive(homeSkillsDir)
	}

	rulesDir := resolvePath(filepath.Join(projectDir, ".clinerules"))
	count := countFiles(rulesDir)
	if count > 0 {
		r.Skills += count
		r.Sources = append(r.Sources, "./.clinerules/")
	}
	bytes += dirBytes(rulesDir)

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanCursor_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentCursor}
	var bytes int64

	rulesDir := resolvePath(filepath.Join(projectDir, ".cursor", "rules"))
	entries, err := os.ReadDir(rulesDir)
	if err == nil {
		found := false
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".mdc" {
				r.Skills++
				found = true
			}
		}
		if found {
			r.Sources = append(r.Sources, "./.cursor/rules/")
		}
	}
	bytes += dirBytes(rulesDir)

	if homeDir != "" {
		mcpPath := resolvePath(filepath.Join(homeDir, ".cursor", "mcp.json"))
		mcpCount := parseMCPServers(mcpPath)
		if mcpCount > 0 {
			r.MCP += mcpCount
			r.Sources = append(r.Sources, "~/.cursor/mcp.json")
		}
		bytes += fileBytes(mcpPath)
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanOpenCode_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentOpenCode, Advisory: true}
	var totalBytes int64

	if homeDir != "" {
		instructions, mcp, b, found := parseOpenCodeJSON(resolvePath(filepath.Join(homeDir, ".config", "opencode", "opencode.json")))
		if found {
			r.Skills += instructions
			r.MCP += mcp
			totalBytes += b
			r.Sources = append(r.Sources, "~/.config/opencode/opencode.json (advisory)")
		}
	}

	instructions, mcp, b, found := parseOpenCodeJSON(resolvePath(filepath.Join(projectDir, "opencode.json")))
	if found {
		r.Skills += instructions
		r.MCP += mcp
		totalBytes += b
		r.Sources = append(r.Sources, "./opencode.json (advisory)")
	}

	r.Tokens = bytesToTokens(totalBytes)
	return r
}

func scanCodex_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentCodex, Advisory: true}
	var bytes int64

	if homeDir != "" {
		homeAgents := resolvePath(filepath.Join(homeDir, ".codex", "AGENTS.md"))
		if _, err := os.Stat(homeAgents); err == nil {
			r.Skills++
			r.Sources = append(r.Sources, "~/.codex/AGENTS.md (advisory)")
		}
		bytes += fileBytes(homeAgents)

		homeSkillsDir := resolvePath(filepath.Join(homeDir, ".codex", "skills"))
		homeSkillCount := countSkillDirs(homeSkillsDir)
		if homeSkillCount > 0 {
			r.Skills += homeSkillCount
			r.Sources = append(r.Sources, "~/.codex/skills/ (advisory)")
		}
		bytes += dirBytesRecursive(homeSkillsDir)

		configPath := resolvePath(filepath.Join(homeDir, ".codex", "config.toml"))
		mcpCount := parseCodexTOMLMCP(configPath)
		if mcpCount > 0 {
			r.MCP += mcpCount
			r.Sources = append(r.Sources, "~/.codex/config.toml (advisory)")
		}
		bytes += fileBytes(configPath)
	}

	agentsPath := resolvePath(filepath.Join(projectDir, "AGENTS.md"))
	if _, err := os.Stat(agentsPath); err == nil {
		r.Skills++
		r.Sources = append(r.Sources, "./AGENTS.md (advisory)")
	}
	bytes += fileBytes(agentsPath)

	codexDir := resolvePath(filepath.Join(projectDir, ".codex"))
	count := countFiles(codexDir)
	if count > 0 {
		r.Skills += count
		r.Sources = append(r.Sources, "./.codex/")
	}
	bytes += dirBytes(codexDir)

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanQwen_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentQwen, Advisory: true}
	var bytes int64

	if homeDir == "" {
		return r
	}

	homeSkillsDir := resolvePath(filepath.Join(homeDir, ".qwen", "skills"))
	homeCount := countSkillDirs(homeSkillsDir)
	if homeCount > 0 {
		r.Skills += homeCount
		r.Sources = append(r.Sources, "~/.qwen/skills/ (advisory)")
	}
	bytes += dirBytesRecursive(homeSkillsDir)

	cfgPath := resolvePath(filepath.Join(homeDir, ".qwen", "settings.json"))
	mcpCount := parseMCPServers(cfgPath)
	if mcpCount > 0 {
		r.MCP = mcpCount
		r.Sources = append(r.Sources, "~/.qwen/settings.json (advisory)")
	}
	bytes += fileBytes(cfgPath)

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanOpenClaw_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentOpenClaw, Advisory: true}
	var bytes int64

	if homeDir == "" {
		return r
	}

	homeSkillsDir := resolvePath(filepath.Join(homeDir, ".openclaw", "skills"))
	homeCount := countSkillDirs(homeSkillsDir)
	if homeCount > 0 {
		r.Skills += homeCount
		r.Sources = append(r.Sources, "~/.openclaw/skills/ (advisory)")
	}
	bytes += dirBytesRecursive(homeSkillsDir)

	cfgPath := resolvePath(filepath.Join(homeDir, ".openclaw", "openclaw.json"))
	mcpCount := parseMCPServers(cfgPath)
	if mcpCount > 0 {
		r.MCP += mcpCount
		r.Sources = append(r.Sources, "~/.openclaw/openclaw.json (advisory)")
	}
	bytes += fileBytes(cfgPath)

	mcporterPath := resolvePath(filepath.Join(homeDir, ".openclaw", "config", "mcporter.json"))
	mcporterCount := parseMCPServers(mcporterPath)
	if mcporterCount > 0 {
		r.MCP += mcporterCount
		r.Sources = append(r.Sources, "~/.openclaw/config/mcporter.json (advisory)")
	}
	bytes += fileBytes(mcporterPath)

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanWindsurf_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentWindsurf}
	var bytes int64

	rulesFile := resolvePath(filepath.Join(projectDir, ".windsurfrules"))
	if fb := fileBytes(rulesFile); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, "./.windsurfrules")
		bytes += fb
	}

	projRulesDir := resolvePath(filepath.Join(projectDir, ".windsurf", "rules"))
	projCount := countFiles(projRulesDir)
	if projCount > 0 {
		r.Skills += projCount
		r.Sources = append(r.Sources, "./.windsurf/rules/")
	}
	bytes += dirBytes(projRulesDir)

	if homeDir != "" {
		homeRulesDir := resolvePath(filepath.Join(homeDir, ".windsurf", "rules"))
		homeCount := countFiles(homeRulesDir)
		if homeCount > 0 {
			r.Skills += homeCount
			r.Sources = append(r.Sources, "~/.windsurf/rules/")
		}
		bytes += dirBytes(homeRulesDir)

		mcpPath := resolvePath(filepath.Join(homeDir, ".codeium", "windsurf", "mcp_config.json"))
		mcpCount := parseMCPServers(mcpPath)
		if mcpCount > 0 {
			r.MCP += mcpCount
			r.Sources = append(r.Sources, "~/.codeium/windsurf/mcp_config.json")
		}
		bytes += fileBytes(mcpPath)
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanAider_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentAider, Advisory: true}
	var bytes int64

	projConf := resolvePath(filepath.Join(projectDir, ".aider.conf.yml"))
	if fb := fileBytes(projConf); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, "./.aider.conf.yml (advisory)")
		bytes += fb
	}

	if homeDir != "" {
		homeConf := resolvePath(filepath.Join(homeDir, ".aider.conf.yml"))
		if fb := fileBytes(homeConf); fb > 0 {
			r.Skills++
			r.Sources = append(r.Sources, "~/.aider.conf.yml (advisory)")
			bytes += fb
		}
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanContinue_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentContinue, Advisory: true}
	var bytes int64

	if homeDir != "" {
		yamlConf := resolvePath(filepath.Join(homeDir, ".continue", "config.yaml"))
		if fb := fileBytes(yamlConf); fb > 0 {
			r.Skills++
			r.Sources = append(r.Sources, "~/.continue/config.yaml (advisory)")
			bytes += fb
		}

		jsonConf := resolvePath(filepath.Join(homeDir, ".continue", "config.json"))
		if fb := fileBytes(jsonConf); fb > 0 {
			r.Skills++
			r.Sources = append(r.Sources, "~/.continue/config.json (advisory)")
			bytes += fb
		}
	}

	projConf := resolvePath(filepath.Join(projectDir, ".continuerc.json"))
	if fb := fileBytes(projConf); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, "./.continuerc.json (advisory)")
		bytes += fb
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanCopilot_original(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentCopilot}
	var bytes int64

	instrFile := resolvePath(filepath.Join(projectDir, ".github", "copilot-instructions.md"))
	if fb := fileBytes(instrFile); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, "./.github/copilot-instructions.md")
		bytes += fb
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}
