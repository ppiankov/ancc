package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var format string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "validate [path]",
		Short: "Validate a repo against the ANCC convention",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			// TODO: wire validator (WO-003, WO-004)
			_, _ = format, verbose
			fmt.Fprintf(cmd.OutOrStdout(), "validating %s (not yet implemented)\n", path)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show all checks including passing")

	return cmd
}
