package cli

import (
	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ancc",
		Short:         "Static validator for the Agent-Native CLI Convention",
		Version:       version,
		SilenceErrors: true,
	}

	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newSkillsCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newAuditCmd())
	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newContextCmd())
	cmd.AddCommand(newDiffCmd())

	return cmd
}

// Execute runs the root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
