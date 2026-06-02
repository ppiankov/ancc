package skills

import (
	"encoding/json"
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
			"home/.config/kilo/kilo.jsonc":     `{"instructions": [{}]}`, // WO-86: Kilo CLI config
			"project/.kilocode/rules/rule.txt": "rule",
			// aider expanded
			"project/CONVENTIONS.md": "conventions",
			// goose
			"home/.config/goose/config.yaml": "provider: anthropic",
			"project/.goosehints":            "goose hints",
			// WO-66: Antigravity scanner fixture.
			"home/.gemini/GEMINI.md":                                   "gemini rules",
			"home/.gemini/antigravity-cli/skills/review/SKILL.md":      "review skill",
			"home/.gemini/antigravity-cli/global_workflows/release.md": "release workflow",
			"project/.antigravitycli/skills/project/SKILL.md":          "project skill",
			"project/.antigravitycli/workflows/project-workflow.md":    "project workflow",
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
			// aider expanded
			"home/.aider/skills/review",
			// vibe
			"home/.vibe/skills/commit",
			// WO-66: Antigravity scanner fixture.
			"home/.gemini/antigravity-cli/skills/review",
			"home/.gemini/antigravity-cli/global_workflows",
			"project/.antigravitycli/skills/project",
			"project/.antigravitycli/workflows",
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
		{"Aider", scanAider, 4, 0, 0}, // home conf + home skill dir + project conf + CONVENTIONS.md
		{"Continue", scanContinue, 3, 0, 0},
		{"Copilot", scanCopilot, 2, 0, 0},   // instructions + AGENTS.md
		{"Kilocode", scanKilocode, 5, 1, 0}, // WO-86: +1 skill from kilo.jsonc instructions
		{"Vibe", scanVibe, 2, 0, 0},         // home skill dir + project AGENTS.md
		{"Goose", scanGoose, 2, 0, 0},       // home config + project .goosehints
		{"Antigravity", scanAntigravity, 6, 0, 0},
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
			expectedEnforcement := EnforcementUnverified
			expectedAdvisory := false
			if tc.name == "Antigravity" {
				expectedEnforcement = EnforcementAdvisory
				expectedAdvisory = true
			}
			if result.Enforcement != expectedEnforcement {
				t.Errorf("Expected enforcement %q, got %q", expectedEnforcement, result.Enforcement)
			}
			if result.Advisory != expectedAdvisory {
				t.Errorf("Expected advisory %v, got %v", expectedAdvisory, result.Advisory)
			}
			if tc.name != "Antigravity" {
				if result.EnforcementEvidence != "" {
					t.Errorf("Expected no enforcement evidence for %s, got %q", tc.name, result.EnforcementEvidence)
				}
				if len(result.Evidence) != 0 {
					t.Errorf("Expected no structured evidence for %s, got %+v", tc.name, result.Evidence)
				}
				if result.Warning != "" {
					t.Errorf("Expected no warning for %s, got %q", tc.name, result.Warning)
				}
			}
		})
	}
}

// WO-77: enforcing/advisory posture requires valid probe evidence and otherwise normalizes.
func TestNormalizeEnforcementRequiresEvidence(t *testing.T) {
	result := AgentResult{Enforcement: EnforcementAdvisory}
	result.NormalizeEnforcement()
	if result.Enforcement != EnforcementUnverified {
		t.Errorf("Expected advisory without evidence to normalize to unverified, got %q", result.Enforcement)
	}
	if result.Advisory {
		t.Errorf("Expected advisory alias to be false after normalization")
	}
	if len(result.Evidence) != 0 || result.Warning != "" || result.EnforcementEvidence != "" {
		t.Errorf("Expected advisory without evidence to normalize to plain unverified, got %+v", result)
	}

	result = AgentResult{
		Enforcement: EnforcementEnforcing,
		Warning:     SecurityProbeSelfReportWarning,
		Evidence: []EvidenceItem{
			{Kind: EvidenceVendorDocs, Note: "vendor docs claim secure mode"},
			{Kind: EvidenceAgentSelfReport, Note: "agent said YES"},
		},
	}
	result.NormalizeEnforcement()
	if result.Enforcement != EnforcementUnverified {
		t.Errorf("Expected enforcing with invalid evidence kinds to normalize to unverified, got %q", result.Enforcement)
	}
	if len(result.Evidence) != 0 || result.Warning != "" || result.EnforcementEvidence != "" {
		t.Errorf("Expected invalid-only evidence to normalize to plain unverified, got %+v", result)
	}

	result = AgentResult{
		Enforcement: EnforcementUnverified,
		Warning:     SecurityProbeSelfReportWarning,
		Evidence: []EvidenceItem{
			{Kind: EvidenceAgentSelfReport, Note: "agent said YES"},
		},
	}
	result.NormalizeEnforcement()
	if result.Enforcement != EnforcementUnverified {
		t.Errorf("Expected explicit unverified to stay unverified, got %q", result.Enforcement)
	}
	if len(result.Evidence) != 0 || result.Warning != "" || result.EnforcementEvidence != "" {
		t.Errorf("Expected explicit unverified to normalize to plain state, got %+v", result)
	}

	result = AgentResult{
		Enforcement: EnforcementAdvisory,
		Warning:     SecurityProbeSelfReportWarning,
		Evidence: []EvidenceItem{
			{Kind: EvidenceRealToolResult, Note: "read outside workspace returned a real payload"},
		},
	}
	result.NormalizeEnforcement()
	if result.Enforcement != EnforcementAdvisory {
		t.Errorf("Expected advisory with real-tool evidence to stay advisory, got %q", result.Enforcement)
	}
	if result.EnforcementEvidence != "read outside workspace returned a real payload" {
		t.Errorf("Expected legacy evidence summary, got %q", result.EnforcementEvidence)
	}
	if result.Warning != SecurityProbeSelfReportWarning || !result.Advisory || len(result.Evidence) != 1 {
		t.Errorf("Expected valid advisory evidence and warning to remain, got %+v", result)
	}
}

// WO-66/WO-67: Antigravity scanner coverage includes SKILL.md filtering.
func TestScanAntigravity(t *testing.T) {
	mfs := &mockFileSystem{
		files: map[string]string{
			"home/.gemini/GEMINI.md":                                   "global rule",
			"home/.gemini/antigravity-cli/skills/review/SKILL.md":      "global skill",
			"home/.gemini/antigravity-cli/skills/missing/README.md":    "not a skill",
			"home/.gemini/antigravity-cli/skills/nested/deep/SKILL.md": "not immediate",
			"home/.gemini/antigravity-cli/global_workflows/release.md": "global workflow",
			"home/.gemini/antigravity-cli/workflows/compat.md":         "compat workflow",
			"project/AGENTS.md": "project rule",
			"project/.antigravitycli/skills/project/SKILL.md":       "project skill",
			"project/.antigravitycli/skills/empty/README.md":        "not a skill",
			"project/.antigravitycli/workflows/project-workflow.md": "project workflow",
		},
		dirs: []string{
			"home/.gemini/antigravity-cli/skills/review",
			"home/.gemini/antigravity-cli/skills/missing",
			"home/.gemini/antigravity-cli/skills/empty",
			"home/.gemini/antigravity-cli/skills/nested/deep",
			"home/.gemini/antigravity-cli/global_workflows",
			"home/.gemini/antigravity-cli/workflows",
			"project/.antigravitycli/skills/project",
			"project/.antigravitycli/skills/empty",
			"project/.antigravitycli/workflows",
		},
	}
	tempRoot := mfs.setup(t)
	homeDir := filepath.Join(tempRoot, "home")
	projectDir := filepath.Join(tempRoot, "project")

	result := scanAntigravity(projectDir, homeDir)
	sort.Strings(result.Sources)

	expectedSources := []string{
		"./.antigravitycli/skills/ (advisory)",
		"./.antigravitycli/workflows/ (advisory)",
		"./AGENTS.md (advisory)",
		"~/.gemini/GEMINI.md (advisory)",
		"~/.gemini/antigravity-cli/global_workflows/ (advisory)",
		"~/.gemini/antigravity-cli/skills/ (advisory)",
		"~/.gemini/antigravity-cli/workflows/ (advisory)",
	}

	if result.Name != AgentAntigravity {
		t.Errorf("Expected name %s, got %s", AgentAntigravity, result.Name)
	}
	if !result.Advisory {
		t.Errorf("Expected Antigravity result to be advisory")
	}
	if result.Enforcement != EnforcementAdvisory {
		t.Errorf("Expected enforcement %q, got %q", EnforcementAdvisory, result.Enforcement)
	}
	if result.EnforcementEvidence != enforcementEvidenceSummary(antigravityEvidence) {
		t.Errorf("Expected antigravity enforcement evidence, got %q", result.EnforcementEvidence)
	}
	if len(result.Evidence) != 3 {
		t.Fatalf("Expected 3 evidence items, got %d", len(result.Evidence))
	}
	expectedKinds := []EvidenceKind{EvidenceRealToolResult, EvidenceUnfakeableOutput, EvidenceAgentSelfReport}
	for i, want := range expectedKinds {
		if result.Evidence[i].Kind != want {
			t.Errorf("Expected evidence kind %q at %d, got %q", want, i, result.Evidence[i].Kind)
		}
	}
	if result.Warning != SecurityProbeSelfReportWarning {
		t.Errorf("Expected warning %q, got %q", SecurityProbeSelfReportWarning, result.Warning)
	}
	if result.ConfigDir != "~/.gemini/antigravity-cli" {
		t.Errorf("Expected config_dir ~/.gemini/antigravity-cli, got %s", result.ConfigDir)
	}
	if result.Skills != 7 {
		t.Errorf("Expected 7 skills, got %d", result.Skills)
	}
	if result.Hooks != 0 {
		t.Errorf("Expected 0 hooks, got %d", result.Hooks)
	}
	if result.MCP != 0 {
		t.Errorf("Expected 0 MCP, got %d", result.MCP)
	}
	if !reflect.DeepEqual(result.Sources, expectedSources) {
		t.Errorf("Expected sources %v, got %v", expectedSources, result.Sources)
	}
	countedBytes := int64(len("global rule") + len("global skill") + len("global workflow") +
		len("compat workflow") + len("project rule") + len("project skill") +
		len("project workflow"))
	if result.Tokens != bytesToTokens(countedBytes) {
		t.Errorf("Expected %d tokens, got %d", bytesToTokens(countedBytes), result.Tokens)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}
	if !jsonFieldExists(encoded, "enforcement") ||
		!jsonFieldExists(encoded, "enforcement_evidence") ||
		!jsonFieldExists(encoded, "evidence") ||
		!jsonFieldExists(encoded, "warning") {
		t.Errorf("Expected JSON output to include enforcement fields: %s", encoded)
	}
	expectedInvalid := []string{
		"~/.gemini/antigravity-cli/skills/empty",
		"~/.gemini/antigravity-cli/skills/missing",
		"~/.gemini/antigravity-cli/skills/nested",
		"./.antigravitycli/skills/empty",
	}
	if len(result.InvalidLocations) != len(expectedInvalid) {
		t.Errorf("Expected %d invalid locations, got %d", len(expectedInvalid), len(result.InvalidLocations))
	}
	for _, path := range expectedInvalid {
		if !invalidLocationExists(result.InvalidLocations, AgentAntigravity, path, "missing required file SKILL.md") {
			t.Errorf("Expected invalid location %s", path)
		}
	}
	if skillFilePathExists(result.SkillFiles, "~/.gemini/antigravity-cli/skills/missing") {
		t.Errorf("Expected missing SKILL.md directory to be ignored")
	}
	if skillFilePathExists(result.SkillFiles, "~/.gemini/antigravity-cli/skills/empty") {
		t.Errorf("Expected empty directory to be ignored")
	}
	if skillFilePathExists(result.SkillFiles, "~/.gemini/antigravity-cli/skills/nested") {
		t.Errorf("Expected nested-only SKILL.md directory to be ignored")
	}
	if ContextWindow(AgentAntigravity) != 1_000_000 {
		t.Errorf("Expected Antigravity context window 1000000, got %d", ContextWindow(AgentAntigravity))
	}
}

// WO-70: invalid-only Antigravity skill roots must not emit a token-only agent.
func TestScanAntigravityInvalidSkillRootsDoNotEmitAgent(t *testing.T) {
	mfs := &mockFileSystem{
		files: map[string]string{
			"home/.gemini/antigravity-cli/skills/scratch/README.md": "not a skill",
			"project/.antigravitycli/skills/scratch/README.md":      "not a skill",
		},
		dirs: []string{
			"home/.gemini/antigravity-cli/skills/scratch",
			"project/.antigravitycli/skills/scratch",
		},
	}
	tempRoot := mfs.setup(t)
	homeDir := filepath.Join(tempRoot, "home")
	projectDir := filepath.Join(tempRoot, "project")

	result, err := ScanWithHome(projectDir, homeDir)
	if err != nil {
		t.Fatalf("ScanWithHome returned error: %v", err)
	}
	if agentResultExists(result.Agents, AgentAntigravity) {
		t.Errorf("Expected invalid-only Antigravity skill roots to be ignored")
	}
	if len(result.InvalidLocations) != 2 {
		t.Fatalf("Expected 2 invalid Antigravity locations, got %d", len(result.InvalidLocations))
	}
	if !invalidLocationExists(result.InvalidLocations, AgentAntigravity, "~/.gemini/antigravity-cli/skills/scratch", "missing required file SKILL.md") {
		t.Errorf("Expected invalid home Antigravity skill root")
	}
	if !invalidLocationExists(result.InvalidLocations, AgentAntigravity, "./.antigravitycli/skills/scratch", "missing required file SKILL.md") {
		t.Errorf("Expected invalid project Antigravity skill root")
	}
}

// WO-68: Optional Antigravity workflow directories must be absent-safe.
func TestScanAntigravityWithoutWorkflows(t *testing.T) {
	mfs := &mockFileSystem{
		files: map[string]string{
			"home/.gemini/GEMINI.md":                              "global rule",
			"home/.gemini/antigravity-cli/skills/review/SKILL.md": "global skill",
			"project/AGENTS.md":                                   "project rule",
			"project/.antigravitycli/skills/project/SKILL.md":     "project skill",
		},
		dirs: []string{
			"home/.gemini/antigravity-cli/skills/review",
			"project/.antigravitycli/skills/project",
		},
	}
	tempRoot := mfs.setup(t)
	homeDir := filepath.Join(tempRoot, "home")
	projectDir := filepath.Join(tempRoot, "project")

	result := scanAntigravity(projectDir, homeDir)

	if result.Skills != 4 {
		t.Errorf("Expected 4 skills, got %d", result.Skills)
	}
	if skillFilePathExists(result.SkillFiles, "~/.gemini/antigravity-cli/global_workflows") {
		t.Errorf("Expected missing global_workflows directory to be ignored")
	}
	if skillFilePathExists(result.SkillFiles, "~/.gemini/antigravity-cli/workflows") {
		t.Errorf("Expected missing workflows directory to be ignored")
	}
	if skillFilePathExists(result.SkillFiles, "./.antigravitycli/workflows") {
		t.Errorf("Expected missing project workflows directory to be ignored")
	}
}

// WO-68: Candidate workflow paths may alias the same directory.
func TestScanAntigravityDeduplicatesWorkflowAliases(t *testing.T) {
	mfs := &mockFileSystem{
		files: map[string]string{
			"home/.gemini/GEMINI.md":                                   "global rule",
			"home/.gemini/antigravity-cli/global_workflows/release.md": "global workflow",
		},
		dirs: []string{
			"home/.gemini/antigravity-cli/global_workflows",
		},
	}
	tempRoot := mfs.setup(t)
	homeDir := filepath.Join(tempRoot, "home")
	projectDir := filepath.Join(tempRoot, "project")

	target := filepath.Join(homeDir, ".gemini/antigravity-cli/global_workflows")
	link := filepath.Join(homeDir, ".gemini/antigravity-cli/workflows")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	result := scanAntigravity(projectDir, homeDir)

	if result.Skills != 2 {
		t.Errorf("Expected 2 skills, got %d", result.Skills)
	}
}

func skillFilePathExists(files []SkillFile, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func agentResultExists(agents []AgentResult, name string) bool {
	for _, agent := range agents {
		if agent.Name == name {
			return true
		}
	}
	return false
}

func invalidLocationExists(locations []InvalidLocation, agent, path, reason string) bool {
	for _, loc := range locations {
		if loc.Agent == agent && loc.Path == path && loc.Reason == reason {
			return true
		}
	}
	return false
}

func jsonFieldExists(data []byte, field string) bool {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	_, ok := raw[field]
	return ok
}
