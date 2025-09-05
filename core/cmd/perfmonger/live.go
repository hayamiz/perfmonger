package main

import (
	"fmt"
	"os"
	"strings"
	
	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/recorder"
)

// liveCommand represents the live command with direct RecorderOption setting
type liveCommand struct {
	// Direct field (no embedding) for maximum efficiency
	RecorderOpt *recorder.RecorderOption
	
	// Ruby-specific options only (inherited from record command)
	Kill       bool
	Status     bool
	RecordIntr bool
	NoGzip     bool
	Verbose    bool
	
	// Live-specific options
	Color  bool
	Pretty bool
}

// newLiveCommandStruct creates liveCommand with Ruby-compatible defaults
func newLiveCommandStruct() *liveCommand {
	opt := recorder.NewRecorderOption()
	// Ruby defaults differ from recorder defaults
	opt.Output = "perfmonger.pgr"  // Ruby default logfile name
	opt.NoNet = true               // Ruby default: don't record network
	opt.Gzip = true                // Ruby default: use gzip
	
	return &liveCommand{
		RecorderOpt: opt,
		Kill:        false,
		Status:      false,
		RecordIntr:  false,
		NoGzip:      false,
		Verbose:     false,
		Color:       false,
		Pretty:        false,
	}
}

// validateOptions performs validation using cobra's PreRunE approach
func (cmd *liveCommand) validateOptions() error {
	// Validate mutually exclusive options (inherited from record)
	if cmd.Kill && cmd.Status {
		return fmt.Errorf("--kill and --status cannot be used together")
	}
	
	// If kill or status, no other validation needed
	if cmd.Kill || cmd.Status {
		return nil
	}
	
	// Validate timing parameters (inherited from record)
	if cmd.RecorderOpt.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	
	if cmd.RecorderOpt.StartDelay < 0 {
		return fmt.Errorf("start-delay cannot be negative")
	}
	
	// Validate interval last (since it's always set)
	if cmd.RecorderOpt.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	
	return nil
}

// run executes the live command logic with direct API calls
func (cmd *liveCommand) run() error {
	// Handle kill/status commands (inherited from record)
	if cmd.Kill {
		return cmd.killSession()
	}
	
	if cmd.Status {
		return cmd.showStatus()
	}
	
	// Apply Ruby-specific logic (inherited from record)
	cmd.applyRubySpecificLogic()
	
	// Live mode always outputs to stdout (will be piped to player)
	cmd.RecorderOpt.Output = "-"
	
	// Find the player binary for live mode
	playerBin, err := findPlayerBinary()
	if err != nil {
		return err
	}
	
	// Set player binary for live mode
	cmd.RecorderOpt.PlayerBin = playerBin
	
	// Set color and pretty options for live display
	cmd.RecorderOpt.Color = cmd.Color
	cmd.RecorderOpt.Pretty = cmd.Pretty
	
	// Direct API call - no conversion needed
	recorder.RunWithOption(cmd.RecorderOpt)
	return nil
}

// killSession kills a running background session (Ruby-compatible, inherited from record)
func (cmd *liveCommand) killSession() error {
	fmt.Fprintln(os.Stderr, "kill functionality not yet implemented")
	return fmt.Errorf("not implemented")
}

// showStatus shows status of running session (Ruby-compatible, inherited from record)
func (cmd *liveCommand) showStatus() error {
	fmt.Fprintln(os.Stderr, "status functionality not yet implemented") 
	return fmt.Errorf("not implemented")
}

// applyRubySpecificLogic applies minimal Ruby-specific logic (inherited from record)
func (cmd *liveCommand) applyRubySpecificLogic() {
	// Convert DevsParts slice to comma-separated Disks string (only if needed)
	if len(cmd.RecorderOpt.DevsParts) > 0 {
		cmd.RecorderOpt.Disks = strings.Join(cmd.RecorderOpt.DevsParts, ",")
	}
	
	// Handle Ruby-specific logic (minimal processing)
	if cmd.NoGzip {
		cmd.RecorderOpt.Gzip = false
	}
	
	// Handle record interrupts (Ruby --record-intr vs Go --no-intr)
	if !cmd.RecordIntr {
		cmd.RecorderOpt.NoIntr = true
	}
}

// findPlayerBinary finds the player binary (using same logic as Ruby CoreFinder)
func findPlayerBinary() (string, error) {
	// In the unified binary approach, we use perfmonger-core with play subcommand
	return "perfmonger-core", nil
}

// newLiveCommand creates the live subcommand with direct cobra setting
func newLiveCommand() *cobra.Command {
	liveCmd := newLiveCommandStruct()
	
	cmd := &cobra.Command{
		Use:   "live [options]",
		Short: "Monitor live system performance",
		Long:  `Record and play system performance information in real-time with JSON output.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validation moved to PreRunE for cobra integration
			return liveCmd.validateOptions()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Direct execution - no additional validation needed
			return liveCmd.run()
		},
	}
	
	// Direct cobra flag setting to RecorderOption fields (no conversion needed) - same as record
	cmd.Flags().StringSliceVarP(&liveCmd.RecorderOpt.DevsParts, "disk", "d", liveCmd.RecorderOpt.DevsParts, 
		"Device name to be monitored (e.g. sda, sdb, md0, dm-1).")
	
	// Ruby-compatible duration setting (accepts both float64 seconds and duration format)
	cmd.Flags().VarP(&secondsDurationValue{target: &liveCmd.RecorderOpt.Interval}, "interval", "i", 
		"Amount of time between each measurement report. Floating point is o.k.")
	cmd.Flags().VarP(&secondsDurationValue{target: &liveCmd.RecorderOpt.StartDelay}, "start-delay", "s", 
		"Amount of wait time before starting measurement. Floating point is o.k.")
	cmd.Flags().VarP(&secondsDurationValue{target: &liveCmd.RecorderOpt.Timeout}, "timeout", "t", 
		"Amount of measurement time. Floating point is o.k.")
	
	// Control flags (Ruby-specific)
	cmd.Flags().BoolVar(&liveCmd.Kill, "kill", liveCmd.Kill, 
		"Stop currently running perfmonger-record")
	cmd.Flags().BoolVar(&liveCmd.Status, "status", liveCmd.Status, 
		"Show currently running perfmonger-record status")
	
	// Feature flags (direct setting to RecorderOption) - same as record
	cmd.Flags().BoolVar(&liveCmd.RecordIntr, "record-intr", liveCmd.RecordIntr, 
		"Record per core interrupts count (experimental)")
	cmd.Flags().BoolVar(&liveCmd.RecorderOpt.NoCPU, "no-cpu", liveCmd.RecorderOpt.NoCPU, 
		"Suppress recording CPU usage.")
	cmd.Flags().BoolVar(&liveCmd.RecorderOpt.NoNet, "no-net", liveCmd.RecorderOpt.NoNet, 
		"Suppress recording network usage")
	cmd.Flags().BoolVar(&liveCmd.RecorderOpt.NoMem, "no-mem", liveCmd.RecorderOpt.NoMem, 
		"Suppress recording memory usage")
	cmd.Flags().BoolVar(&liveCmd.NoGzip, "no-gzip", liveCmd.NoGzip, 
		"Do not save a logfile in gzipped format")
		
	// Live-specific options
	cmd.Flags().BoolVarP(&liveCmd.Color, "color", "c", liveCmd.Color,
		"Use colored JSON output")
	cmd.Flags().BoolVar(&liveCmd.Pretty, "pretty", liveCmd.Pretty,
		"Use human readable JSON output")
	
	// Debug flags  
	cmd.Flags().BoolVarP(&liveCmd.Verbose, "verbose", "v", liveCmd.Verbose, 
		"Verbose output")
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}

