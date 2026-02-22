package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var name string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a template SKILL.md with all required sections",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: implement SKILL.md template generation (WO-004)
			_, _ = name, force
			fmt.Fprintln(cmd.OutOrStdout(), "init not yet implemented")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "tool name (default: directory name)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing SKILL.md")

	return cmd
}
