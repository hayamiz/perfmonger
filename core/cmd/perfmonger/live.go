package main

import (
	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/recorder"
)

// liveOptions represents all options for the live command
// It extends recordOptions with color and pretty options for live display
type liveOptions struct {
	*recordOptions
	Color  bool
	Pretty bool
}

// newLiveOptions creates liveOptions with Ruby-compatible defaults
func newLiveOptions() *liveOptions {
	return &liveOptions{
		recordOptions: newRecordOptions(),
		Color:         false,
		Pretty:        false,
	}
}

// parseArgs validates and processes the parsed arguments
func (opts *liveOptions) parseArgs(args []string, cmd *cobra.Command) error {
	// First parse the record options
	if err := opts.recordOptions.parseArgs(args, cmd); err != nil {
		return err
	}
	
	// Live mode always outputs to stdout (will be piped to player)
	opts.recordOptions.Logfile = "-"
	
	return nil
}

// run executes the live command logic
func (opts *liveOptions) run() error {
	// Build the recorder command arguments
	recorderArgs := opts.recordOptions.buildRecorderArgs()
	
	// Find the player binary
	playerBin, err := findPlayerBinary()
	if err != nil {
		return err
	}
	
	// Add player-bin argument for live mode
	recorderArgs = append(recorderArgs, "-player-bin", playerBin)
	
	// Add color option if specified
	if opts.Color {
		recorderArgs = append(recorderArgs, "-color")
	}
	
	// Add pretty option if specified  
	if opts.Pretty {
		recorderArgs = append(recorderArgs, "-pretty")
	}
	
	// Execute the recorder with live mode
	return runRecorderWithArgs(recorderArgs)
}

// runRecorderWithArgs executes the recorder with given arguments
func runRecorderWithArgs(args []string) error {
	recorder.Run(args)
	return nil
}

// findPlayerBinary finds the player binary (using same logic as Ruby CoreFinder)
func findPlayerBinary() (string, error) {
	// In the unified binary approach, we use perfmonger-core with play subcommand
	return "perfmonger-core", nil
}

// newLiveCommand creates the live subcommand with Ruby-compatible options
func newLiveCommand() *cobra.Command {
	opts := newLiveOptions()
	
	cmd := &cobra.Command{
		Use:   "live [options]",
		Short: "Monitor live system performance",
		Long:  `Record and play system performance information in real-time with JSON output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.parseArgs(args, cmd); err != nil {
				return err
			}
			return opts.run()
		},
	}
	
	// Add all record options (disk, interval, etc.) 
	addRecordFlags(cmd, opts.recordOptions)
	
	// Add live-specific options
	cmd.Flags().BoolVarP(&opts.Color, "color", "c", opts.Color,
		"Use colored JSON output")
	cmd.Flags().BoolVar(&opts.Pretty, "pretty", opts.Pretty,
		"Use human readable JSON output")
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	
	return cmd
}

// addRecordFlags adds the common record flags to the command
// This reuses the same flag definitions as the record command
func addRecordFlags(cmd *cobra.Command, opts *recordOptions) {
	cmd.Flags().StringSliceVarP(&opts.Disks, "disk", "d", opts.Disks,
		"Device name to be monitored (e.g. sda, sdb, md0, dm-1)")
	cmd.Flags().Float64VarP(&opts.Interval, "interval", "i", opts.Interval,
		"Amount of time between each measurement report. Floating point is o.k.")
	
	// Note: We don't add -l/--logfile flag for live mode as it always outputs to stdout
	
	// Add other record options that make sense for live mode
	cmd.Flags().BoolVar(&opts.NoCPU, "no-cpu", opts.NoCPU,
		"Do not record CPU usage")
	cmd.Flags().BoolVar(&opts.NoDisk, "no-disk", opts.NoDisk,
		"Do not record disk usage")
	cmd.Flags().BoolVar(&opts.NoNet, "no-net", opts.NoNet,
		"Do not record network usage")
	cmd.Flags().BoolVar(&opts.NoMem, "no-mem", opts.NoMem,
		"Do not record memory usage")
		
	// Timeout can be useful for live mode
	cmd.Flags().Float64VarP(&opts.Timeout, "timeout", "t", opts.Timeout,
		"Timeout in seconds")
}