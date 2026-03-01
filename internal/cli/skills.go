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

	cmd := &cobra.Command{
		Use:   "skills [path]",
		Short: "Scan for agent configurations in a directory",
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

			w := cmd.OutOrStdout()
			switch format {
			case "json":
				if err := formatSkillsJSON(w, result); err != nil {
					return fmt.Errorf("formatting output: %w", err)
				}
			default:
				formatSkillsText(w, result)
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")

	return cmd
}

const (
	skillsAgentWidth  = 14
	skillsNumWidth    = 8
	skillsSourceWidth = 50
)

func formatSkillsText(w io.Writer, result *skills.ScanResult) {
	if len(result.Agents) == 0 && result.Product == nil {
		_, _ = fmt.Fprintln(w, "No agent configurations found.")
		return
	}

	if len(result.Agents) > 0 {
		_, _ = fmt.Fprintf(w, "  %-*s %-*s %-*s %-*s %s\n",
			skillsAgentWidth, "Agent",
			skillsNumWidth, "Skills",
			skillsNumWidth, "Hooks",
			skillsNumWidth, "MCP",
			"Source",
		)

		for _, a := range result.Agents {
			sources := strings.Join(a.Sources, ", ")
			_, _ = fmt.Fprintf(w, "  %-*s %-*d %-*d %-*d %s\n",
				skillsAgentWidth, a.Name,
				skillsNumWidth, a.Skills,
				skillsNumWidth, a.Hooks,
				skillsNumWidth, a.MCP,
				sources,
			)
		}
	}

	if result.Product != nil {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "  ANCC product: %s (name: %s)\n",
			result.Product.Path, result.Product.Name)
	}
}

func formatSkillsJSON(w io.Writer, result *skills.ScanResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
