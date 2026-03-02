package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ppiankov/ancc/internal/skills"
	"github.com/spf13/cobra"
)

const (
	diffAgentWidth = 14
	diffCountWidth = 10
	diffTokenWidth = 16
)

func newDiffCmd() *cobra.Command {
	var format string
	var agentFilter string
	var showTokens bool

	cmd := &cobra.Command{
		Use:   "diff <path-a> <path-b>",
		Short: "Compare agent configurations between two directories",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resultA, err := skills.Scan(args[0])
			if err != nil {
				return fmt.Errorf("scanning %s: %w", args[0], err)
			}
			resultB, err := skills.Scan(args[1])
			if err != nil {
				return fmt.Errorf("scanning %s: %w", args[1], err)
			}

			diff := skills.DiffConfigs(resultA, resultB)

			if agentFilter != "" {
				diff = filterDiffAgent(diff, agentFilter)
			}

			w := cmd.OutOrStdout()
			switch format {
			case "json":
				if err := formatDiffJSON(w, diff); err != nil {
					return fmt.Errorf("formatting output: %w", err)
				}
			default:
				formatDiffText(w, diff, showTokens)
			}

			if diff.Summary.Added > 0 || diff.Summary.Removed > 0 || diff.Summary.Changed > 0 {
				return &ExitError{Code: 1}
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	cmd.Flags().StringVar(&agentFilter, "agent", "", "compare only this agent")
	cmd.Flags().BoolVar(&showTokens, "tokens", false, "show token counts")

	return cmd
}

func filterDiffAgent(result *skills.DiffResult, name string) *skills.DiffResult {
	filtered := &skills.DiffResult{
		PathA: result.PathA,
		PathB: result.PathB,
	}
	for _, d := range result.Agents {
		if d.Name == name {
			filtered.Agents = append(filtered.Agents, d)
		}
	}
	for _, d := range filtered.Agents {
		filtered.Summary.Total++
		switch d.Status {
		case skills.DiffAdded:
			filtered.Summary.Added++
		case skills.DiffRemoved:
			filtered.Summary.Removed++
		case skills.DiffChanged:
			filtered.Summary.Changed++
		case skills.DiffIdentical:
			filtered.Summary.Identical++
		}
	}
	return filtered
}

func formatDiffText(w io.Writer, result *skills.DiffResult, showTokens bool) {
	if len(result.Agents) == 0 {
		_, _ = fmt.Fprintln(w, "No agent configurations found in either directory.")
		return
	}

	_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %-*s",
		diffAgentWidth, "Agent",
		diffCountWidth, "Skills",
		diffCountWidth, "Hooks",
		diffCountWidth, "MCP",
	)
	if showTokens {
		_, _ = fmt.Fprintf(w, " %-*s", diffTokenWidth, "Tokens")
	}
	_, _ = fmt.Fprintln(w, " Status")

	for _, d := range result.Agents {
		if d.Status == skills.DiffIdentical {
			_, _ = fmt.Fprintf(w, "  %-*s %-*d %-*d %-*d",
				diffAgentWidth, d.Name,
				diffCountWidth, d.Skills.A,
				diffCountWidth, d.Hooks.A,
				diffCountWidth, d.MCP.A,
			)
			if showTokens {
				_, _ = fmt.Fprintf(w, " %-*s", diffTokenWidth, formatTokenCount(d.Tokens.A))
			}
			_, _ = fmt.Fprintln(w, " identical")
			continue
		}

		_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %-*s",
			diffAgentWidth, d.Name,
			diffCountWidth, formatCountDiff(d.Skills),
			diffCountWidth, formatCountDiff(d.Hooks),
			diffCountWidth, formatCountDiff(d.MCP),
		)
		if showTokens {
			_, _ = fmt.Fprintf(w, " %-*s", diffTokenWidth, formatTokenDiffStr(d.Tokens))
		}
		_, _ = fmt.Fprintf(w, " %s\n", d.Status)

		for _, s := range d.Added {
			_, _ = fmt.Fprintf(w, "    + %s\n", s)
		}
		for _, s := range d.Removed {
			_, _ = fmt.Fprintf(w, "    - %s\n", s)
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  Summary: %d agents compared", result.Summary.Total)
	if result.Summary.Added > 0 {
		_, _ = fmt.Fprintf(w, ", %d added", result.Summary.Added)
	}
	if result.Summary.Removed > 0 {
		_, _ = fmt.Fprintf(w, ", %d removed", result.Summary.Removed)
	}
	if result.Summary.Changed > 0 {
		_, _ = fmt.Fprintf(w, ", %d changed", result.Summary.Changed)
	}
	if result.Summary.Identical > 0 {
		_, _ = fmt.Fprintf(w, ", %d identical", result.Summary.Identical)
	}
	_, _ = fmt.Fprintln(w)
}

func formatDiffJSON(w io.Writer, result *skills.DiffResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func formatCountDiff(c skills.CountDiff) string {
	return fmt.Sprintf("%d -> %d", c.A, c.B)
}

func formatTokenDiffStr(t skills.TokenDiff) string {
	return fmt.Sprintf("%s -> %s", formatTokenCount(t.A), formatTokenCount(t.B))
}
