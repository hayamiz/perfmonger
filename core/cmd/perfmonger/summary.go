package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/summarizer"
)

// summaryOptions represents all options for the summary command
type summaryOptions struct {
	// Basic options
	JSON      bool
	Pager     string
	DiskOnly  string
	
	// Positional argument
	LogFile   string
}

// newSummaryOptions creates summaryOptions with Ruby-compatible defaults
func newSummaryOptions() *summaryOptions {
	return &summaryOptions{
		JSON:     false,
		Pager:    "",
		DiskOnly: "",
		LogFile:  "",
	}
}

// parseArgs validates and processes the parsed arguments
func (opts *summaryOptions) parseArgs(args []string, cmd *cobra.Command) error {
	// Validate that log file is provided
	if len(args) == 0 {
		return fmt.Errorf("PerfMonger log file is required")
	}
	
	// Take the first argument as log file
	opts.LogFile = args[0]
	
	// Check if file exists
	if _, err := os.Stat(opts.LogFile); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", opts.LogFile)
	}
	
	// Handle pager option validation (Ruby-compatible)
	if cmd.Flags().Changed("pager") && opts.Pager == "" {
		// If --pager was specified without a value, use PAGER environment variable
		if pager := os.Getenv("PAGER"); pager != "" {
			opts.Pager = pager
		} else {
			return fmt.Errorf("no pager is available. Please set PAGER or give pager name to --pager option")
		}
	}
	
	return nil
}

// run executes the summary command logic
func (opts *summaryOptions) run() error {
	// Build arguments for the existing summarizer
	args := opts.buildSummarizerArgs()
	
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running summarizer with args: %v\n", args)
	}
	
	// If pager is specified, run summarizer through pager
	if opts.Pager != "" {
		return opts.runWithPager(args)
	}
	
	// Call the existing summarizer.Run function
	summarizer.Run(args)
	return nil
}

// runWithPager runs the summarizer command through a pager
func (opts *summaryOptions) runWithPager(summarizerArgs []string) error {
	// Create summarizer command
	cmd := exec.Command("go", "run", "github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/summarizer")
	cmd.Args = append(cmd.Args, summarizerArgs...)
	
	// Create pager command
	pagerCmd := exec.Command(opts.Pager)
	
	// Pipe summarizer output to pager
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %v", err)
	}
	
	pagerCmd.Stdin = pipe
	pagerCmd.Stdout = os.Stdout
	pagerCmd.Stderr = os.Stderr
	
	// Start pager
	if err := pagerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start pager: %v", err)
	}
	
	// Start summarizer
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start summarizer: %v", err)
	}
	
	// Wait for summarizer to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("summarizer failed: %v", err)
	}
	
	// Close pipe and wait for pager
	pipe.Close()
	if err := pagerCmd.Wait(); err != nil {
		return fmt.Errorf("pager failed: %v", err)
	}
	
	return nil
}

// buildSummarizerArgs creates arguments for the existing summarizer
func (opts *summaryOptions) buildSummarizerArgs() []string {
	var args []string
	
	// Add options
	if opts.JSON {
		args = append(args, "-json")
	}
	
	if opts.DiskOnly != "" {
		args = append(args, "-disk-only", opts.DiskOnly)
	}
	
	// Add title based on log file name
	args = append(args, "-title", opts.LogFile)
	
	// Add log file as the last argument
	args = append(args, opts.LogFile)
	
	return args
}

// newSummaryCommand creates the summary subcommand with Ruby-compatible options
func newSummaryCommand() *cobra.Command {
	opts := newSummaryOptions()
	
	cmd := &cobra.Command{
		Use:   "summary [options] LOG_FILE",
		Short: "Summarize system performance data",
		Long:  `Show a summary of a perfmonger log file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.parseArgs(args, cmd); err != nil {
				return err
			}
			return opts.run()
		},
	}
	
	// Ruby-compatible flags
	cmd.Flags().BoolVar(&opts.JSON, "json", opts.JSON,
		"Output summary in JSON")
	cmd.Flags().StringVarP(&opts.Pager, "pager", "p", opts.Pager,
		"Use pager to see summary output.")
	cmd.Flags().StringVar(&opts.DiskOnly, "disk-only", opts.DiskOnly,
		"Select disk devices that matches REGEX (Ex. 'sd[b-d]')")
	
	// Add aliases
	cmd.Aliases = []string{"summarize"}
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}