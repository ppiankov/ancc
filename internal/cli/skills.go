package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ppiankov/ancc/internal/skills"
	"github.com/spf13/cobra"
)

func newSkillsCmd() *cobra.Command {
	var format string
	var showTokens bool
	var budget int

	cmd := &cobra.Command{
		Use:   "skills [path]",
		Short: "Scan for agent configurations in a directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			if budget > 0 {
				showTokens = true
			}

			result, err := skills.Scan(path)
			if err != nil {
				return fmt.Errorf("scan error: %w", err)
			}

			w := cmd.OutOrStdout()
			switch format {
			case "json":
				if err := formatSkillsJSON(w, result, budget); err != nil {
					return fmt.Errorf("formatting output: %w", err)
				}
			default:
				formatSkillsText(w, result, showTokens, budget)
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	cmd.Flags().BoolVar(&showTokens, "tokens", false, "show estimated token counts")
	cmd.Flags().IntVar(&budget, "budget", 0, "context window size in tokens (implies --tokens)")

	return cmd
}

const (
	skillsAgentWidth        = 14
	skillsNumWidth          = 8
	skillsEnforcementWidth  = 10
	skillsAutonomyModeWidth = 32
	skillsSourceKindWidth   = 12
	skillsTokenWidth        = 10
	skillsBudgetWidth       = 8
)

func formatSkillsText(w io.Writer, result *skills.ScanResult, showTokens bool, budget int) {
	if len(result.Agents) == 0 && result.Product == nil && len(result.InvalidLocations) == 0 {
		_, _ = fmt.Fprintln(w, "No agent configurations found.")
		return
	}

	if len(result.Agents) == 0 && result.Product == nil {
		_, _ = fmt.Fprintln(w, "No agent configurations found.")
	}

	if len(result.Agents) > 0 {
		// Header.
		_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %-*s %-*s",
			skillsAgentWidth, "Agent",
			skillsNumWidth, "Skills",
			skillsNumWidth, "Hooks",
			skillsNumWidth, "MCP",
			skillsEnforcementWidth, "Posture",
		)
		if showTokens {
			_, _ = fmt.Fprintf(w, " %-*s", skillsTokenWidth, "Tokens")
		}
		if budget > 0 {
			_, _ = fmt.Fprintf(w, " %-*s", skillsBudgetWidth, "Budget")
		}
		_, _ = fmt.Fprintln(w, " Source")

		// Rows.
		for _, a := range result.Agents {
			sources := strings.Join(a.Sources, ", ")
			_, _ = fmt.Fprintf(w, "  %-*s %-*d %-*d %-*d %-*s",
				skillsAgentWidth, a.Name,
				skillsNumWidth, a.Skills,
				skillsNumWidth, a.Hooks,
				skillsNumWidth, a.MCP,
				skillsEnforcementWidth, agentEnforcement(a),
			)
			if showTokens {
				_, _ = fmt.Fprintf(w, " %-*s", skillsTokenWidth, formatTokenCount(a.Tokens))
			}
			if budget > 0 {
				pct := float64(a.Tokens) / float64(budget) * 100
				_, _ = fmt.Fprintf(w, " %-*s", skillsBudgetWidth, fmt.Sprintf("%.1f%%", pct))
			}
			_, _ = fmt.Fprintf(w, " %s\n", sources)
		}

		writeSkillsPostureDetails(w, result.Agents)
		writeSkillsCompoundCautionDetails(w, result.Agents)
		writeSkillsAutonomyDetails(w, result.Agents)
	}

	if len(result.InvalidLocations) > 0 {
		// WO-72: show rejected candidates separately so counts stay meaningful.
		if len(result.Agents) > 0 {
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintln(w, "  Invalid locations:")
		for _, loc := range result.InvalidLocations {
			_, _ = fmt.Fprintf(w, "  %-*s %s (%s)\n",
				skillsAgentWidth, loc.Agent, loc.Path, loc.Reason)
		}
	}

	if result.Product != nil {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "  ANCC product: %s (name: %s)\n",
			result.Product.Path, result.Product.Name)
	}
}

// WO-78: keep posture rendering evidence-based and non-blocking.
func agentEnforcement(a skills.AgentResult) skills.EnforcementPosture {
	switch a.Enforcement {
	case skills.EnforcementEnforcing, skills.EnforcementAdvisory, skills.EnforcementUnverified:
		return a.Enforcement
	default:
		return skills.EnforcementUnverified
	}
}

func writeSkillsPostureDetails(w io.Writer, agents []skills.AgentResult) {
	wroteHeader := false
	for _, a := range agents {
		if agentEnforcement(a) != skills.EnforcementAdvisory {
			continue
		}
		if !wroteHeader {
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "  Warnings:")
			wroteHeader = true
		}
		writeAdvisoryWarningBlock(w, a.Name, a.Warning)
	}

	wroteHeader = false
	for _, a := range agents {
		if agentEnforcement(a) != skills.EnforcementEnforcing {
			continue
		}
		if enforcementEvidenceCitation(a.EnforcementEvidence, a.Evidence) == "" {
			continue
		}
		if !wroteHeader {
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "  Evidence:")
			wroteHeader = true
		}
		writeEnforcingEvidenceBlock(w, a.Name, a.EnforcementEvidence, a.Evidence)
	}
}

// WO-93: compound caution is a synthesis on top of posture and autonomy blocks.
func writeSkillsCompoundCautionDetails(w io.Writer, agents []skills.AgentResult) {
	wroteHeader := false
	for _, a := range agents {
		caution := a.CompoundRiskCaution()
		if caution == nil {
			continue
		}
		if !wroteHeader {
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "  Compound cautions:")
			wroteHeader = true
		}
		writeCompoundCautionBlock(w, a.Name, caution)
	}
}

// WO-90: keep autonomy as a separate output block from enforcement posture.
func writeSkillsAutonomyDetails(w io.Writer, agents []skills.AgentResult) {
	wroteHeader := false
	for _, a := range agents {
		if len(a.Autonomy) == 0 {
			continue
		}
		if !wroteHeader {
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, "  Autonomy:")
			_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %s\n",
				skillsAgentWidth, "Agent",
				skillsAutonomyModeWidth, "Mode",
				skillsSourceKindWidth, "Source",
				"Capability",
			)
			wroteHeader = true
		}
		for _, capability := range a.Autonomy {
			writeAutonomyCapabilityBlock(w, a.Name, capability)
		}
	}
}

func writeAdvisoryWarningBlock(w io.Writer, name, warning string) {
	displayName := agentDisplayName(name)
	if warning == "" {
		warning = skills.SecurityProbeSelfReportWarning
	}

	_, _ = fmt.Fprintf(w, "  %-*s %s  (warning)\n",
		skillsAgentWidth, name, strings.ToUpper(string(skills.EnforcementAdvisory)))
	if name == skills.AgentAntigravity {
		_, _ = fmt.Fprintf(w, "    %s's workspace trust is not an enforcement boundary.\n", displayName)
		_, _ = fmt.Fprintln(w, "    Live probes show it can read outside the declared workspace when the OS allows it.")
		_, _ = fmt.Fprintln(w, "    Do not rely on agent self-reports as proof of access or enforcement:")
		_, _ = fmt.Fprintln(w, "    a headless YES/NO probe reported success for a TCC-blocked file, but an actual read failed with Operation not permitted.")
	} else {
		_, _ = fmt.Fprintf(w, "    %s has advisory guardrails, not an enforcement boundary.\n", displayName)
		_, _ = fmt.Fprintf(w, "    %s.\n", strings.TrimSuffix(warning, "."))
	}
	_, _ = fmt.Fprintln(w, "    Evidence standard:")
	_, _ = fmt.Fprintf(w, "      valid:   %s\n", strings.Join(skills.ValidEvidenceStandard, ", "))
	_, _ = fmt.Fprintf(w, "      invalid: %s\n", strings.Join(skills.InvalidEvidenceStandard, ", "))
}

// WO-78: enforcing posture is plain, but still cites the probe evidence.
func writeEnforcingEvidenceBlock(w io.Writer, name, evidence string, items []skills.EvidenceItem) {
	citation := enforcementEvidenceCitation(evidence, items)
	if citation == "" {
		return
	}
	_, _ = fmt.Fprintf(w, "  %-*s %s\n",
		skillsAgentWidth, name, strings.ToUpper(string(skills.EnforcementEnforcing)))
	_, _ = fmt.Fprintf(w, "    evidence: %s\n", citation)
}

// WO-90: autonomy is documented capability, not enforcement evidence.
func writeAutonomyCapabilityBlock(w io.Writer, name string, capability skills.AutonomyCapability) {
	_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %s\n",
		skillsAgentWidth, name,
		skillsAutonomyModeWidth, capability.Mode,
		skillsSourceKindWidth, capability.SourceKind,
		capability.Disables,
	)
	if capability.Source != "" {
		_, _ = fmt.Fprintf(w, "    source: %s\n", capability.Source)
	}
}

func writeCompoundCautionBlock(w io.Writer, name string, caution *skills.CompoundCaution) {
	if caution == nil || caution.Message == "" {
		return
	}
	_, _ = fmt.Fprintf(w, "  %-*s %s\n", skillsAgentWidth, name, caution.Message)
}

func enforcementEvidenceCitation(evidence string, items []skills.EvidenceItem) string {
	if citation := strings.TrimSpace(evidence); citation != "" {
		return citation
	}

	var notes []string
	for _, item := range items {
		note := strings.TrimSpace(item.Note)
		if note == "" {
			continue
		}
		switch item.Kind {
		case skills.EvidenceRealToolResult, skills.EvidenceUnfakeableOutput:
			notes = append(notes, note)
		}
	}
	return strings.Join(notes, "; ")
}

func agentDisplayName(name string) string {
	switch name {
	case skills.AgentAntigravity:
		return "Antigravity"
	default:
		return name
	}
}

// formatTokenCount returns a human-readable token count with ~ prefix and comma separators.
func formatTokenCount(tokens int64) string {
	if tokens == 0 {
		return "~0"
	}
	s := fmt.Sprintf("%d", tokens)
	n := len(s)
	if n <= 3 {
		return "~" + s
	}
	var buf []byte
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, byte(c))
	}
	return "~" + string(buf)
}

// agentWithBudget extends AgentResult with budget percentage for JSON output.
type agentWithBudget struct {
	skills.AgentResult
	BudgetPct float64 `json:"budget_pct"`
}

type scanResultWithBudget struct {
	Path             string                   `json:"path"`
	Agents           []agentWithBudget        `json:"agents"`
	InvalidLocations []skills.InvalidLocation `json:"invalid_locations,omitempty"`
	Product          *skills.ANCCProduct      `json:"product,omitempty"`
}

func formatSkillsJSON(w io.Writer, result *skills.ScanResult, budget int) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if budget <= 0 {
		return enc.Encode(result)
	}

	out := scanResultWithBudget{
		Path:             result.Path,
		InvalidLocations: result.InvalidLocations,
		Product:          result.Product,
	}
	for _, a := range result.Agents {
		pct := float64(a.Tokens) / float64(budget) * 100
		out.Agents = append(out.Agents, agentWithBudget{
			AgentResult: a,
			BudgetPct:   pct,
		})
	}
	return enc.Encode(out)
}
