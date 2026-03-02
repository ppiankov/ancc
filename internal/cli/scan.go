package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ppiankov/ancc/internal/validator"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	var format string
	var depth int

	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Batch validate all repos in a directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			result, err := validator.ScanRepos(path, depth)
			if err != nil {
				return fmt.Errorf("scan error: %w", err)
			}

			w := cmd.OutOrStdout()
			switch format {
			case "json":
				if err := formatScanJSON(w, result); err != nil {
					return fmt.Errorf("formatting output: %w", err)
				}
			default:
				formatScanText(w, result)
			}

			if result.Summary.Fail > 0 {
				return &ExitError{Code: 1}
			}
			if result.Summary.Partial > 0 {
				return &ExitError{Code: 2}
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	cmd.Flags().IntVar(&depth, "depth", 2, "maximum directory depth to search")

	return cmd
}

func formatScanText(w io.Writer, result *validator.ScanResult) {
	if len(result.Repos) == 0 {
		_, _ = fmt.Fprintln(w, "No repos found.")
		return
	}

	_, _ = fmt.Fprintf(w, "  %-20s %-10s %5s %5s %5s\n", "Repo", "Status", "Pass", "Fail", "Warn")

	for _, repo := range result.Repos {
		if repo.Status == validator.ScanStatusMissing {
			_, _ = fmt.Fprintf(w, "  %-20s %-10s %5s %5s %5s\n",
				repo.Name, repo.Status, "-", "-", "-")
		} else if repo.Summary != nil {
			_, _ = fmt.Fprintf(w, "  %-20s %-10s %5d %5d %5d\n",
				repo.Name, repo.Status, repo.Summary.Pass, repo.Summary.Fail, repo.Summary.Warn)
		}
	}

	_, _ = fmt.Fprintln(w)
	s := result.Summary
	_, _ = fmt.Fprintf(w, "  Summary: %d repos", s.Total)
	if s.Pass > 0 {
		_, _ = fmt.Fprintf(w, ", %d pass", s.Pass)
	}
	if s.Fail > 0 {
		_, _ = fmt.Fprintf(w, ", %d fail", s.Fail)
	}
	if s.Partial > 0 {
		_, _ = fmt.Fprintf(w, ", %d partial", s.Partial)
	}
	if s.Missing > 0 {
		_, _ = fmt.Fprintf(w, ", %d missing", s.Missing)
	}
	_, _ = fmt.Fprintln(w)
}

func formatScanJSON(w io.Writer, result *validator.ScanResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
