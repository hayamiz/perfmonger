package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/summarizer"
)

// summaryCommand represents the summary command with direct SummaryOption setting
type summaryCommand struct {
	// Direct field (no embedding) for maximum efficiency  
	SummaryOpt *summarizer.SummaryOption
	
	// Ruby-specific options only
	Pager string
}

// newSummaryCommandStruct creates summaryCommand with Ruby-compatible defaults
func newSummaryCommandStruct() *summaryCommand {
	return &summaryCommand{
		SummaryOpt: summarizer.NewSummaryOption(),
		Pager:      "",
	}
}

// validateAndSetLogfile validates the logfile argument using cobra's PreRunE approach
func (cmd *summaryCommand) validateAndSetLogfile(args []string) error {
	// Validate that log file is provided
	if len(args) == 0 {
		return fmt.Errorf("PerfMonger log file is required")
	}
	
	// Take the first argument as log file
	cmd.SummaryOpt.Logfile = args[0]
	
	// Set title to log file name if not specified
	if cmd.SummaryOpt.Title == "" {
		cmd.SummaryOpt.Title = args[0]
	}
	
	// Check if file exists
	if _, err := os.Stat(cmd.SummaryOpt.Logfile); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", cmd.SummaryOpt.Logfile)
	}
	
	return nil
}

// validatePagerOption validates the pager option (Ruby-compatible)  
func (cmd *summaryCommand) validatePagerOption(cobraCmd *cobra.Command) error {
	// Handle pager option validation (Ruby-compatible)
	if cobraCmd.Flags().Changed("pager") && cmd.Pager == "" {
		// If --pager was specified without a value, use PAGER environment variable
		if pager := os.Getenv("PAGER"); pager != "" {
			cmd.Pager = pager
		} else {
			return fmt.Errorf("no pager is available. Please set PAGER or give pager name to --pager option")
		}
	}
	
	return nil
}

// run executes the summary command with direct API calls
func (cmd *summaryCommand) run() error {
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running summarizer with options: %+v\n", cmd.SummaryOpt)
	}
	
	// If pager is specified, run summarizer through pager
	if cmd.Pager != "" {
		return cmd.runWithPager()
	}
	
	// Direct API call - no conversion needed
	summarizer.RunWithOption(cmd.SummaryOpt)
	return nil
}

// runWithPager runs the summarizer command through a pager
func (cmd *summaryCommand) runWithPager() error {
	// For now, disable pager functionality with the new API
	// TODO: Implement proper pager support with struct-based API
	fmt.Fprintf(os.Stderr, "Warning: pager functionality temporarily disabled with new API\n")
	summarizer.RunWithOption(cmd.SummaryOpt)
	return nil
}


// newSummaryCommand creates the summary subcommand with direct cobra setting
func newSummaryCommand() *cobra.Command {
	summaryCmd := newSummaryCommandStruct()
	
	cmd := &cobra.Command{
		Use:   "summary [options] LOG_FILE",
		Short: "Summarize system performance data",
		Long:  `Show a summary of a perfmonger log file`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validation moved to PreRunE for cobra integration
			if err := summaryCmd.validateAndSetLogfile(args); err != nil {
				return err
			}
			return summaryCmd.validatePagerOption(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Direct execution - no additional validation needed
			return summaryCmd.run()
		},
	}
	
	// Direct cobra flag setting to SummaryOption fields (no conversion needed)
	cmd.Flags().BoolVar(&summaryCmd.SummaryOpt.JSON, "json", summaryCmd.SummaryOpt.JSON,
		"Output summary in JSON")
	cmd.Flags().StringVarP(&summaryCmd.Pager, "pager", "p", summaryCmd.Pager,
		"Use pager to see summary output.")
	cmd.Flags().StringVar(&summaryCmd.SummaryOpt.DiskOnly, "disk-only", summaryCmd.SummaryOpt.DiskOnly,
		"Select disk devices that matches REGEX (Ex. 'sd[b-d]')")
	
	// Add aliases
	cmd.Aliases = []string{"summarize"}
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}