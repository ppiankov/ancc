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
	skillsAgentWidth  = 14
	skillsNumWidth    = 8
	skillsTokenWidth  = 10
	skillsBudgetWidth = 8
)

func formatSkillsText(w io.Writer, result *skills.ScanResult, showTokens bool, budget int) {
	if len(result.Agents) == 0 && result.Product == nil {
		_, _ = fmt.Fprintln(w, "No agent configurations found.")
		return
	}

	if len(result.Agents) > 0 {
		// Header.
		_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %-*s",
			skillsAgentWidth, "Agent",
			skillsNumWidth, "Skills",
			skillsNumWidth, "Hooks",
			skillsNumWidth, "MCP",
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
			_, _ = fmt.Fprintf(w, "  %-*s %-*d %-*d %-*d",
				skillsAgentWidth, a.Name,
				skillsNumWidth, a.Skills,
				skillsNumWidth, a.Hooks,
				skillsNumWidth, a.MCP,
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
	}

	if result.Product != nil {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "  ANCC product: %s (name: %s)\n",
			result.Product.Path, result.Product.Name)
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
	Path    string              `json:"path"`
	Agents  []agentWithBudget   `json:"agents"`
	Product *skills.ANCCProduct `json:"product,omitempty"`
}

func formatSkillsJSON(w io.Writer, result *skills.ScanResult, budget int) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if budget <= 0 {
		return enc.Encode(result)
	}

	out := scanResultWithBudget{
		Path:    result.Path,
		Product: result.Product,
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
