package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// countSkillDirs counts subdirectories in a skills directory.
func countSkillDirs(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	return count
}

// countFiles counts regular files in a directory.
func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}

// parseClaudeSettings reads a Claude settings.json and counts hooks and MCP servers.
func parseClaudeSettings(path string) (hooks, mcp int, found bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, false
	}

	var settings struct {
		Hooks      map[string]json.RawMessage `json:"hooks"`
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return 0, 0, true
	}

	for _, eventRaw := range settings.Hooks {
		var matchers []struct {
			Hooks []json.RawMessage `json:"hooks"`
		}
		if err := json.Unmarshal(eventRaw, &matchers); err != nil {
			continue
		}
		for _, m := range matchers {
			hooks += len(m.Hooks)
		}
	}

	mcp = len(settings.MCPServers)
	return hooks, mcp, true
}

// parseMCPServers reads a JSON file and counts entries under "mcpServers".
func parseMCPServers(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var cfg struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0
	}
	return len(cfg.MCPServers)
}

func scanClaudeCode(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentClaudeCode}

	if homeDir != "" {
		globalSkillsDir := filepath.Join(homeDir, ".claude", "skills")
		r.Skills += countSkillDirs(globalSkillsDir)

		globalSettings := filepath.Join(homeDir, ".claude", "settings.json")
		hooks, mcp, found := parseClaudeSettings(globalSettings)
		if found {
			r.Hooks += hooks
			r.MCP += mcp
			r.Sources = append(r.Sources, "~/.claude/settings.json")
		}
		if r.Skills > 0 {
			r.Sources = append(r.Sources, "~/.claude/skills/")
		}
	}

	projectSkillsDir := filepath.Join(projectDir, ".claude", "skills")
	projSkillCount := countSkillDirs(projectSkillsDir)
	r.Skills += projSkillCount
	if projSkillCount > 0 {
		r.Sources = append(r.Sources, ".claude/skills/")
	}

	localSettings := filepath.Join(projectDir, ".claude", "settings.local.json")
	if _, err := os.Stat(localSettings); err == nil {
		r.Sources = append(r.Sources, ".claude/settings.local.json")
	}

	return r
}

func scanCline(projectDir, _ string) AgentResult {
	r := AgentResult{Name: AgentCline}

	rulesDir := filepath.Join(projectDir, ".clinerules")
	count := countFiles(rulesDir)
	if count > 0 {
		r.Skills = count
		r.Sources = append(r.Sources, ".clinerules/")
	}

	return r
}

func scanCursor(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentCursor}

	rulesDir := filepath.Join(projectDir, ".cursor", "rules")
	entries, err := os.ReadDir(rulesDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".mdc" {
				r.Skills++
			}
		}
		if r.Skills > 0 {
			r.Sources = append(r.Sources, ".cursor/rules/")
		}
	}

	if homeDir != "" {
		mcpPath := filepath.Join(homeDir, ".cursor", "mcp.json")
		mcpCount := parseMCPServers(mcpPath)
		if mcpCount > 0 {
			r.MCP += mcpCount
			r.Sources = append(r.Sources, "~/.cursor/mcp.json")
		}
	}

	return r
}

func scanOpenCode(_ string, homeDir string) AgentResult {
	r := AgentResult{Name: AgentOpenCode, Advisory: true}

	if homeDir == "" {
		return r
	}

	cfgPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return r
	}

	r.Sources = append(r.Sources, "~/.config/opencode/ (advisory)")

	var cfg struct {
		MCP          map[string]json.RawMessage `json:"mcp"`
		Instructions []json.RawMessage          `json:"instructions"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return r
	}

	r.Skills = len(cfg.Instructions)
	r.MCP = len(cfg.MCP)

	return r
}

func scanCodex(projectDir, _ string) AgentResult {
	r := AgentResult{Name: AgentCodex, Advisory: true}

	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	if _, err := os.Stat(agentsPath); err == nil {
		r.Skills++
		r.Sources = append(r.Sources, "AGENTS.md (advisory)")
	}

	codexDir := filepath.Join(projectDir, ".codex")
	count := countFiles(codexDir)
	if count > 0 {
		r.Skills += count
		r.Sources = append(r.Sources, ".codex/")
	}

	return r
}

func scanQwen(_ string, homeDir string) AgentResult {
	r := AgentResult{Name: AgentQwen, Advisory: true}

	if homeDir == "" {
		return r
	}

	cfgPath := filepath.Join(homeDir, ".qwen", "settings.json")
	mcpCount := parseMCPServers(cfgPath)
	if mcpCount > 0 {
		r.MCP = mcpCount
		r.Sources = append(r.Sources, "~/.qwen/settings.json (advisory)")
	}

	return r
}
