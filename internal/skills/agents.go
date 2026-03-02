package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// fileBytes returns the size of a single file in bytes, or 0 on error.
func fileBytes(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// dirBytes returns the total size of all regular files in a directory (non-recursive).
// Uses os.Stat to follow symlinks and get the target file size.
func dirBytes(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var total int64
	for _, e := range entries {
		info, err := os.Stat(filepath.Join(dir, e.Name()))
		if err != nil || info.IsDir() {
			continue
		}
		total += info.Size()
	}
	return total
}

// dirBytesRecursive returns the total size of all regular files under dir, recursively.
// Follows symlinks so that symlinked directories and files are included.
func dirBytesRecursive(dir string) int64 {
	return dirBytesRecursiveDepth(dir, 10)
}

func dirBytesRecursiveDepth(dir string, maxDepth int) int64 {
	if maxDepth <= 0 {
		return 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var total int64
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			total += dirBytesRecursiveDepth(path, maxDepth-1)
		} else {
			total += info.Size()
		}
	}
	return total
}

// bytesToTokens converts a byte count to an approximate token count (bytes / 4).
func bytesToTokens(b int64) int64 {
	return b / 4
}

// countSkillDirs counts subdirectories in a skills directory.
// Follows symlinks so that symlinked skill directories are counted.
func countSkillDirs(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
			continue
		}
		if e.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(filepath.Join(dir, e.Name()))
			if err == nil && info.IsDir() {
				count++
			}
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
	var bytes int64

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

		bytes += fileBytes(globalSettings)
		bytes += dirBytesRecursive(globalSkillsDir)
		bytes += fileBytes(filepath.Join(homeDir, ".claude", "CLAUDE.md"))
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

	bytes += fileBytes(localSettings)
	bytes += dirBytesRecursive(projectSkillsDir)
	bytes += fileBytes(filepath.Join(projectDir, "CLAUDE.md"))
	bytes += fileBytes(filepath.Join(projectDir, "CLAUDE.local.md"))

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanCline(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentCline}
	var bytes int64

	if homeDir != "" {
		homeSkillsDir := filepath.Join(homeDir, ".cline", "skills")
		homeCount := countSkillDirs(homeSkillsDir)
		r.Skills += homeCount
		if homeCount > 0 {
			r.Sources = append(r.Sources, "~/.cline/skills/")
		}
		bytes += dirBytesRecursive(homeSkillsDir)
	}

	rulesDir := filepath.Join(projectDir, ".clinerules")
	count := countFiles(rulesDir)
	if count > 0 {
		r.Skills += count
		r.Sources = append(r.Sources, ".clinerules/")
	}
	bytes += dirBytes(rulesDir)

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanCursor(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentCursor}
	var bytes int64

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
	bytes += dirBytes(rulesDir)

	if homeDir != "" {
		mcpPath := filepath.Join(homeDir, ".cursor", "mcp.json")
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
		r.Tokens = bytesToTokens(fileBytes(cfgPath))
		return r
	}

	r.Skills = len(cfg.Instructions)
	r.MCP = len(cfg.MCP)

	r.Tokens = bytesToTokens(fileBytes(cfgPath))
	return r
}

func scanCodex(projectDir, _ string) AgentResult {
	r := AgentResult{Name: AgentCodex, Advisory: true}
	var bytes int64

	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	if _, err := os.Stat(agentsPath); err == nil {
		r.Skills++
		r.Sources = append(r.Sources, "AGENTS.md (advisory)")
	}
	bytes += fileBytes(agentsPath)

	codexDir := filepath.Join(projectDir, ".codex")
	count := countFiles(codexDir)
	if count > 0 {
		r.Skills += count
		r.Sources = append(r.Sources, ".codex/")
	}
	bytes += dirBytes(codexDir)

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanQwen(_ string, homeDir string) AgentResult {
	r := AgentResult{Name: AgentQwen, Advisory: true}
	var bytes int64

	if homeDir == "" {
		return r
	}

	homeSkillsDir := filepath.Join(homeDir, ".qwen", "skills")
	homeCount := countSkillDirs(homeSkillsDir)
	r.Skills += homeCount
	if homeCount > 0 {
		r.Sources = append(r.Sources, "~/.qwen/skills/ (advisory)")
	}
	bytes += dirBytesRecursive(homeSkillsDir)

	cfgPath := filepath.Join(homeDir, ".qwen", "settings.json")
	mcpCount := parseMCPServers(cfgPath)
	if mcpCount > 0 {
		r.MCP = mcpCount
		r.Sources = append(r.Sources, "~/.qwen/settings.json (advisory)")
	}
	bytes += fileBytes(cfgPath)

	r.Tokens = bytesToTokens(bytes)
	return r
}
