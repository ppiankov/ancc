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

// EnforcementPosture identifies whether an agent boundary is proven by valid evidence.
// WO-77: scanner posture is a 3-state evidence-backed claim.
type EnforcementPosture string

// Enforcement is the legacy type name for enforcement posture.
type Enforcement = EnforcementPosture

const (
	EnforcementEnforcing  EnforcementPosture = "enforcing"
	EnforcementAdvisory   EnforcementPosture = "advisory"
	EnforcementUnverified EnforcementPosture = "unverified"
)

// EvidenceKind identifies the class of evidence behind an enforcement posture.
// WO-77: self-reports and docs are not valid security-probe evidence.
type EvidenceKind string

const (
	EvidenceVendorDocs       EvidenceKind = "vendor_docs"
	EvidenceAgentSelfReport  EvidenceKind = "agent_self_report"
	EvidenceRealToolResult   EvidenceKind = "real_tool_result"
	EvidenceUnfakeableOutput EvidenceKind = "unfakeable_output"
)

// SecurityProbeSelfReportWarning is shown when posture evidence includes rejected self-report probes.
const SecurityProbeSelfReportWarning = "agent self-reports are not valid evidence for security probes"

var (
	ValidEvidenceStandard = []string{ // WO-77: valid evidence classes for advisory/enforcing posture.
		"real OS result",
		"real tool error",
		"unfakeable payload",
	}
	InvalidEvidenceStandard = []string{ // WO-77: invalid evidence classes for security probes.
		"vendor docs",
		"agent says \"YES\"",
		"model explanation",
	}
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

// EvidenceItem records why an enforcement posture can or cannot be trusted.
// WO-77: posture evidence is structured by kind so invalid proof cannot justify advisory/enforcing.
type EvidenceItem struct {
	Kind EvidenceKind `json:"kind" yaml:"kind"` // WO-77: evidence class controls posture validity
	Note string       `json:"note" yaml:"note"` // WO-77: concise cited probe note
}

// AgentResult holds the scan result for a single agent.
type AgentResult struct {
	Name                string             `json:"name"`
	ConfigDir           string             `json:"config_dir"`
	Skills              int                `json:"skills"`
	SkillFiles          []SkillFile        `json:"skill_files,omitempty"`
	Hooks               int                `json:"hooks"`
	HookConfigs         []HookConfig       `json:"hook_configs,omitempty"`
	MCP                 int                `json:"mcp"`
	MCPServers          []MCPServer        `json:"mcp_servers,omitempty"`
	Tokens              int64              `json:"tokens"`
	Sources             []string           `json:"sources"`
	InvalidLocations    []InvalidLocation  `json:"-" yaml:"-"`                     // WO-72: aggregate into ScanResult without emitting invalid-only agents
	Enforcement         EnforcementPosture `json:"enforcement"`                    // WO-77: evidence-backed enforcement posture
	Evidence            []EvidenceItem     `json:"evidence,omitempty"`             // WO-77: structured posture evidence
	Warning             string             `json:"warning,omitempty"`              // WO-77: evidence-quality caveat for advisory agents
	EnforcementEvidence string             `json:"enforcement_evidence,omitempty"` // WO-77: legacy evidence summary
	Advisory            bool               `json:"advisory"`                       // WO-77: deprecated alias for Enforcement==advisory
}

// NormalizeEnforcement applies the evidence requirement and legacy alias.
func (r *AgentResult) NormalizeEnforcement() {
	r.EnforcementEvidence = strings.TrimSpace(r.EnforcementEvidence)
	r.Warning = strings.TrimSpace(r.Warning)
	for i := range r.Evidence {
		r.Evidence[i].Kind = EvidenceKind(strings.TrimSpace(string(r.Evidence[i].Kind)))
		r.Evidence[i].Note = strings.TrimSpace(r.Evidence[i].Note)
	}

	switch r.Enforcement {
	case EnforcementEnforcing, EnforcementAdvisory:
		if !hasValidEnforcementEvidence(r.Evidence) {
			r.Enforcement = EnforcementUnverified
		}
	case EnforcementUnverified, "":
		r.Enforcement = EnforcementUnverified
		r.EnforcementEvidence = ""
	default:
		r.Enforcement = EnforcementUnverified
		r.EnforcementEvidence = ""
	}
	if r.Enforcement != EnforcementUnverified && r.EnforcementEvidence == "" {
		r.EnforcementEvidence = enforcementEvidenceSummary(r.Evidence)
	}
	if r.Enforcement == EnforcementUnverified {
		// WO-82: unverified posture must stay plain in structured output.
		r.Evidence = nil
		r.Warning = ""
		r.EnforcementEvidence = ""
	}
	r.Advisory = r.Enforcement == EnforcementAdvisory
}

func hasValidEnforcementEvidence(evidence []EvidenceItem) bool {
	for _, item := range evidence {
		if item.Note == "" {
			continue
		}
		switch item.Kind {
		case EvidenceRealToolResult, EvidenceUnfakeableOutput:
			return true
		}
	}
	return false
}

func enforcementEvidenceSummary(evidence []EvidenceItem) string {
	var notes []string
	for _, item := range evidence {
		if item.Note == "" {
			continue
		}
		switch item.Kind {
		case EvidenceRealToolResult, EvidenceUnfakeableOutput:
			notes = append(notes, item.Note)
		}
	}
	return strings.Join(notes, "; ")
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
