package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	missingRequiredSkillFileReasonPrefix = "missing required file "
)

var antigravityEvidence = []EvidenceItem{
	{
		Kind: EvidenceRealToolResult,
		Note: "trustedWorkspaces does not confine reads to workspace",
	},
	{
		Kind: EvidenceUnfakeableOutput,
		Note: "outside-workspace /tmp read returned a UUID-verified probe payload",
	},
	{
		Kind: EvidenceAgentSelfReport,
		Note: "YES/NO self-report probes are unreliable: agy hallucinated success for a TCC-blocked file",
	},
}

// WO-90: vendor docs can prove autonomy mode existence, not enforcement.
func vendorAutonomy(mode, disables, source string) []AutonomyCapability {
	return []AutonomyCapability{
		{
			Mode:       mode,
			Disables:   disables,
			SourceKind: AutonomySourceVendorDocs,
			Source:     source,
		},
	}
}

// agentPathSpec defines the configuration for scanning an agent's files.
type agentPathSpec struct {
	Name        AgentName
	Autonomy    []AutonomyCapability // WO-90: documented prompt-disabling modes
	Enforcement Enforcement
	Evidence    []EvidenceItem
	Warning     string
	ConfigDir   string
	Home        []pathSpec
	Project     []pathSpec
}

type pathSpec struct {
	Path          string
	Type          pathType
	FileExt       string // For file-based skills with specific extensions
	SourcePrefix  string // e.g., "~/"
	RecursiveSize bool
	RequiredFile  string // WO-67: require a marker file before counting a skill directory
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

func skillDirHome(path, comment string) pathSpec {
	return pathSpec{
		Path:          path,
		SourcePrefix:  "~/",
		Type:          pathTypeDirSkills,
		RecursiveSize: true,
		Comment:       comment,
	}
}

func skillDirProject(path, comment string) pathSpec {
	return pathSpec{
		Path:          path,
		SourcePrefix:  "./",
		Type:          pathTypeDirSkills,
		RecursiveSize: true,
		Comment:       comment,
	}
}

// WO-67: Some agents require a marker file before a skill directory is active.
func skillDirHomeRequiredFile(path, requiredFile, comment string) pathSpec {
	spec := skillDirHome(path, comment)
	spec.RequiredFile = requiredFile
	return spec
}

// WO-67: Some agents require a marker file before a skill directory is active.
func skillDirProjectRequiredFile(path, requiredFile, comment string) pathSpec {
	spec := skillDirProject(path, comment)
	spec.RequiredFile = requiredFile
	return spec
}

func dirFilesHome(path, comment string) pathSpec {
	return pathSpec{
		Path:         path,
		SourcePrefix: "~/",
		Type:         pathTypeDirFiles,
		Comment:      comment,
	}
}

func dirFilesProject(path, comment string) pathSpec {
	return pathSpec{
		Path:         path,
		SourcePrefix: "./",
		Type:         pathTypeDirFiles,
		Comment:      comment,
	}
}

func configFileHome(path, comment string, parse func(string, *AgentResult) (bool, int64)) pathSpec {
	return pathSpec{
		Path:         path,
		SourcePrefix: "~/",
		Parse:        parse,
		Comment:      comment,
	}
}

func configFileProject(path, comment string, parse func(string, *AgentResult) (bool, int64)) pathSpec {
	return pathSpec{
		Path:         path,
		SourcePrefix: "./",
		Parse:        parse,
		Comment:      comment,
	}
}

func fileHome(path, comment string) pathSpec {
	return pathSpec{
		Path:         path,
		SourcePrefix: "~/",
		Type:         pathTypeFile,
		Comment:      comment,
	}
}

func fileProject(path, comment string) pathSpec {
	return pathSpec{
		Path:         path,
		SourcePrefix: "./",
		Type:         pathTypeFile,
		Comment:      comment,
	}
}

func customDirProject(path, comment string) pathSpec {
	return pathSpec{
		Path:         path,
		SourcePrefix: "./",
		Type:         pathTypeCustom,
		Comment:      comment,
	}
}

func scanAgentPaths(projectDir, homeDir string, spec agentPathSpec) AgentResult {
	r := AgentResult{
		Name:        spec.Name,
		Autonomy:    append([]AutonomyCapability(nil), spec.Autonomy...),
		Enforcement: spec.Enforcement,
		Evidence:    append([]EvidenceItem(nil), spec.Evidence...),
		Warning:     spec.Warning,
	}

	// Set config_dir from home directory paths
	if homeDir != "" && spec.ConfigDir != "" {
		r.ConfigDir = filepath.Join("~", spec.ConfigDir)
	}

	var totalBytes int64
	seenResolvedPaths := make(map[string]struct{})

	process := func(baseDir, path, sourcePrefix, comment string, pt pathType, recursive bool, requiredFile string, parseFunc func(string, *AgentResult) (bool, int64)) {
		if baseDir == "" {
			return
		}

		fullPath := resolvePath(filepath.Join(baseDir, path))
		if _, ok := seenResolvedPaths[fullPath]; ok {
			// WO-68: candidate Antigravity workflow paths may alias the same directory.
			return
		}
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
				if requiredFile != "" {
					skillDirs = listSkillDirsContaining(fullPath, requiredFile)
					r.InvalidLocations = append(r.InvalidLocations,
						invalidSkillDirs(spec.Name, fullPath, sourcePrefix+path, requiredFile)...)
				} else {
					skillDirs = listSkillDirs(fullPath)
				}
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
				if requiredFile != "" {
					size = skillDirBytes(fullPath, skillDirs, recursive)
				} else if recursive {
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
			seenResolvedPaths[fullPath] = struct{}{}
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
		process(homeDir, p.Path, p.SourcePrefix, p.Comment, p.Type, p.RecursiveSize, p.RequiredFile, p.Parse)
	}
	for _, p := range spec.Project {
		process(projectDir, p.Path, p.SourcePrefix, p.Comment, p.Type, p.RecursiveSize, p.RequiredFile, p.Parse)
	}

	r.Tokens = bytesToTokens(totalBytes)
	r.NormalizeAutonomy()
	r.NormalizeEnforcement()
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

// WO-67: agy skill roots only load immediate child directories with SKILL.md.
func listSkillDirsContaining(dir, requiredFile string) []string {
	var dirs []string
	for _, skillDir := range listSkillDirs(dir) {
		if regularFileExists(filepath.Join(dir, skillDir, requiredFile)) {
			dirs = append(dirs, skillDir)
		}
	}
	return dirs
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// WO-72: expose required-marker candidate dirs that exist but are not valid skills.
func invalidSkillDirs(agent, dir, sourcePath, requiredFile string) []InvalidLocation {
	var invalid []InvalidLocation
	for _, skillDir := range listSkillDirs(dir) {
		if regularFileExists(filepath.Join(dir, skillDir, requiredFile)) {
			continue
		}
		invalid = append(invalid, InvalidLocation{
			Agent:  agent,
			Path:   sourcePath + "/" + skillDir,
			Reason: missingRequiredSkillFileReasonPrefix + requiredFile,
		})
	}
	return invalid
}

// WO-70: required-marker skill roots should count only dirs that became skills.
func skillDirBytes(dir string, skillDirs []string, recursive bool) int64 {
	var total int64
	for _, skillDir := range skillDirs {
		path := filepath.Join(dir, skillDir)
		if recursive {
			total += dirBytesRecursive(path)
			continue
		}
		total += dirBytes(path)
	}
	return total
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
		Name: AgentClaudeCode,
		Autonomy: vendorAutonomy(
			"--dangerously-skip-permissions",
			"permission prompts",
			"Claude Code permission modes docs: https://code.claude.com/docs/en/permission-modes",
		),
		ConfigDir: ".claude",
		Home: []pathSpec{
			configFileHome(".claude/settings.json", "", parseClaudeSettings),
			skillDirHome(".claude/skills", ""),
			fileHome(".claude/CLAUDE.md", ""),
		},
		Project: []pathSpec{
			skillDirProject(".claude/skills", ""),
			fileProject("CLAUDE.md", ""),
			fileProject(".claude/settings.local.json", ""),
			fileProject("CLAUDE.local.md", ""),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCline(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentCline,
		ConfigDir: ".cline",
		Home: []pathSpec{
			skillDirHome(".cline/skills", ""),
		},
		Project: []pathSpec{
			dirFilesProject(".clinerules", ""),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCursor(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name: AgentCursor,
		// WO-92: Cursor citation must stay on a specific auto-run source.
		Autonomy: vendorAutonomy(
			"Agent Auto-run",
			"tool approval prompts for auto-run surfaces",
			"Cursor Agent overview docs (Terminal Integration): https://cursor.com/docs/agent/overview#terminal-integration; mentions configurable auto-run or confirmation",
		),
		ConfigDir: ".cursor",
		Home: []pathSpec{
			configFileHome(".cursor/mcp.json", "", parseMCPServers),
		},
		Project: []pathSpec{
			customDirProject(".cursor/rules", ""),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanOpenCode(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentOpenCode,
		ConfigDir: ".config/opencode",
		Home: []pathSpec{
			configFileHome(".config/opencode/opencode.json", "(advisory)", parseOpenCodeJSON),
			dirFilesHome(".config/opencode/commands", "(advisory)"),
			skillDirHome(".config/opencode/skills", "(advisory)"),
		},
		Project: []pathSpec{
			configFileProject("opencode.json", "(advisory)", parseOpenCodeJSON),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCodex(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name: AgentCodex,
		Autonomy: vendorAutonomy(
			"--full-auto",
			"edit and command approval prompts within the selected sandbox mode",
			"OpenAI Codex CLI approval modes docs: https://help.openai.com/en/articles/11096431",
		),
		ConfigDir: ".codex",
		Home: []pathSpec{
			fileHome(".codex/AGENTS.md", "(advisory)"),
			skillDirHome(".codex/skills", "(advisory)"),
			configFileHome(".codex/config.toml", "(advisory)", parseCodexTOMLMCP),
		},
		Project: []pathSpec{
			fileProject("AGENTS.md", "(advisory)"),
			dirFilesProject(".codex", ""),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanQwen(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentQwen,
		ConfigDir: ".qwen",
		Home: []pathSpec{
			skillDirHome(".qwen/skills", "(advisory)"),
			configFileHome(".qwen/settings.json", "(advisory)", parseMCPServers),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanOpenClaw(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentOpenClaw,
		ConfigDir: ".openclaw",
		Home: []pathSpec{
			skillDirHome(".openclaw/skills", "(advisory)"),
			configFileHome(".openclaw/openclaw.json", "(advisory)", parseMCPServers),
			configFileHome(".openclaw/config/mcporter.json", "(advisory)", parseMCPServers),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanWindsurf(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentWindsurf,
		ConfigDir: ".windsurf",
		Home: []pathSpec{
			dirFilesHome(".windsurf/rules", ""),
			configFileHome(".codeium/windsurf/mcp_config.json", "", parseMCPServers),
		},
		Project: []pathSpec{
			fileProject(".windsurfrules", ""),
			dirFilesProject(".windsurf/rules", ""),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanAider(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name: AgentAider,
		Autonomy: vendorAutonomy(
			"--yes-always",
			"confirmation prompts",
			"Aider options reference: https://aider.chat/docs/config/options.html",
		),
		ConfigDir: ".aider",
		Home: []pathSpec{
			fileHome(".aider.conf.yml", "(advisory)"),
			skillDirHome(".aider/skills", "(advisory)"),
		},
		Project: []pathSpec{
			fileProject(".aider.conf.yml", "(advisory)"),
			fileProject("CONVENTIONS.md", "(advisory)"),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanContinue(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentContinue,
		ConfigDir: ".continue",
		Home: []pathSpec{
			fileHome(".continue/config.yaml", "(advisory)"),
			fileHome(".continue/config.json", "(advisory)"),
		},
		Project: []pathSpec{
			fileProject(".continuerc.json", "(advisory)"),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanCopilot(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentCopilot,
		ConfigDir: ".copilot",
		Home: []pathSpec{
			skillDirHome(".copilot/skills", ""),
		},
		Project: []pathSpec{
			fileProject(".github/copilot-instructions.md", ""),
			fileProject("AGENTS.md", ""),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanKilocode(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name: AgentKilocode,
		Autonomy: vendorAutonomy(
			"kilo run --auto",
			"user interaction prompts in autonomous mode",
			"Kilo CLI autonomous mode docs: https://kilo.ai/docs/code-with-ai/platforms/cli",
		),
		ConfigDir: ".kilocode",
		Home: []pathSpec{
			skillDirHome(".kilocode/skills", ""),
			configFileHome(".config/kilo/opencode.json", "", parseOpenCodeJSON),
			configFileHome(".config/kilo/kilo.jsonc", "", parseOpenCodeJSON), // WO-86: Kilo CLI config; absent = not-found
		},
		Project: []pathSpec{
			dirFilesProject(".kilocode/rules", ""),
			skillDirProject(".kilocode/skills", ""),
			configFileProject("kilo.jsonc", "", parseOpenCodeJSON), // WO-86: project-level Kilo CLI config
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanVibe(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentVibe,
		ConfigDir: ".vibe",
		Home: []pathSpec{
			skillDirHome(".vibe/skills", "(advisory)"),
		},
		Project: []pathSpec{
			fileProject("AGENTS.md", "(advisory)"),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

func scanGoose(projectDir, homeDir string) AgentResult {
	spec := agentPathSpec{
		Name:      AgentGoose,
		ConfigDir: ".config/goose",
		Home: []pathSpec{
			fileHome(".config/goose/config.yaml", "(advisory)"),
			skillDirHome(".config/goose/skills", "(advisory)"),
		},
		Project: []pathSpec{
			fileProject(".goosehints", "(advisory)"),
		},
	}
	return scanAgentPaths(projectDir, homeDir, spec)
}

// WO-77: Antigravity posture is based on real probes, not agent self-report.
func scanAntigravity(projectDir, homeDir string) AgentResult {
	const antigravitySkillFile = "SKILL.md"

	spec := agentPathSpec{
		Name: AgentAntigravity,
		Autonomy: vendorAutonomy(
			"--dangerously-skip-permissions",
			"permission prompts",
			"Google Antigravity CLI docs: https://antigravity.google/docs/cli-using",
		),
		Enforcement: EnforcementAdvisory,
		Evidence:    antigravityEvidence,
		Warning:     SecurityProbeSelfReportWarning,
		ConfigDir:   ".gemini/antigravity-cli",
		Home: []pathSpec{
			fileHome(".gemini/GEMINI.md", "(advisory)"),
			skillDirHomeRequiredFile(".gemini/antigravity-cli/skills", antigravitySkillFile, "(advisory)"),
			dirFilesHome(".gemini/antigravity-cli/global_workflows", "(advisory)"), // WO-68: legacy global workflow name.
			dirFilesHome(".gemini/antigravity-cli/workflows", "(advisory)"),        // WO-68: agy CLI workflow candidate.
		},
		Project: []pathSpec{
			fileProject("AGENTS.md", "(advisory)"),
			skillDirProjectRequiredFile(".antigravitycli/skills", antigravitySkillFile, "(advisory)"),
			dirFilesProject(".antigravitycli/workflows", "(advisory)"),
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
