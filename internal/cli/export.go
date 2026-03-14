package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ppiankov/ancc/internal/skills"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// exportOutput is the top-level export structure.
type exportOutput struct {
	Agents []exportAgent `json:"agents" yaml:"agents"`
}

// exportAgent holds one agent's configuration summary for export.
type exportAgent struct {
	Name     string   `json:"name" yaml:"name"`
	Skills   int      `json:"skills" yaml:"skills"`
	Hooks    int      `json:"hooks" yaml:"hooks"`
	MCP      int      `json:"mcp" yaml:"mcp"`
	Tokens   int64    `json:"tokens" yaml:"tokens"`
	Sources  []string `json:"sources" yaml:"sources"`
	Advisory bool     `json:"advisory" yaml:"advisory"`
}

func newExportCmd() *cobra.Command {
	var format string
	var agent string

	cmd := &cobra.Command{
		Use:   "export [path]",
		Short: "Export agent configuration summary as JSON or YAML",
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

			out := buildExportOutput(result, agent)

			if agent != "" && len(out.Agents) == 0 {
				return fmt.Errorf("agent %q not found", agent)
			}

			w := cmd.OutOrStdout()
			switch format {
			case "yaml":
				return writeExportYAML(w, out)
			default:
				return writeExportJSON(w, out)
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "json", "output format (json, yaml)")
	cmd.Flags().StringVar(&agent, "agent", "", "export only the named agent")

	return cmd
}

func buildExportOutput(result *skills.ScanResult, agentFilter string) *exportOutput {
	out := &exportOutput{}
	for _, a := range result.Agents {
		if agentFilter != "" && a.Name != agentFilter {
			continue
		}
		out.Agents = append(out.Agents, exportAgent{
			Name:     a.Name,
			Skills:   a.Skills,
			Hooks:    a.Hooks,
			MCP:      a.MCP,
			Tokens:   a.Tokens,
			Sources:  a.Sources,
			Advisory: a.Advisory,
		})
	}
	return out
}

func writeExportJSON(w io.Writer, out *exportOutput) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeExportYAML(w io.Writer, out *exportOutput) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(out)
}
