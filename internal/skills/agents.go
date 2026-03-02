package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// parseCodexTOMLMCP counts [mcp_servers.*] table sections in a Codex config.toml.
func parseCodexTOMLMCP(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[mcp_servers.") && strings.HasSuffix(line, "]") && !strings.HasPrefix(line, "[[") {
			count++
		}
	}
	return count
}

// parseOpenCodeJSON reads an opencode.json and returns instruction count, MCP count, byte size, and whether the file exists.
func parseOpenCodeJSON(path string) (instructions, mcp int, bytes int64, found bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, false
	}
	b := fileBytes(path)
	var cfg struct {
		MCP          map[string]json.RawMessage `json:"mcp"`
		Instructions []json.RawMessage          `json:"instructions"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, 0, b, true
	}
	return len(cfg.Instructions), len(cfg.MCP), b, true
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

func scanOpenCode(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentOpenCode, Advisory: true}
	var totalBytes int64

	if homeDir != "" {
		instructions, mcp, b, found := parseOpenCodeJSON(filepath.Join(homeDir, ".config", "opencode", "opencode.json"))
		if found {
			r.Skills += instructions
			r.MCP += mcp
			totalBytes += b
			r.Sources = append(r.Sources, "~/.config/opencode/ (advisory)")
		}
	}

	instructions, mcp, b, found := parseOpenCodeJSON(filepath.Join(projectDir, "opencode.json"))
	if found {
		r.Skills += instructions
		r.MCP += mcp
		totalBytes += b
		r.Sources = append(r.Sources, "opencode.json (advisory)")
	}

	r.Tokens = bytesToTokens(totalBytes)
	return r
}

func scanCodex(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentCodex, Advisory: true}
	var bytes int64

	if homeDir != "" {
		homeAgents := filepath.Join(homeDir, ".codex", "AGENTS.md")
		if _, err := os.Stat(homeAgents); err == nil {
			r.Skills++
			r.Sources = append(r.Sources, "~/.codex/AGENTS.md (advisory)")
		}
		bytes += fileBytes(homeAgents)

		homeSkillsDir := filepath.Join(homeDir, ".codex", "skills")
		homeSkillCount := countSkillDirs(homeSkillsDir)
		r.Skills += homeSkillCount
		if homeSkillCount > 0 {
			r.Sources = append(r.Sources, "~/.codex/skills/ (advisory)")
		}
		bytes += dirBytesRecursive(homeSkillsDir)

		configPath := filepath.Join(homeDir, ".codex", "config.toml")
		mcpCount := parseCodexTOMLMCP(configPath)
		if mcpCount > 0 {
			r.MCP += mcpCount
			r.Sources = append(r.Sources, "~/.codex/config.toml (advisory)")
		}
		bytes += fileBytes(configPath)
	}

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

func scanOpenClaw(_ string, homeDir string) AgentResult {
	r := AgentResult{Name: AgentOpenClaw, Advisory: true}
	var bytes int64

	if homeDir == "" {
		return r
	}

	homeSkillsDir := filepath.Join(homeDir, ".openclaw", "skills")
	homeCount := countSkillDirs(homeSkillsDir)
	r.Skills += homeCount
	if homeCount > 0 {
		r.Sources = append(r.Sources, "~/.openclaw/skills/ (advisory)")
	}
	bytes += dirBytesRecursive(homeSkillsDir)

	cfgPath := filepath.Join(homeDir, ".openclaw", "openclaw.json")
	mcpCount := parseMCPServers(cfgPath)
	if mcpCount > 0 {
		r.MCP += mcpCount
		r.Sources = append(r.Sources, "~/.openclaw/openclaw.json (advisory)")
	}
	bytes += fileBytes(cfgPath)

	mcporterPath := filepath.Join(homeDir, ".openclaw", "config", "mcporter.json")
	mcporterCount := parseMCPServers(mcporterPath)
	if mcporterCount > 0 {
		r.MCP += mcporterCount
		r.Sources = append(r.Sources, "~/.openclaw/config/mcporter.json (advisory)")
	}
	bytes += fileBytes(mcporterPath)

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanWindsurf(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentWindsurf}
	var bytes int64

	// Project: .windsurfrules (single file).
	rulesFile := filepath.Join(projectDir, ".windsurfrules")
	if fb := fileBytes(rulesFile); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, ".windsurfrules")
		bytes += fb
	}

	// Project: .windsurf/rules/ (directory of rule files).
	projRulesDir := filepath.Join(projectDir, ".windsurf", "rules")
	projCount := countFiles(projRulesDir)
	if projCount > 0 {
		r.Skills += projCount
		r.Sources = append(r.Sources, ".windsurf/rules/")
	}
	bytes += dirBytes(projRulesDir)

	if homeDir != "" {
		// Global: ~/.windsurf/rules/.
		homeRulesDir := filepath.Join(homeDir, ".windsurf", "rules")
		homeCount := countFiles(homeRulesDir)
		if homeCount > 0 {
			r.Skills += homeCount
			r.Sources = append(r.Sources, "~/.windsurf/rules/")
		}
		bytes += dirBytes(homeRulesDir)

		// MCP: ~/.codeium/windsurf/mcp_config.json.
		mcpPath := filepath.Join(homeDir, ".codeium", "windsurf", "mcp_config.json")
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

func scanAider(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentAider, Advisory: true}
	var bytes int64

	// Project: .aider.conf.yml.
	projConf := filepath.Join(projectDir, ".aider.conf.yml")
	if fb := fileBytes(projConf); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, ".aider.conf.yml (advisory)")
		bytes += fb
	}

	if homeDir != "" {
		// Global: ~/.aider.conf.yml.
		homeConf := filepath.Join(homeDir, ".aider.conf.yml")
		if fb := fileBytes(homeConf); fb > 0 {
			r.Skills++
			r.Sources = append(r.Sources, "~/.aider.conf.yml (advisory)")
			bytes += fb
		}
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanContinue(projectDir, homeDir string) AgentResult {
	r := AgentResult{Name: AgentContinue, Advisory: true}
	var bytes int64

	if homeDir != "" {
		// Global: ~/.continue/config.yaml.
		yamlConf := filepath.Join(homeDir, ".continue", "config.yaml")
		if fb := fileBytes(yamlConf); fb > 0 {
			r.Skills++
			r.Sources = append(r.Sources, "~/.continue/config.yaml (advisory)")
			bytes += fb
		}

		// Global: ~/.continue/config.json (deprecated).
		jsonConf := filepath.Join(homeDir, ".continue", "config.json")
		if fb := fileBytes(jsonConf); fb > 0 {
			r.Skills++
			r.Sources = append(r.Sources, "~/.continue/config.json (advisory)")
			bytes += fb
		}
	}

	// Project: .continuerc.json.
	projConf := filepath.Join(projectDir, ".continuerc.json")
	if fb := fileBytes(projConf); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, ".continuerc.json (advisory)")
		bytes += fb
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}

func scanCopilot(projectDir, _ string) AgentResult {
	r := AgentResult{Name: AgentCopilot}
	var bytes int64

	// Project: .github/copilot-instructions.md.
	instrFile := filepath.Join(projectDir, ".github", "copilot-instructions.md")
	if fb := fileBytes(instrFile); fb > 0 {
		r.Skills++
		r.Sources = append(r.Sources, ".github/copilot-instructions.md")
		bytes += fb
	}

	r.Tokens = bytesToTokens(bytes)
	return r
}
