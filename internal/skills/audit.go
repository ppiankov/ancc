package skills

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// AuditStatus represents the result of an audit check.
type AuditStatus string

const (
	AuditOK    AuditStatus = "ok"
	AuditWarn  AuditStatus = "warn"
	AuditError AuditStatus = "error"
)

// AuditEntry is a single audit finding.
type AuditEntry struct {
	Category string      `json:"category"`
	Name     string      `json:"name"`
	Status   AuditStatus `json:"status"`
	Message  string      `json:"message"`
	Path     string      `json:"path,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

// AgentAudit is the audit result for one agent.
type AgentAudit struct {
	Name    string       `json:"name"`
	Entries []AuditEntry `json:"entries"`
}

// AuditSummary holds aggregate counts.
type AuditSummary struct {
	Total  int `json:"total"`
	OK     int `json:"ok"`
	Warn   int `json:"warn"`
	Errors int `json:"errors"`
}

// AuditResult is the top-level audit output.
type AuditResult struct {
	Path        string       `json:"path"`
	Agents      []AgentAudit `json:"agents"`
	Environment []AuditEntry `json:"environment,omitempty"`
	Budget      []AuditEntry `json:"budget,omitempty"`
	Summary     AuditSummary `json:"summary"`
}

// auditEnv holds injectable dependencies for testing.
type auditEnv struct {
	lookPath func(string) (string, error)
	homeDir  string
	readDir  func(string) ([]os.DirEntry, error)
	stat     func(string) (os.FileInfo, error)
	goos     string
}

func defaultAuditEnv(homeDir string) *auditEnv {
	return &auditEnv{
		lookPath: exec.LookPath,
		homeDir:  homeDir,
		readDir:  os.ReadDir,
		stat:     os.Stat,
		goos:     runtime.GOOS,
	}
}

// expandTilde replaces a leading ~/ with the home directory path.
func expandTilde(path, homeDir string) string {
	if homeDir != "" && strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// Audit scans and audits the given directory and user home.
func Audit(projectDir string) (*AuditResult, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	return AuditWithHome(absDir, homeDir, defaultAuditEnv(homeDir))
}

// AuditWithHome is the testable core with explicit home directory and environment.
func AuditWithHome(projectDir, homeDir string, env *auditEnv) (*AuditResult, error) {
	result := &AuditResult{Path: projectDir}

	type auditor struct {
		name string
		fn   func(string, string, *auditEnv) AgentAudit
	}

	auditors := []auditor{
		{AgentClaudeCode, auditClaudeCode},
		{AgentCline, auditCline},
		{AgentCursor, auditCursor},
		{AgentOpenCode, auditOpenCode},
		{AgentCodex, auditCodex},
		{AgentQwen, auditQwen},
		{AgentOpenClaw, auditOpenClaw},
	}

	for _, a := range auditors {
		agent := a.fn(projectDir, homeDir, env)
		if len(agent.Entries) > 0 {
			result.Agents = append(result.Agents, agent)
		}
	}

	result.Environment = auditEnvironment(env)

	scanResult, _ := ScanWithHome(projectDir, homeDir)
	if scanResult != nil {
		result.Budget = auditBudget(scanResult, homeDir, projectDir)
	}

	for _, agent := range result.Agents {
		for _, e := range agent.Entries {
			result.Summary.Total++
			switch e.Status {
			case AuditOK:
				result.Summary.OK++
			case AuditWarn:
				result.Summary.Warn++
			case AuditError:
				result.Summary.Errors++
			}
		}
	}
	for _, e := range result.Environment {
		result.Summary.Total++
		switch e.Status {
		case AuditOK:
			result.Summary.OK++
		case AuditWarn:
			result.Summary.Warn++
		case AuditError:
			result.Summary.Errors++
		}
	}
	for _, e := range result.Budget {
		result.Summary.Total++
		switch e.Status {
		case AuditOK:
			result.Summary.OK++
		case AuditWarn:
			result.Summary.Warn++
		case AuditError:
			result.Summary.Errors++
		}
	}

	return result, nil
}

// --- MCP audit helpers ---

// mcpDetail holds parsed MCP server info.
type mcpDetail struct {
	name    string
	command string
}

// parseMCPDetails extracts individual MCP server names and commands from a JSON config.
func parseMCPDetails(path, key string) []mcpDetail {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	serversRaw, ok := raw[key]
	if !ok {
		return nil
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(serversRaw, &servers); err != nil {
		return nil
	}

	var details []mcpDetail
	for name, serverRaw := range servers {
		var server struct {
			Command json.RawMessage `json:"command"`
		}
		if err := json.Unmarshal(serverRaw, &server); err != nil {
			details = append(details, mcpDetail{name: name})
			continue
		}
		cmd := extractCommand(server.Command)
		details = append(details, mcpDetail{name: name, command: cmd})
	}

	sort.Slice(details, func(i, j int) bool {
		return details[i].name < details[j].name
	})
	return details
}

// extractCommand gets the binary name from a JSON command field (string or array).
func extractCommand(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0]
	}
	return ""
}

// auditMCPEntries checks each MCP server command exists in PATH.
func auditMCPEntries(details []mcpDetail, sourceLabel string, env *auditEnv) []AuditEntry {
	var entries []AuditEntry
	for _, d := range details {
		if d.command == "" {
			entries = append(entries, AuditEntry{
				Category: "mcp",
				Name:     d.name,
				Status:   AuditWarn,
				Message:  "no command specified",
				Path:     sourceLabel,
			})
			continue
		}

		cmd := expandTilde(d.command, env.homeDir)
		if filepath.IsAbs(cmd) {
			if _, err := os.Stat(cmd); err != nil {
				entries = append(entries, AuditEntry{
					Category: "mcp",
					Name:     d.name,
					Status:   AuditError,
					Message:  fmt.Sprintf("%s (not found)", d.command),
					Path:     sourceLabel,
				})
			} else {
				entries = append(entries, AuditEntry{
					Category: "mcp",
					Name:     d.name,
					Status:   AuditOK,
					Message:  fmt.Sprintf("%s (found)", d.command),
					Path:     sourceLabel,
				})
			}
		} else {
			_, err := env.lookPath(cmd)
			if err != nil {
				entries = append(entries, AuditEntry{
					Category: "mcp",
					Name:     d.name,
					Status:   AuditError,
					Message:  fmt.Sprintf("%s (not found in PATH)", d.command),
					Path:     sourceLabel,
				})
			} else {
				entries = append(entries, AuditEntry{
					Category: "mcp",
					Name:     d.name,
					Status:   AuditOK,
					Message:  fmt.Sprintf("%s (found)", d.command),
					Path:     sourceLabel,
				})
			}
		}
	}
	return entries
}

// parseCodexMCPDetails extracts MCP server details from a Codex config.toml.
func parseCodexMCPDetails(path string) []mcpDetail {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var details []mcpDetail
	var currentName string

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[mcp_servers.") && strings.HasSuffix(line, "]") && !strings.HasPrefix(line, "[[") {
			currentName = strings.TrimPrefix(line, "[mcp_servers.")
			currentName = strings.TrimSuffix(currentName, "]")
			details = append(details, mcpDetail{name: currentName})
		}
		if currentName != "" && strings.HasPrefix(line, "command") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				cmd := strings.TrimSpace(parts[1])
				cmd = strings.Trim(cmd, `"`)
				details[len(details)-1].command = cmd
			}
		}
	}

	return details
}

// --- Hook audit helpers ---

// hookDetail holds parsed hook info.
type hookDetail struct {
	event   string
	matcher string
	command string
}

// parseClaudeHookDetails extracts individual hook entries from Claude settings.json.
func parseClaudeHookDetails(path string) []hookDetail {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var settings struct {
		Hooks map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}

	var details []hookDetail
	for event, eventRaw := range settings.Hooks {
		var matchers []struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		}
		if err := json.Unmarshal(eventRaw, &matchers); err != nil {
			continue
		}
		for _, m := range matchers {
			for _, h := range m.Hooks {
				details = append(details, hookDetail{
					event:   event,
					matcher: m.Matcher,
					command: h.Command,
				})
			}
		}
	}

	sort.Slice(details, func(i, j int) bool {
		if details[i].event != details[j].event {
			return details[i].event < details[j].event
		}
		return details[i].matcher < details[j].matcher
	})

	return details
}

// auditHookEntries checks each hook command is findable.
func auditHookEntries(details []hookDetail, env *auditEnv) []AuditEntry {
	var entries []AuditEntry
	for _, d := range details {
		name := d.event
		if d.matcher != "" {
			name += "/" + d.matcher
		}

		if d.command == "" {
			entries = append(entries, AuditEntry{
				Category: "hook",
				Name:     name,
				Status:   AuditWarn,
				Message:  "no command specified",
			})
			continue
		}

		// Extract binary name (first token of command string).
		bin := strings.Fields(d.command)[0]
		// Expand ~/... paths to absolute.
		bin = expandTilde(bin, env.homeDir)

		// Check if it's an absolute/relative path or a PATH lookup.
		if filepath.IsAbs(bin) {
			if _, err := os.Stat(bin); err != nil {
				entries = append(entries, AuditEntry{
					Category: "hook",
					Name:     name,
					Status:   AuditError,
					Message:  fmt.Sprintf("%s (not found)", d.command),
				})
			} else {
				entries = append(entries, AuditEntry{
					Category: "hook",
					Name:     name,
					Status:   AuditOK,
					Message:  fmt.Sprintf("%s (found)", d.command),
				})
			}
		} else {
			_, err := env.lookPath(bin)
			if err != nil {
				entries = append(entries, AuditEntry{
					Category: "hook",
					Name:     name,
					Status:   AuditError,
					Message:  fmt.Sprintf("%s (not found in PATH)", d.command),
				})
			} else {
				entries = append(entries, AuditEntry{
					Category: "hook",
					Name:     name,
					Status:   AuditOK,
					Message:  fmt.Sprintf("%s (found)", d.command),
				})
			}
		}
	}
	return entries
}

// --- Skill audit helpers ---

// auditSkillDirs checks each subdirectory in a skills dir for emptiness.
func auditSkillDirs(dir, sourceLabel string) []AuditEntry {
	entries_list, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []AuditEntry
	for _, e := range entries_list {
		isDir := e.IsDir()
		if !isDir && e.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(filepath.Join(dir, e.Name()))
			if err == nil && info.IsDir() {
				isDir = true
			}
		}
		if !isDir {
			continue
		}

		subdir := filepath.Join(dir, e.Name())
		subEntries, err := os.ReadDir(subdir)
		if err != nil {
			results = append(results, AuditEntry{
				Category: "skill",
				Name:     e.Name(),
				Status:   AuditWarn,
				Message:  "unreadable",
				Path:     sourceLabel,
			})
			continue
		}

		fileCount := 0
		for _, se := range subEntries {
			if !se.IsDir() {
				fileCount++
			}
		}

		if fileCount == 0 {
			results = append(results, AuditEntry{
				Category: "skill",
				Name:     e.Name(),
				Status:   AuditWarn,
				Message:  "empty (no files)",
				Path:     sourceLabel,
			})
		} else {
			results = append(results, AuditEntry{
				Category: "skill",
				Name:     e.Name(),
				Status:   AuditOK,
				Message:  fmt.Sprintf("%d files", fileCount),
				Path:     sourceLabel,
			})
		}
	}
	return results
}

// auditSkillFiles checks each file in a rules directory.
func auditSkillFiles(dir, sourceLabel string) []AuditEntry {
	entries_list, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []AuditEntry
	for _, e := range entries_list {
		if e.IsDir() {
			continue
		}
		info, err := os.Stat(filepath.Join(dir, e.Name()))
		if err != nil {
			results = append(results, AuditEntry{
				Category: "skill",
				Name:     e.Name(),
				Status:   AuditWarn,
				Message:  "unreadable",
				Path:     sourceLabel,
			})
			continue
		}
		if info.Size() == 0 {
			results = append(results, AuditEntry{
				Category: "skill",
				Name:     e.Name(),
				Status:   AuditWarn,
				Message:  "empty file (0 bytes)",
				Path:     sourceLabel,
			})
		} else {
			results = append(results, AuditEntry{
				Category: "skill",
				Name:     e.Name(),
				Status:   AuditOK,
				Message:  fmt.Sprintf("%d bytes", info.Size()),
				Path:     sourceLabel,
			})
		}
	}
	return results
}

// --- Per-agent audit functions ---

func auditClaudeCode(projectDir, homeDir string, env *auditEnv) AgentAudit {
	a := AgentAudit{Name: AgentClaudeCode}

	if homeDir != "" {
		settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
		a.Entries = append(a.Entries, auditHookEntries(parseClaudeHookDetails(settingsPath), env)...)
		a.Entries = append(a.Entries, auditMCPEntries(parseMCPDetails(settingsPath, "mcpServers"), "~/.claude/settings.json", env)...)
		a.Entries = append(a.Entries, auditSkillDirs(filepath.Join(homeDir, ".claude", "skills"), "~/.claude/skills/")...)
	}

	a.Entries = append(a.Entries, auditSkillDirs(filepath.Join(projectDir, ".claude", "skills"), ".claude/skills/")...)

	return a
}

func auditCline(projectDir, homeDir string, env *auditEnv) AgentAudit {
	a := AgentAudit{Name: AgentCline}

	if homeDir != "" {
		a.Entries = append(a.Entries, auditSkillDirs(filepath.Join(homeDir, ".cline", "skills"), "~/.cline/skills/")...)
	}

	a.Entries = append(a.Entries, auditSkillFiles(filepath.Join(projectDir, ".clinerules"), ".clinerules/")...)

	return a
}

func auditCursor(projectDir, homeDir string, env *auditEnv) AgentAudit {
	a := AgentAudit{Name: AgentCursor}

	rulesDir := filepath.Join(projectDir, ".cursor", "rules")
	a.Entries = append(a.Entries, auditSkillFiles(rulesDir, ".cursor/rules/")...)

	if homeDir != "" {
		mcpPath := filepath.Join(homeDir, ".cursor", "mcp.json")
		a.Entries = append(a.Entries, auditMCPEntries(parseMCPDetails(mcpPath, "mcpServers"), "~/.cursor/mcp.json", env)...)
	}

	return a
}

func auditOpenCode(projectDir, homeDir string, env *auditEnv) AgentAudit {
	a := AgentAudit{Name: AgentOpenCode}

	if homeDir != "" {
		homeCfg := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
		a.Entries = append(a.Entries, auditMCPEntries(parseMCPDetails(homeCfg, "mcp"), "~/.config/opencode/opencode.json", env)...)
	}

	projCfg := filepath.Join(projectDir, "opencode.json")
	a.Entries = append(a.Entries, auditMCPEntries(parseMCPDetails(projCfg, "mcp"), "opencode.json", env)...)

	return a
}

func auditCodex(projectDir, homeDir string, env *auditEnv) AgentAudit {
	a := AgentAudit{Name: AgentCodex}

	if homeDir != "" {
		configPath := filepath.Join(homeDir, ".codex", "config.toml")
		a.Entries = append(a.Entries, auditMCPEntries(parseCodexMCPDetails(configPath), "~/.codex/config.toml", env)...)
		a.Entries = append(a.Entries, auditSkillDirs(filepath.Join(homeDir, ".codex", "skills"), "~/.codex/skills/")...)
	}

	a.Entries = append(a.Entries, auditSkillFiles(filepath.Join(projectDir, ".codex"), ".codex/")...)

	return a
}

func auditQwen(_ string, homeDir string, env *auditEnv) AgentAudit {
	a := AgentAudit{Name: AgentQwen}

	if homeDir != "" {
		a.Entries = append(a.Entries, auditSkillDirs(filepath.Join(homeDir, ".qwen", "skills"), "~/.qwen/skills/")...)
		cfgPath := filepath.Join(homeDir, ".qwen", "settings.json")
		a.Entries = append(a.Entries, auditMCPEntries(parseMCPDetails(cfgPath, "mcpServers"), "~/.qwen/settings.json", env)...)
	}

	return a
}

func auditOpenClaw(_ string, homeDir string, env *auditEnv) AgentAudit {
	a := AgentAudit{Name: AgentOpenClaw}

	if homeDir != "" {
		a.Entries = append(a.Entries, auditSkillDirs(filepath.Join(homeDir, ".openclaw", "skills"), "~/.openclaw/skills/")...)
		cfgPath := filepath.Join(homeDir, ".openclaw", "openclaw.json")
		a.Entries = append(a.Entries, auditMCPEntries(parseMCPDetails(cfgPath, "mcpServers"), "~/.openclaw/openclaw.json", env)...)
		mcporterPath := filepath.Join(homeDir, ".openclaw", "config", "mcporter.json")
		a.Entries = append(a.Entries, auditMCPEntries(parseMCPDetails(mcporterPath, "mcpServers"), "~/.openclaw/config/mcporter.json", env)...)
	}

	return a
}

// --- Budget audit ---

const (
	budgetWarnPct     = 10.0
	budgetCriticalPct = 20.0
	largeSkillTokens  = 2000
	emptySkillTokens  = 50
)

// auditBudget checks per-agent token budget thresholds.
func auditBudget(scanResult *ScanResult, homeDir, projectDir string) []AuditEntry {
	var entries []AuditEntry
	for _, a := range scanResult.Agents {
		window := ContextWindow(a.Name)
		pct := math.Round(float64(a.Tokens)/float64(window)*1000) / 10

		if pct >= budgetCriticalPct {
			entries = append(entries, AuditEntry{
				Category: "budget",
				Name:     "config-budget-critical",
				Status:   AuditWarn,
				Message:  fmt.Sprintf("%s: %s tokens (%.1f%% of ~%sK window)", a.Name, formatBudgetTokens(a.Tokens), pct, formatK(window)),
				Data: map[string]interface{}{
					"agent":   a.Name,
					"tokens":  a.Tokens,
					"window":  window,
					"percent": pct,
				},
			})
		} else if pct >= budgetWarnPct {
			entries = append(entries, AuditEntry{
				Category: "budget",
				Name:     "config-budget-warning",
				Status:   AuditWarn,
				Message:  fmt.Sprintf("%s: %s tokens (%.1f%% of ~%sK window)", a.Name, formatBudgetTokens(a.Tokens), pct, formatK(window)),
				Data: map[string]interface{}{
					"agent":   a.Name,
					"tokens":  a.Tokens,
					"window":  window,
					"percent": pct,
				},
			})
		}

		// Per-source checks.
		var large []string
		var empty []string
		for _, src := range a.Sources {
			tokens := sourceTokens(expandSourcePath(src, homeDir, projectDir))
			if tokens > largeSkillTokens {
				large = append(large, fmt.Sprintf("%s: %s", src, formatBudgetTokens(tokens)))
			} else if tokens > 0 && tokens < emptySkillTokens {
				empty = append(empty, src)
			}
		}

		if len(large) > 0 {
			entries = append(entries, AuditEntry{
				Category: "budget",
				Name:     "large-skill-warning",
				Status:   AuditWarn,
				Message:  fmt.Sprintf("%s: %d sources over %d tokens (%s)", a.Name, len(large), largeSkillTokens, strings.Join(large, ", ")),
			})
		}
		if len(empty) > 0 {
			entries = append(entries, AuditEntry{
				Category: "budget",
				Name:     "empty-token-skills",
				Status:   AuditWarn,
				Message:  fmt.Sprintf("%s: %d sources under %d tokens (%s)", a.Name, len(empty), emptySkillTokens, strings.Join(empty, ", ")),
			})
		}
	}
	return entries
}

// expandSourcePath converts a display-label source path to a filesystem path.
func expandSourcePath(source, homeDir, projectDir string) string {
	if strings.HasPrefix(source, "~/") {
		return filepath.Join(homeDir, source[2:])
	}
	return filepath.Join(projectDir, source)
}

// sourceTokens computes the token count for a source path (file or directory).
func sourceTokens(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	if info.IsDir() {
		return bytesToTokens(dirBytesRecursive(path))
	}
	return bytesToTokens(info.Size())
}

// formatBudgetTokens formats a token count with comma separators.
func formatBudgetTokens(tokens int64) string {
	s := fmt.Sprintf("%d", tokens)
	n := len(s)
	if n <= 3 {
		return s
	}
	var buf []byte
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, byte(c))
	}
	return string(buf)
}

// formatK formats a token count as "NNK" (e.g., 165000 → "165").
func formatK(tokens int64) string {
	return fmt.Sprintf("%d", tokens/1000)
}
