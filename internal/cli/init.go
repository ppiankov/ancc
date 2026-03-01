package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var name string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a template SKILL.md with all required sections",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("getting working directory: %w", err)
				}
				name = filepath.Base(cwd)
			}

			outPath := filepath.Join("docs", "SKILL.md")

			if !force {
				if _, err := os.Stat(outPath); err == nil {
					return &ExitError{Code: 1}
				}
			}

			if err := os.MkdirAll("docs", 0o755); err != nil {
				return fmt.Errorf("creating docs directory: %w", err)
			}

			content := generateTemplate(name)
			if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
				return &ExitError{Code: 1}
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", outPath)
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&name, "name", "", "tool name (default: directory name)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing SKILL.md")

	return cmd
}

func generateTemplate(name string) string {
	var b strings.Builder

	b.WriteString("# " + name + "\n\n")
	b.WriteString("One-line description of " + name + ".\n\n")

	b.WriteString("## Install\n\n")
	b.WriteString("```\nbrew install " + name + "\n```\n\n")
	b.WriteString("Or via Go:\n\n")
	b.WriteString("```\ngo install github.com/yourorg/" + name + "/cmd/" + name + "@latest\n```\n\n")

	b.WriteString("## Commands\n\n")

	// Primary command with all required subsections.
	b.WriteString("### " + name + " run\n\n")
	b.WriteString("Describe what this command does.\n\n")
	b.WriteString("**Flags:**\n")
	b.WriteString("- `--format json` — output as JSON (default: human-readable)\n\n")
	b.WriteString("**JSON output:**\n")
	b.WriteString("```json\n")
	b.WriteString("{\n")
	b.WriteString("  \"status\": \"ok\",\n")
	b.WriteString("  \"data\": []\n")
	b.WriteString("}\n")
	b.WriteString("```\n\n")
	b.WriteString("**Exit codes:**\n")
	b.WriteString("- 0: success\n")
	b.WriteString("- 1: failure\n\n")

	// Init command.
	b.WriteString("### " + name + " init\n\n")
	b.WriteString("Initialize configuration.\n\n")

	// Doctor command.
	b.WriteString("### " + name + " doctor\n\n")
	b.WriteString("Check environment health.\n\n")

	b.WriteString("## What this does NOT do\n\n")
	b.WriteString("- Does not do X\n")
	b.WriteString("- Does not do Y\n\n")

	b.WriteString("## Parsing examples\n\n")
	b.WriteString("```bash\n")
	b.WriteString(name + " run --format json | jq '.status'\n")
	b.WriteString("```\n")

	return b.String()
}
