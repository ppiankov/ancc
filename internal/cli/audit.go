package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ppiankov/ancc/internal/skills"
	"github.com/spf13/cobra"
)

func newAuditCmd() *cobra.Command {
	var format string
	var agentFilter string

	cmd := &cobra.Command{
		Use:   "audit [path]",
		Short: "Deep inspection of agent configurations",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			result, err := skills.Audit(path)
			if err != nil {
				return fmt.Errorf("audit error: %w", err)
			}

			if agentFilter != "" {
				result = filterAuditAgent(result, agentFilter)
				result.Environment = nil // skip environment checks for single-agent filter
			}

			w := cmd.OutOrStdout()
			switch format {
			case "json":
				if err := formatAuditJSON(w, result); err != nil {
					return fmt.Errorf("formatting output: %w", err)
				}
			default:
				formatAuditText(w, result)
			}

			if result.Summary.Errors > 0 {
				return &ExitError{Code: 1}
			}
			if result.Summary.Warn > 0 {
				return &ExitError{Code: 2}
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	cmd.Flags().StringVar(&agentFilter, "agent", "", "audit only this agent")

	return cmd
}

func filterAuditAgent(result *skills.AuditResult, name string) *skills.AuditResult {
	filtered := &skills.AuditResult{Path: result.Path}
	for _, a := range result.Agents {
		if a.Name == name {
			filtered.Agents = append(filtered.Agents, a)
		}
	}
	// Recompute summary for filtered set.
	for _, a := range filtered.Agents {
		for _, e := range a.Entries {
			filtered.Summary.Total++
			switch e.Status {
			case skills.AuditOK:
				filtered.Summary.OK++
			case skills.AuditWarn:
				filtered.Summary.Warn++
			case skills.AuditError:
				filtered.Summary.Errors++
			}
		}
	}
	return filtered
}

var auditStatusIcons = map[skills.AuditStatus]string{
	skills.AuditOK:    "ok",
	skills.AuditWarn:  "WARN",
	skills.AuditError: "FAIL",
}

func formatAuditText(w io.Writer, result *skills.AuditResult) {
	if len(result.Agents) == 0 && len(result.Environment) == 0 {
		_, _ = fmt.Fprintln(w, "No agent configurations found to audit.")
		return
	}

	for i, agent := range result.Agents {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintf(w, "  %s\n", agent.Name)

		// Group entries by category.
		categories := groupByCategory(agent.Entries)
		for _, cat := range []string{"hook", "mcp", "skill"} {
			entries, ok := categories[cat]
			if !ok {
				continue
			}
			_, _ = fmt.Fprintf(w, "    %ss:\n", cat)
			for _, e := range entries {
				status := auditStatusIcons[e.Status]
				_, _ = fmt.Fprintf(w, "      [%s] %s: %s\n", status, e.Name, e.Message)
			}
		}
	}

	envWarns := formatEnvironmentSection(w, result)

	_, _ = fmt.Fprintln(w)
	issues := result.Summary.Errors + result.Summary.Warn
	if issues == 0 {
		_, _ = fmt.Fprintln(w, "  Result: all checks passed")
	} else {
		_, _ = fmt.Fprintf(w, "  Result: %d issues found (%d errors, %d warnings)\n",
			issues, result.Summary.Errors, result.Summary.Warn)
	}

	if envWarns > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "  Recommendations:")
		_, _ = fmt.Fprintln(w, "    - Revoke Full Disk Access for your terminal app (System Settings > Privacy & Security)")
		_, _ = fmt.Fprintln(w, "    - Use a sandboxed terminal profile for AI agent sessions")
		_, _ = fmt.Fprintln(w, "    - Restrict agent file access with tool-level permissions (hooks, allowlists)")
	}
}

func formatEnvironmentSection(w io.Writer, result *skills.AuditResult) int {
	if len(result.Environment) == 0 {
		return 0
	}

	if len(result.Agents) > 0 {
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, "  environment")

	categories := groupByCategory(result.Environment)
	envWarns := 0
	for _, cat := range []string{"sensitive-dir", "credential-dir"} {
		entries, ok := categories[cat]
		if !ok {
			continue
		}
		_, _ = fmt.Fprintf(w, "    %ss:\n", cat)
		for _, e := range entries {
			status := auditStatusIcons[e.Status]
			_, _ = fmt.Fprintf(w, "      [%s] %s: %s\n", status, e.Name, e.Message)
			if e.Status == skills.AuditWarn {
				envWarns++
			}
		}
	}
	return envWarns
}

func groupByCategory(entries []skills.AuditEntry) map[string][]skills.AuditEntry {
	groups := make(map[string][]skills.AuditEntry)
	for _, e := range entries {
		groups[e.Category] = append(groups[e.Category], e)
	}
	return groups
}

func formatAuditJSON(w io.Writer, result *skills.AuditResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
