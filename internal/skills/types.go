package skills

// Agent name constants.
const (
	AgentClaudeCode = "claude-code"
	AgentCline      = "cline"
	AgentCursor     = "cursor"
	AgentOpenCode   = "opencode"
	AgentCodex      = "codex"
	AgentQwen       = "qwen"
	AgentOpenClaw   = "openclaw"
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
}

const defaultContextWindow int64 = 128_000

// ContextWindow returns the default context window for the named agent.
func ContextWindow(name string) int64 {
	if w, ok := DefaultContextWindows[name]; ok {
		return w
	}
	return defaultContextWindow
}

// AgentResult holds the scan result for a single agent.
type AgentResult struct {
	Name     string   `json:"name"`
	Skills   int      `json:"skills"`
	Hooks    int      `json:"hooks"`
	MCP      int      `json:"mcp"`
	Tokens   int64    `json:"tokens"`
	Sources  []string `json:"sources"`
	Advisory bool     `json:"advisory"`
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
