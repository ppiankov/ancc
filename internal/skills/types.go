package skills

// Agent name constants.
const (
	AgentClaudeCode = "claude-code"
	AgentCline      = "cline"
	AgentCursor     = "cursor"
	AgentOpenCode   = "opencode"
	AgentCodex      = "codex"
	AgentQwen       = "qwen"
)

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
