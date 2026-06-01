package skills

import "strings"

// AgentName is the type for agent identifiers.
type AgentName = string

// Agent name constants.
const (
	AgentClaudeCode  = "claude-code"
	AgentCline       = "cline"
	AgentCursor      = "cursor"
	AgentOpenCode    = "opencode"
	AgentCodex       = "codex"
	AgentQwen        = "qwen"
	AgentOpenClaw    = "openclaw"
	AgentWindsurf    = "windsurf"
	AgentAider       = "aider"
	AgentContinue    = "continue"
	AgentCopilot     = "copilot"
	AgentKilocode    = "kilocode"
	AgentVibe        = "vibe"
	AgentGoose       = "goose"
	AgentAntigravity = "antigravity" // WO-66: agy scanner target
)

// DefaultContextWindows maps agent names to default context window sizes in tokens.
var DefaultContextWindows = map[string]int64{
	AgentClaudeCode:  165_000,
	AgentCline:       128_000,
	AgentCursor:      128_000,
	AgentOpenCode:    128_000,
	AgentCodex:       128_000,
	AgentQwen:        128_000,
	AgentOpenClaw:    128_000,
	AgentWindsurf:    128_000,
	AgentAider:       128_000,
	AgentContinue:    128_000,
	AgentCopilot:     128_000,
	AgentKilocode:    128_000,
	AgentVibe:        128_000,
	AgentGoose:       128_000,
	AgentAntigravity: 1_000_000, // WO-66: Gemini 3 Pro long context
}

const defaultContextWindow int64 = 128_000

// Enforcement identifies whether an agent configuration boundary is proven to enforce.
type Enforcement string

const (
	EnforcementEnforcing  Enforcement = "enforcing"
	EnforcementAdvisory   Enforcement = "advisory"
	EnforcementUnverified Enforcement = "unverified"
)

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

// InvalidLocation represents a candidate location that was present but unusable.
type InvalidLocation struct {
	Agent  string `json:"agent" yaml:"agent"`   // WO-72: identify which scanner rejected the location
	Path   string `json:"path" yaml:"path"`     // WO-72: user-visible rejected candidate path
	Reason string `json:"reason" yaml:"reason"` // WO-72: deterministic rejection reason
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
	Name                string            `json:"name"`
	ConfigDir           string            `json:"config_dir"`
	Skills              int               `json:"skills"`
	SkillFiles          []SkillFile       `json:"skill_files,omitempty"`
	Hooks               int               `json:"hooks"`
	HookConfigs         []HookConfig      `json:"hook_configs,omitempty"`
	MCP                 int               `json:"mcp"`
	MCPServers          []MCPServer       `json:"mcp_servers,omitempty"`
	Tokens              int64             `json:"tokens"`
	Sources             []string          `json:"sources"`
	InvalidLocations    []InvalidLocation `json:"-" yaml:"-"`                     // WO-72: aggregate into ScanResult without emitting invalid-only agents
	Enforcement         Enforcement       `json:"enforcement"`                    // WO-77: evidence-backed enforcement posture
	EnforcementEvidence string            `json:"enforcement_evidence,omitempty"` // WO-77: citation required for enforcing/advisory
	Advisory            bool              `json:"advisory"`                       // WO-77: deprecated alias for Enforcement==advisory
}

// NormalizeEnforcement applies the evidence requirement and legacy alias.
func (r *AgentResult) NormalizeEnforcement() {
	r.EnforcementEvidence = strings.TrimSpace(r.EnforcementEvidence)
	switch r.Enforcement {
	case EnforcementEnforcing, EnforcementAdvisory:
		if r.EnforcementEvidence == "" {
			r.Enforcement = EnforcementUnverified
		}
	case EnforcementUnverified, "":
		r.Enforcement = EnforcementUnverified
		r.EnforcementEvidence = ""
	default:
		r.Enforcement = EnforcementUnverified
		r.EnforcementEvidence = ""
	}
	r.Advisory = r.Enforcement == EnforcementAdvisory
}

// ANCCProduct holds ANCC product SKILL.md info if present.
type ANCCProduct struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// ScanResult holds the complete scan output.
type ScanResult struct {
	Path             string            `json:"path"`
	Agents           []AgentResult     `json:"agents"`
	InvalidLocations []InvalidLocation `json:"invalid_locations,omitempty"` // WO-72: rejected candidate locations for users and automation
	Product          *ANCCProduct      `json:"product,omitempty"`
}
