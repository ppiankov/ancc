package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/ppiankov/ancc/internal/skills"
	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	var format string
	var agentFilter string
	var windowOverride int

	cmd := &cobra.Command{
		Use:   "context [path]",
		Short: "Show per-agent token budget breakdown",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			result, err := skills.Scan(path)
			if err != nil {
				return fmt.Errorf("scan error: %w", err)
			}

			agents := buildContextAgents(result, agentFilter, int64(windowOverride))

			w := cmd.OutOrStdout()
			switch format {
			case "json":
				if err := formatContextJSON(w, agents); err != nil {
					return fmt.Errorf("formatting output: %w", err)
				}
			default:
				formatContextText(w, agents)
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	cmd.Flags().StringVar(&agentFilter, "agent", "", "show only this agent")
	cmd.Flags().IntVar(&windowOverride, "window", 0, "override context window size in tokens")

	return cmd
}

type contextAgent struct {
	Name            string  `json:"name"`
	ConfigTokens    int64   `json:"config_tokens"`
	ContextWindow   int64   `json:"context_window"`
	AvailableTokens int64   `json:"available_tokens"`
	ConfigPercent   float64 `json:"config_percent"`
	Skills          int     `json:"skills"`
	Hooks           int     `json:"hooks"`
	MCP             int     `json:"mcp"`
}

func buildContextAgents(result *skills.ScanResult, agentFilter string, windowOverride int64) []contextAgent {
	var agents []contextAgent
	for _, a := range result.Agents {
		if agentFilter != "" && a.Name != agentFilter {
			continue
		}

		window := skills.ContextWindow(a.Name)
		if windowOverride > 0 {
			window = windowOverride
		}

		available := window - a.Tokens
		if available < 0 {
			available = 0
		}

		var pct float64
		if window > 0 {
			pct = math.Round(float64(a.Tokens)/float64(window)*1000) / 10
		}

		agents = append(agents, contextAgent{
			Name:            a.Name,
			ConfigTokens:    a.Tokens,
			ContextWindow:   window,
			AvailableTokens: available,
			ConfigPercent:   pct,
			Skills:          a.Skills,
			Hooks:           a.Hooks,
			MCP:             a.MCP,
		})
	}
	return agents
}

const (
	ctxAgentWidth  = 14
	ctxTokenWidth  = 11
	ctxWindowWidth = 12
	ctxAvailWidth  = 12
	ctxPctWidth    = 10
)

func formatContextText(w io.Writer, agents []contextAgent) {
	if len(agents) == 0 {
		_, _ = fmt.Fprintln(w, "No agent configurations found.")
		return
	}

	_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %-*s %-*s\n",
		ctxAgentWidth, "Agent",
		ctxTokenWidth, "Config",
		ctxWindowWidth, "Window",
		ctxAvailWidth, "Available",
		ctxPctWidth, "Config %",
	)

	for _, a := range agents {
		_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %-*s %-*s\n",
			ctxAgentWidth, a.Name,
			ctxTokenWidth, formatTokenCount(a.ConfigTokens),
			ctxWindowWidth, formatTokenCount(a.ContextWindow),
			ctxAvailWidth, formatTokenCount(a.AvailableTokens),
			ctxPctWidth, fmt.Sprintf("%.1f%%", a.ConfigPercent),
		)
	}
}

type contextResult struct {
	Agents []contextAgent `json:"agents"`
}

func formatContextJSON(w io.Writer, agents []contextAgent) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(contextResult{Agents: agents})
}
