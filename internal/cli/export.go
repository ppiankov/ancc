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
	Agents           []exportAgent            `json:"agents" yaml:"agents"`
	InvalidLocations []skills.InvalidLocation `json:"invalid_locations,omitempty" yaml:"invalid_locations,omitempty"` // WO-72: rejected candidate paths
}

// exportAgent holds one agent's configuration summary for export.
type exportAgent struct {
	Name        string              `json:"name" yaml:"name"`
	ConfigDir   string              `json:"config_dir" yaml:"config_dir"`
	Skills      int                 `json:"skills" yaml:"skills"`
	SkillFiles  []skills.SkillFile  `json:"skill_files,omitempty" yaml:"skill_files,omitempty"`
	Hooks       int                 `json:"hooks" yaml:"hooks"`
	HookConfigs []skills.HookConfig `json:"hook_configs,omitempty" yaml:"hook_configs,omitempty"`
	MCP         int                 `json:"mcp" yaml:"mcp"`
	MCPServers  []skills.MCPServer  `json:"mcp_servers,omitempty" yaml:"mcp_servers,omitempty"`
	Tokens      int64               `json:"tokens" yaml:"tokens"`
	Sources     []string            `json:"sources" yaml:"sources"`
	Advisory    bool                `json:"advisory" yaml:"advisory"`
}

func newExportCmd() *cobra.Command {
	var format string
	var agent string

	cmd := &cobra.Command{
		Use:   "export [--format json|yaml] [--agent claude|codex|...] [path]",
		Short: "Export agent configuration summary as JSON or YAML",
		Long: `Export agent configuration summary as JSON or YAML.

By default, exports all detected agents in JSON format.
Use --format yaml for YAML output.
Use --agent <name> to export only a specific agent's configuration.

Output structure:
  {
    "agents": [
      {
        "name": "claude-code",
        "config_dir": "~/.claude",
        "skills": [...],
        "skill_files": [...],
        "hooks": [...],
        "hook_configs": [...],
        "mcp": [...],
        "mcp_servers": [...],
        "tokens": ...,
        "sources": [...],
        "advisory": false
      }
    ],
    "invalid_locations": [
      {
        "agent": "antigravity",
        "path": "./.antigravitycli/skills/draft",
        "reason": "missing required file SKILL.md"
      }
    ]
  }`,
		Args: cobra.MaximumNArgs(1),
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

			if agent != "" && len(out.Agents) == 0 && len(out.InvalidLocations) == 0 {
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
	out := &exportOutput{
		InvalidLocations: filterInvalidLocations(result.InvalidLocations, agentFilter),
	}
	for _, a := range result.Agents {
		if agentFilter != "" && a.Name != agentFilter {
			continue
		}
		out.Agents = append(out.Agents, exportAgent{
			Name:        a.Name,
			ConfigDir:   a.ConfigDir,
			Skills:      a.Skills,
			SkillFiles:  a.SkillFiles,
			Hooks:       a.Hooks,
			HookConfigs: a.HookConfigs,
			MCP:         a.MCP,
			MCPServers:  a.MCPServers,
			Tokens:      a.Tokens,
			Sources:     a.Sources,
			Advisory:    a.Advisory,
		})
	}
	return out
}

// WO-72: keep export invalid-location evidence aligned with an optional agent filter.
func filterInvalidLocations(locations []skills.InvalidLocation, agentFilter string) []skills.InvalidLocation {
	if agentFilter == "" {
		return locations
	}
	var filtered []skills.InvalidLocation
	for _, loc := range locations {
		if loc.Agent == agentFilter {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

func writeExportJSON(w io.Writer, out *exportOutput) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeExportYAML(w io.Writer, out *exportOutput) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(out); err != nil {
		return err
	}
	return enc.Close()
}
