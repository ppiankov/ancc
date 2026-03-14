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
		// Create parent directory first
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create parent dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create mock file %s: %v", path, err)
		}
	}
	return tempDir
}

func TestScanAgentPaths(t *testing.T) {
	mfs := &mockFileSystem{
		files: map[string]string{
			"home/.claude/settings.json":  `{"hooks": {"onSubmit": [{"hooks": [{}]}]}, "mcpServers": {"server1": {}}}`, // 1 hook, 1 mcp
			"home/.claude/CLAUDE.md":      "claude home",
			"home/.cline/skills/skill1/a": "a",
			"home/.cline/skills/skill2/b": "b",
			"project/.clinerules/rule1":   "rule1 content",
			"project/CLAUDE.md":           "claude project",
			"project/.claude/skills/ps/c": "c",
			"project/AGENTS.md":           "agents",
			"home/.codex/config.toml":     `[mcp_servers.server1]`,
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
				Name:      AgentClaudeCode,
				ConfigDir: ".claude",
				Home: []pathSpec{
					{
						Path: ".claude/settings.json", SourcePrefix: "~/",
						Parse: func(path string, r *AgentResult) (bool, int64) {
							return parseClaudeSettings(path, r)
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
				Name:      AgentClaudeCode,
				ConfigDir: "~/.claude",
				Skills:    4, // 1 home skill dir + 1 project skill dir + 1 home CLAUDE.md + 1 project CLAUDE.md
				Hooks:     1,
				MCP:       1,
				Sources:   []string{"./.claude/skills/", "./CLAUDE.md", "~/.claude/CLAUDE.md", "~/.claude/settings.json", "~/.claude/skills/"},
				Tokens:    bytesToTokens(73 + 11 + 1 + 14),
			},
		},
		{
			name: "Cline",
			spec: agentPathSpec{
				Name:      AgentCline,
				ConfigDir: ".cline",
				Home: []pathSpec{
					{Path: ".cline/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true},
				},
				Project: []pathSpec{
					{Path: ".clinerules", SourcePrefix: "./", Type: pathTypeDirFiles},
				},
			},
			expected: AgentResult{
				Name:      AgentCline,
				ConfigDir: "~/.cline",
				Skills:    3, // 2 home skill dirs, 1 project rule file
				Sources:   []string{"./.clinerules/", "~/.cline/skills/"},
				Tokens:    bytesToTokens(2 + 13),
			},
		},
		{
			name: "Codex MCP",
			spec: agentPathSpec{
				Name:      AgentCodex,
				ConfigDir: ".codex",
				Home: []pathSpec{
					{
						Path: ".codex/config.toml", SourcePrefix: "~/",
						Parse: func(path string, r *AgentResult) (bool, int64) {
							return parseCodexTOMLMCP(path, r)
						},
					},
				},
			},
			expected: AgentResult{
				Name:      AgentCodex,
				ConfigDir: "~/.codex",
				MCP:       1,
				Sources:   []string{"~/.codex/config.toml"},
				Tokens:    bytesToTokens(21),
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
			if result.ConfigDir != tc.expected.ConfigDir {
				t.Errorf("Expected config_dir %s, got %s", tc.expected.ConfigDir, result.ConfigDir)
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

func TestAllAgents(t *testing.T) {
	mfs := &mockFileSystem{
		files: map[string]string{
			"home/.claude/settings.json":  `{"hooks": {"onSubmit": [{"hooks": [{}]}]}, "mcpServers": {"server1": {}}}`,
			"home/.claude/CLAUDE.md":      "claude home",
			"home/.cline/skills/skill1/a": "a",
			"home/.cline/skills/skill2/b": "b",
			"project/.clinerules/rule1":   "rule1 content",
			"project/CLAUDE.md":           "claude project",
			"project/.claude/skills/ps/c": "c",
			"project/AGENTS.md":           "agents",
			"home/.codex/config.toml":     `[mcp_servers.server1]`,
			// cursor
			"home/.cursor/mcp.json":          `{"mcpServers": {"cursor1": {}}}`,
			"project/.cursor/rules/rule.mdc": "cursor rule",
			// opencode
			"home/.config/opencode/opencode.json":   `{"mcp": {"opencode1": {}}, "instructions": [{}]}`,
			"home/.config/opencode/commands/cmd.md": "cmd",
			"project/opencode.json":                 `{"mcp": {"opencode1": {}}, "instructions": [{}]}`,
			// qwen
			"home/.qwen/settings.json": `{"mcpServers": {"qwen1": {}}}`,
			// openclaw
			"home/.openclaw/openclaw.json":        `{"mcpServers": {"openclaw1": {}}}`,
			"home/.openclaw/config/mcporter.json": `{"mcpServers": {"mcporter1": {}}}`,
			// windsurf
			"home/.codeium/windsurf/mcp_config.json": `{"mcpServers": {"windsurf1": {}}}`,
			"home/.windsurf/rules/rule.txt":          "rule",
			"project/.windsurfrules":                 "windsurf rules",
			"project/.windsurf/rules/rule2.txt":      "rule2",
			// aider
			"home/.aider.conf.yml":    "aider config",
			"project/.aider.conf.yml": "aider config",
			// continue
			"home/.continue/config.yaml": "continue yaml",
			"home/.continue/config.json": `{"continue": true}`,
			"project/.continuerc.json":   `{"continue": true}`,
			// copilot
			"project/.github/copilot-instructions.md": "copilot instructions",
			// kilocode
			"home/.config/kilo/opencode.json":  `{"mcp": {"kilocode1": {}}, "instructions": [{}]}`,
			"project/.kilocode/rules/rule.txt": "rule",
		},
		dirs: []string{
			"home/.claude/skills/homeskill",
			"home/.cline/skills/skill1",
			"home/.cline/skills/skill2",
			"project/.claude/skills/ps",
			"project/.clinerules",
			// cursor
			"project/.cursor/rules",
			// opencode
			"home/.config/opencode/commands",
			"home/.config/opencode/skills/skill1",
			// qwen
			"home/.qwen/skills/qskill",
			// openclaw
			"home/.openclaw/skills/oskill",
			// windsurf
			"home/.windsurf/rules",
			"project/.windsurf/rules",
			// kilocode
			"home/.kilocode/skills/kskill",
			"project/.kilocode/rules",
			"project/.kilocode/skills/pskills",
		},
	}
	tempRoot := mfs.setup(t)
	homeDir := filepath.Join(tempRoot, "home")
	projectDir := filepath.Join(tempRoot, "project")

	tests := []struct {
		name           string
		fn             func(string, string) AgentResult
		expectedSkills int
		expectedMCP    int
		expectedHooks  int
	}{
		{"ClaudeCode", scanClaudeCode, 4, 1, 1},
		{"Cline", scanCline, 3, 0, 0},
		{"Cursor", scanCursor, 1, 1, 0},
		{"OpenCode", scanOpenCode, 4, 2, 0},
		{"Codex", scanCodex, 1, 1, 0},
		{"Qwen", scanQwen, 1, 1, 0},
		{"OpenClaw", scanOpenClaw, 1, 2, 0},
		{"Windsurf", scanWindsurf, 3, 1, 0},
		{"Aider", scanAider, 2, 0, 0},
		{"Continue", scanContinue, 3, 0, 0},
		{"Copilot", scanCopilot, 1, 0, 0},
		{"Kilocode", scanKilocode, 4, 1, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.fn(projectDir, homeDir)
			if result.Skills != tc.expectedSkills {
				t.Errorf("Expected %d skills, got %d", tc.expectedSkills, result.Skills)
			}
			if result.MCP != tc.expectedMCP {
				t.Errorf("Expected %d MCP, got %d", tc.expectedMCP, result.MCP)
			}
			if result.Hooks != tc.expectedHooks {
				t.Errorf("Expected %d hooks, got %d", tc.expectedHooks, result.Hooks)
			}
		})
	}
}
