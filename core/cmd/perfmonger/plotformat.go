package main

import (
	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/plotformatter"
)

// newPlotFormatCommand creates a hidden plot-format subcommand that wraps the
// plotformatter package. This is used internally by the plot command so that
// the unified perfmonger binary can replace the old perfmonger-core binary.
func newPlotFormatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "plot-format [flags]",
		Short:  "Format recorded data for gnuplot (internal)",
		Hidden: true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			plotformatter.Run(args)
			return nil
		},
	}
	return cmd
}
