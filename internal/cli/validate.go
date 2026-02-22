package cli

import (
	"fmt"

	"github.com/ppiankov/ancc/internal/validator"
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

			result, err := validator.Validate(path)
			if err != nil {
				return fmt.Errorf("validation error: %w", err)
			}

			w := cmd.OutOrStdout()
			switch format {
			case "json":
				if err := formatJSON(w, result); err != nil {
					return fmt.Errorf("formatting output: %w", err)
				}
			default:
				formatText(w, result, verbose)
			}

			switch result.Status {
			case validator.OverallFail:
				return &ExitError{Code: 1}
			case validator.OverallPartial:
				return &ExitError{Code: 2}
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show all checks including passing")

	return cmd
}
