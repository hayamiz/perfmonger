package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/player"
)

// playOptions represents all options for the play command
type playOptions struct {
	// Basic options
	Color     bool
	Pretty    bool
	DiskOnly  string
	
	// Positional argument
	LogFile   string
}

// newPlayOptions creates playOptions with Ruby-compatible defaults
func newPlayOptions() *playOptions {
	return &playOptions{
		Color:    false,
		Pretty:   false,
		DiskOnly: "",
		LogFile:  "",
	}
}

// parseArgs validates and processes the parsed arguments
func (opts *playOptions) parseArgs(args []string, cmd *cobra.Command) error {
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
	
	return nil
}

// run executes the play command logic
func (opts *playOptions) run() error {
	// Build arguments for the existing player
	args := opts.buildPlayerArgs()
	
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running player with args: %v\n", args)
	}
	
	// Call the existing player.Run function
	player.Run(args)
	return nil
}

// buildPlayerArgs creates arguments for the existing player
func (opts *playOptions) buildPlayerArgs() []string {
	var args []string
	
	// Add options
	if opts.Color {
		args = append(args, "-color")
	}
	
	if opts.Pretty {
		args = append(args, "-pretty")
	}
	
	if opts.DiskOnly != "" {
		args = append(args, "-disk-only", opts.DiskOnly)
	}
	
	// Add log file as the last argument
	args = append(args, opts.LogFile)
	
	return args
}

// newPlayCommand creates the play subcommand with Ruby-compatible options
func newPlayCommand() *cobra.Command {
	opts := newPlayOptions()
	
	cmd := &cobra.Command{
		Use:   "play [options] LOG_FILE",
		Short: "Play a recorded perfmonger session",
		Long:  `Play a perfmonger log file in JSON`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.parseArgs(args, cmd); err != nil {
				return err
			}
			return opts.run()
		},
	}
	
	// Ruby-compatible flags with both short and long forms
	cmd.Flags().BoolVarP(&opts.Color, "color", "c", opts.Color,
		"Use colored JSON output")
	cmd.Flags().BoolVarP(&opts.Pretty, "pretty", "p", opts.Pretty,
		"Use human readable JSON output")
	cmd.Flags().StringVar(&opts.DiskOnly, "disk-only", opts.DiskOnly,
		"Select disk devices that matches REGEX (Ex. 'sd[b-d]')")
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}