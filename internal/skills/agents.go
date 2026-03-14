package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// agentPathSpec defines the configuration for scanning an agent's files.
type agentPathSpec struct {
	Name     AgentName
	Advisory bool
	Home     []pathSpec
	Project  []pathSpec
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
	var totalBytes int64

	process := func(baseDir, path, sourcePrefix, comment string, pt pathType, recursive bool, parseFunc func(string, *AgentResult) (bool, int64)) {
		if baseDir == "" {
			return
		}

		fullPath := resolvePath(filepath.Join(baseDir, path))
		found := false
		var size int64

		if parseFunc != nil {
			found, size = parseFunc(fullPath, &r)
		} else {
			switch pt {
			case pathTypeFile:
				if s := fileBytes(fullPath); s > 0 {
					r.Skills++
					found = true
					size = s
				}
			case pathTypeDirSkills:
				count := countSkillDirs(fullPath)
				if count > 0 {
					r.Skills += count
					found = true
				}
				if recursive {
					size = dirBytesRecursive(fullPath)
				} else {
					size = dirBytes(fullPath)
				}
			case pathTypeDirFiles:
				count := countFiles(fullPath)
				if count > 0 {
					r.Skills += count
					found = true
				}
				if recursive {
					size = dirBytesRecursive(fullPath)
				} else {
					size = dirBytes(fullPath)
				}
			case pathTypeCustom:
				// Custom handling for cursor .mdc files
				entries, err := os.ReadDir(fullPath)
				if err == nil {
					for _, e := range entries {
						if !e.IsDir() && filepath.Ext(e.Name()) == ".mdc" {
							r.Skills++
							found = true
						}
					}
				}
				size = dirBytes(fullPath)
			}
		}

		if found {
			source := sourcePrefix + path
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
	spec := agentPathSpec{
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
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCline(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name: AgentCline,
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
		Name: AgentCursor,
		Home: []pathSpec{
			{
				Path: ".cursor/mcp.json", SourcePrefix: "~/",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					mcp := parseMCPServers(path)
					if mcp > 0 {
						r.MCP += mcp
						return true, fileBytes(path)
					}
					return false, 0
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
		Name:     AgentOpenCode,
		Advisory: true,
		Home: []pathSpec{
			{
				Path: ".config/opencode/opencode.json", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					skills, mcp, bytes, found := parseOpenCodeJSON(path)
					r.Skills += skills
					r.MCP += mcp
					return found, bytes
				},
			},
		},
		Project: []pathSpec{
			{
				Path: "opencode.json", SourcePrefix: "./", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					skills, mcp, bytes, found := parseOpenCodeJSON(path)
					r.Skills += skills
					r.MCP += mcp
					return found, bytes
				},
			},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCodex(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:     AgentCodex,
		Advisory: true,
		Home: []pathSpec{
			{Path: ".codex/AGENTS.md", SourcePrefix: "~/", Type: pathTypeFile, Comment: "(advisory)"},
			{Path: ".codex/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true, Comment: "(advisory)"},
			{
				Path: ".codex/config.toml", SourcePrefix: "~/", Comment: "(advisory)",
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
		Project: []pathSpec{
			{Path: "AGENTS.md", SourcePrefix: "./", Type: pathTypeFile, Comment: "(advisory)"},
			{Path: ".codex", SourcePrefix: "./", Type: pathTypeDirFiles},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanQwen(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:     AgentQwen,
		Advisory: true,
		Home: []pathSpec{
			{Path: ".qwen/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true, Comment: "(advisory)"},
			{
				Path: ".qwen/settings.json", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					mcp := parseMCPServers(path)
					if mcp > 0 {
						r.MCP += mcp
						return true, fileBytes(path)
					}
					return false, 0
				},
			},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanOpenClaw(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:     AgentOpenClaw,
		Advisory: true,
		Home: []pathSpec{
			{Path: ".openclaw/skills", SourcePrefix: "~/", Type: pathTypeDirSkills, RecursiveSize: true, Comment: "(advisory)"},
			{
				Path: ".openclaw/openclaw.json", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					mcp := parseMCPServers(path)
					if mcp > 0 {
						r.MCP += mcp
						return true, fileBytes(path)
					}
					return false, 0
				},
			},
			{
				Path: ".openclaw/config/mcporter.json", SourcePrefix: "~/", Comment: "(advisory)",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					mcp := parseMCPServers(path)
					if mcp > 0 {
						r.MCP += mcp
						return true, fileBytes(path)
					}
					return false, 0
				},
			},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanWindsurf(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name: AgentWindsurf,
		Home: []pathSpec{
			{Path: ".windsurf/rules", SourcePrefix: "~/", Type: pathTypeDirFiles},
			{
				Path: ".codeium/windsurf/mcp_config.json", SourcePrefix: "~/",
				Parse: func(path string, r *AgentResult) (bool, int64) {
					mcp := parseMCPServers(path)
					if mcp > 0 {
						r.MCP += mcp
						return true, fileBytes(path)
					}
					return false, 0
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
		Name:     AgentAider,
		Advisory: true,
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
		Name:     AgentContinue,
		Advisory: true,
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
		Name: AgentCopilot,
		Project: []pathSpec{
			{Path: ".github/copilot-instructions.md", SourcePrefix: "./", Type: pathTypeFile},
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}
