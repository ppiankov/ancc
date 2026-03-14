package skills

// AgentName is the type for agent identifiers.
type AgentName = string

// Agent name constants.
const (
	AgentClaudeCode = "claude-code"
	AgentCline      = "cline"
	AgentCursor     = "cursor"
	AgentOpenCode   = "opencode"
	AgentCodex      = "codex"
	AgentQwen       = "qwen"
	AgentOpenClaw   = "openclaw"
	AgentWindsurf   = "windsurf"
	AgentAider      = "aider"
	AgentContinue   = "continue"
	AgentCopilot    = "copilot"
	AgentKilocode   = "kilocode"
)

// DefaultContextWindows maps agent names to default context window sizes in tokens.
var DefaultContextWindows = map[string]int64{
	AgentClaudeCode: 165_000,
	AgentCline:      128_000,
	AgentCursor:     128_000,
	AgentOpenCode:   128_000,
	AgentCodex:      128_000,
	AgentQwen:       128_000,
	AgentOpenClaw:   128_000,
	AgentWindsurf:   128_000,
	AgentAider:      128_000,
	AgentContinue:   128_000,
	AgentCopilot:    128_000,
	AgentKilocode:   128_000,
}

const defaultContextWindow int64 = 128_000

// ContextWindow returns the default context window for the named agent.
func ContextWindow(name string) int64 {
	if w, ok := DefaultContextWindows[name]; ok {
		return w
	}
	return defaultContextWindow
}

// SkillFile represents a single skill file or directory.
type SkillFile struct {
	Path string `json:"path" yaml:"path"`
	Type string `json:"type" yaml:"type"` // "file" or "dir"
}

// HookConfig represents a hook configuration.
type HookConfig struct {
	Event string `json:"event" yaml:"event"`
	Path  string `json:"path,omitempty" yaml:"path,omitempty"`
}

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	Name   string `json:"name" yaml:"name"`
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
	Config string `json:"config,omitempty" yaml:"config,omitempty"`
}

// AgentResult holds the scan result for a single agent.
type AgentResult struct {
	Name        string       `json:"name"`
	ConfigDir   string       `json:"config_dir"`
	Skills      int          `json:"skills"`
	SkillFiles  []SkillFile  `json:"skill_files,omitempty"`
	Hooks       int          `json:"hooks"`
	HookConfigs []HookConfig `json:"hook_configs,omitempty"`
	MCP         int          `json:"mcp"`
	MCPServers  []MCPServer  `json:"mcp_servers,omitempty"`
	Tokens      int64        `json:"tokens"`
	Sources     []string     `json:"sources"`
	Advisory    bool         `json:"advisory"`
}

// ANCCProduct holds ANCC product SKILL.md info if present.
type ANCCProduct struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// ScanResult holds the complete scan output.
type ScanResult struct {
	Path    string        `json:"path"`
	Agents  []AgentResult `json:"agents"`
	Product *ANCCProduct  `json:"product,omitempty"`
}
