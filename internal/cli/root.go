package cli

import (
	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ancc",
		Short:   "Static validator for the Agent-Native CLI Convention",
		Version: version,
	}

	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newInitCmd())

	return cmd
}

// Execute runs the root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
