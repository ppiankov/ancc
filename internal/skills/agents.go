package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// agentPathSpec defines the configuration for scanning an agent's files.
type agentPathSpec struct {
	Name      AgentName
	Advisory  bool
	ConfigDir string
	Home      []pathSpec
	Project   []pathSpec
}

type pathSpec struct {
	Path          string
	Type          pathType
	FileExt       string // For file-based skills with specific extensions
	SourcePrefix  string // e.g., "~/"
	RecursiveSize bool
	Parse         func(path string, r *AgentResult) (found bool, size int64)
	Comment       string
}

type pathType int

const (
	pathTypeFile pathType = iota
	pathTypeDirSkills
	pathTypeDirFiles
	pathTypeCustom
)

func scanAgentPaths(projectDir, homeDir string, spec agentPathSpec) AgentResult {
	r := AgentResult{Name: spec.Name, Advisory: spec.Advisory}

	// Set config_dir from home directory paths
	if homeDir != "" && spec.ConfigDir != "" {
		r.ConfigDir = filepath.Join("~", spec.ConfigDir)
	}

	var totalBytes int64

	process := func(baseDir, path, sourcePrefix, comment string, pt pathType, recursive bool, parseFunc func(string, *AgentResult) (bool, int64)) {
		if baseDir == "" {
			return
		}

		fullPath := resolvePath(filepath.Join(baseDir, path))
		found := false
		var size int64
		var skillDirs []string
		var files []string
		var mdcFiles []string

		if parseFunc != nil {
			found, size = parseFunc(fullPath, &r)
		} else {
			switch pt {
			case pathTypeFile:
				if s := fileBytes(fullPath); s > 0 {
					r.Skills++
					r.SkillFiles = append(r.SkillFiles, SkillFile{Path: sourcePrefix + path, Type: "file"})
					found = true
					size = s
				}
			case pathTypeDirSkills:
				skillDirs = listSkillDirs(fullPath)
				if len(skillDirs) > 0 {
					r.Skills += len(skillDirs)
					for _, sd := range skillDirs {
						r.SkillFiles = append(r.SkillFiles, SkillFile{
							Path: sourcePrefix + path + "/" + sd,
							Type: "dir",
						})
					}
					found = true
				}
				if recursive {
					size = dirBytesRecursive(fullPath)
				} else {
					size = dirBytes(fullPath)
				}
			case pathTypeDirFiles:
				files = listFiles(fullPath)
				if len(files) > 0 {
					r.Skills += len(files)
					for _, f := range files {
						r.SkillFiles = append(r.SkillFiles, SkillFile{
							Path: sourcePrefix + path + "/" + f,
							Type: "file",
						})
					}
					found = true
				}
				if recursive {
					size = dirBytesRecursive(fullPath)
				} else {
					size = dirBytes(fullPath)
				}
			case pathTypeCustom:
				// Custom handling for cursor .mdc files
				mdcFiles = listMdcFiles(fullPath)
				if len(mdcFiles) > 0 {
					r.Skills += len(mdcFiles)
					for _, f := range mdcFiles {
						r.SkillFiles = append(r.SkillFiles, SkillFile{
							Path: sourcePrefix + path + "/" + f,
							Type: "file",
						})
					}
					found = true
				}
				size = dirBytes(fullPath)
			}
		}

		if found {
			source := sourcePrefix + path
			// Add trailing slash for directory types
			if pt == pathTypeDirSkills || pt == pathTypeDirFiles || pt == pathTypeCustom {
				source += "/"
			}
			if comment != "" {
				source += " " + comment
			}
			r.Sources = append(r.Sources, source)
		}
		totalBytes += size
	}

	for _, p := range spec.Home {
		process(homeDir, p.Path, p.SourcePrefix, p.Comment, p.Type, p.RecursiveSize, p.Parse)
	}
	for _, p := range spec.Project {
		process(projectDir, p.Path, p.SourcePrefix, p.Comment, p.Type, p.RecursiveSize, p.Parse)
	}

	r.Tokens = bytesToTokens(totalBytes)
	return r
}

// resolvePath resolves symlinks in a path if it exists, otherwise returns the original path.
func resolvePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

// fileBytes returns the size of a single file in bytes, or 0 on error.
func fileBytes(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// dirBytes returns the total size of all regular files in a directory (non-recursive).
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

// listSkillDirs returns subdirectory names in a skills directory.
func listSkillDirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
			continue
		}
		if e.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(filepath.Join(dir, e.Name()))
			if err == nil && info.IsDir() {
				dirs = append(dirs, e.Name())
			}
		}
	}
	return dirs
}

// listFiles returns regular file names in a directory.
func listFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files
}

// listMdcFiles returns .mdc file names in a directory.
func listMdcFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".mdc" {
			files = append(files, e.Name())
		}
	}
	return files
}

// parseClaudeSettings reads a Claude settings.json and extracts hooks and MCP servers.
func parseClaudeSettings(path string, r *AgentResult) (found bool, size int64) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, 0
	}

	var settings struct {
		Hooks      map[string]json.RawMessage `json:"hooks"`
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return true, fileBytes(path)
	}

	// Extract hooks
	for event, eventRaw := range settings.Hooks {
		var matchers []struct {
			Hooks []json.RawMessage `json:"hooks"`
		}
		if err := json.Unmarshal(eventRaw, &matchers); err != nil {
			continue
		}
		for _, m := range matchers {
			for range m.Hooks {
				r.Hooks++
				r.HookConfigs = append(r.HookConfigs, HookConfig{Event: event, Path: path})
			}
		}
	}

	// Extract MCP servers
	for name := range settings.MCPServers {
		r.MCP++
		r.MCPServers = append(r.MCPServers, MCPServer{Name: name, Path: path})
	}

	return true, fileBytes(path)
}

// parseCodexTOMLMCP counts [mcp_servers.*] table sections in a Codex config.toml.
func parseCodexTOMLMCP(path string, r *AgentResult) (found bool, size int64) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[mcp_servers.") && strings.HasSuffix(line, "]") && !strings.HasPrefix(line, "[[") {
			count++
			serverName := strings.TrimSuffix(strings.TrimPrefix(line, "[mcp_servers."), "]")
			r.MCP++
			r.MCPServers = append(r.MCPServers, MCPServer{Name: serverName, Path: path})
		}
	}
	return count > 0, fileBytes(path)
}

// parseOpenCodeJSON reads an opencode.json and extracts instructions and MCP servers.
func parseOpenCodeJSON(path string, r *AgentResult) (found bool, size int64) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, 0
	}
	b := fileBytes(path)
	var cfg struct {
		MCP          map[string]json.RawMessage `json:"mcp"`
		Instructions []json.RawMessage          `json:"instructions"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return true, b
	}

	for range cfg.Instructions {
		r.Skills++
	}
	for name := range cfg.MCP {
		r.MCP++
		r.MCPServers = append(r.MCPServers, MCPServer{Name: name, Path: path})
	}

	return true, b
}

// parseMCPServers reads a JSON file and extracts MCP server entries.
func parseMCPServers(path string, r *AgentResult) (found bool, size int64) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, 0
	}
	var cfg struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false, 0
	}

	for name := range cfg.MCPServers {
		r.MCP++
		r.MCPServers = append(r.MCPServers, MCPServer{Name: name, Path: path})
	}

	return len(cfg.MCPServers) > 0, fileBytes(path)
}

func scanClaudeCode(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
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
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCline(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentCline,
		ConfigDir: ".cline",
		Home: []pathSpec{
			{Path: ".cline/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true},
		},
		Project: []pathSpec{
			{Path: ".clinerules", SourcePrefix: "./", Type: pathTypeDirFiles},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCursor(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentCursor,
		ConfigDir: ".cursor",
		Home: []pathSpec{
			{
				Path: ".cursor/mcp.json", SourcePrefix: "~/",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseMCPServers(path, r)
				},
			},
		},
		Project: []pathSpec{
			{Path: ".cursor/rules", SourcePrefix: "./", Type: pathTypeCustom},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanOpenCode(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentOpenCode,
		Advisory:  true,
		ConfigDir: ".config/opencode",
		Home: []pathSpec{
			{
				Path: ".config/opencode/opencode.json", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseOpenCodeJSON(path, r)
				},
			},
			{Path: ".config/opencode/commands", SourcePrefix: "~/", Type: pathTypeDirFiles, Comment: "(advisory)"},
			{Path: ".config/opencode/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true, Comment: "(advisory)"},
		},
		Project: []pathSpec{
			{
				Path: "opencode.json", SourcePrefix: "./", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseOpenCodeJSON(path, r)
				},
			},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCodex(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentCodex,
		Advisory:  true,
		ConfigDir: ".codex",
		Home: []pathSpec{
			{Path: ".codex/AGENTS.md", SourcePrefix: "~/", Type: pathTypeFile, Comment: "(advisory)"},
			{Path: ".codex/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true, Comment: "(advisory)"},
			{
				Path: ".codex/config.toml", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseCodexTOMLMCP(path, r)
				},
			},
		},
		Project: []pathSpec{
			{Path: "AGENTS.md", SourcePrefix: "./", Type: pathTypeFile, Comment: "(advisory)"},
			{Path: ".codex", SourcePrefix: "./", Type: pathTypeDirFiles},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanQwen(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentQwen,
		Advisory:  true,
		ConfigDir: ".qwen",
		Home: []pathSpec{
			{Path: ".qwen/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true, Comment: "(advisory)"},
			{
				Path: ".qwen/settings.json", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseMCPServers(path, r)
				},
			},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanOpenClaw(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentOpenClaw,
		Advisory:  true,
		ConfigDir: ".openclaw",
		Home: []pathSpec{
			{Path: ".openclaw/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true, Comment: "(advisory)"},
			{
				Path: ".openclaw/openclaw.json", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseMCPServers(path, r)
				},
			},
			{
				Path: ".openclaw/config/mcporter.json", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseMCPServers(path, r)
				},
			},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanWindsurf(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentWindsurf,
		ConfigDir: ".windsurf",
		Home: []pathSpec{
			{Path: ".windsurf/rules", SourcePrefix: "~/", Type: pathTypeDirFiles},
			{
				Path: ".codeium/windsurf/mcp_config.json", SourcePrefix: "~/",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseMCPServers(path, r)
				},
			},
		},
		Project: []pathSpec{
			{Path: ".windsurfrules", SourcePrefix: "./", Type: pathTypeFile},
			{Path: ".windsurf/rules", SourcePrefix: "./", Type: pathTypeDirFiles},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanAider(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentAider,
		Advisory:  true,
		ConfigDir: "",
		Home: []pathSpec{
			{Path: ".aider.conf.yml", SourcePrefix: "~/", Type: pathTypeFile, Comment: "(advisory)"},
		},
		Project: []pathSpec{
			{Path: ".aider.conf.yml", SourcePrefix: "./", Type: pathTypeFile, Comment: "(advisory)"},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanContinue(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentContinue,
		Advisory:  true,
		ConfigDir: ".continue",
		Home: []pathSpec{
			{Path: ".continue/config.yaml", SourcePrefix: "~/", Type: pathTypeFile, Comment: "(advisory)"},
			{Path: ".continue/config.json", SourcePrefix: "~/", Type: pathTypeFile, Comment: "(advisory)"},
		},
		Project: []pathSpec{
			{Path: ".continuerc.json", SourcePrefix: "./", Type: pathTypeFile, Comment: "(advisory)"},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCopilot(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentCopilot,
		ConfigDir: "",
		Project: []pathSpec{
			{Path: ".github/copilot-instructions.md", SourcePrefix: "./", Type: pathTypeFile},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanKilocode(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentKilocode,
		ConfigDir: ".kilocode",
		Home: []pathSpec{
			{Path: ".kilocode/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true},
			{
				Path: ".config/kilo/opencode.json", SourcePrefix: "~/",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					return parseOpenCodeJSON(path, r)
				},
			},
		},
		Project: []pathSpec{
			{Path: ".kilocode/rules", SourcePrefix: "./", Type: pathTypeDirFiles},
			{Path: ".kilocode/skills", SourcePrefix: "./", Type: pathTypeDirSkills, RecursiveSize: true},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

// Backward-compatible helper functions for tests (old signatures).

// parseClaudeSettingsLegacy reads a Claude settings.json and counts hooks and MCP servers (old signature for tests).
func parseClaudeSettingsLegacy(path string) (hooks, mcp int, found bool) {
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

// parseCodexTOMLMCPLegacy counts [mcp_servers.*] table sections in a Codex config.toml (old signature for tests).
func parseCodexTOMLMCPLegacy(path string) int {
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

// parseMCPServersLegacy reads a JSON file and counts entries under "mcpServers" (old signature for tests).
func parseMCPServersLegacy(path string) int {
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

// countSkillDirs counts subdirectories in a skills directory (for tests).
func countSkillDirs(dir string) int {
	return len(listSkillDirs(dir))
}

// countFiles counts regular files in a directory (for tests).
func countFiles(dir string) int {
	return len(listFiles(dir))
}

// parseOpenCodeJSONLegacy reads an opencode.json and returns instruction count, MCP count, byte size, and whether the file exists (old signature for tests).
func parseOpenCodeJSONLegacy(path string) (instructions, mcp int, bytes int64, found bool) {
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
