package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/recorder"
)

// secondsDurationValue is a custom flag value that accepts float64 seconds (Ruby-compatible)
type secondsDurationValue struct {
	target *time.Duration
}

func (s *secondsDurationValue) String() string {
	if s.target == nil {
		return "0s"
	}
	return s.target.String()
}

func (s *secondsDurationValue) Set(value string) error {
	// Try parsing as float64 seconds first (Ruby compatibility)
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		*s.target = time.Duration(seconds * float64(time.Second))
		return nil
	}
	
	// Fall back to duration parsing
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid duration format: %v", err)
	}
	*s.target = duration
	return nil
}

func (s *secondsDurationValue) Type() string {
	return "duration"
}

// recordCommand represents the record command with direct RecorderOption setting
type recordCommand struct {
	// Direct field (no embedding) for maximum efficiency
	RecorderOpt *recorder.RecorderOption
	
	// Ruby-specific options only
	Kill       bool
	Status     bool
	RecordIntr bool
	NoGzip     bool
	Verbose    bool
}

// newRecordCommandStruct creates recordCommand with Ruby-compatible defaults
func newRecordCommandStruct() *recordCommand {
	opt := recorder.NewRecorderOption()
	// Ruby defaults differ from recorder defaults
	opt.Output = "perfmonger.pgr"  // Ruby default logfile name
	opt.NoNet = true               // Ruby default: don't record network
	opt.Gzip = true                // Ruby default: use gzip
	
	return &recordCommand{
		RecorderOpt: opt,
		Kill:        false,
		Status:      false,
		RecordIntr:  false,
		NoGzip:      false,
		Verbose:     false,
	}
}

// validateOptions performs validation using cobra's PreRunE approach
func (cmd *recordCommand) validateOptions() error {
	// Validate mutually exclusive options
	if cmd.Kill && cmd.Status {
		return fmt.Errorf("--kill and --status cannot be used together")
	}
	
	// If kill or status, no other validation needed
	if cmd.Kill || cmd.Status {
		return nil
	}
	
	// Validate timing parameters (order matters for test expectations)
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

// run executes the record command logic with direct API calls
func (cmd *recordCommand) run() error {
	// Handle kill command
	if cmd.Kill {
		return cmd.killSession()
	}
	
	// Handle status command
	if cmd.Status {
		return cmd.showStatus()
	}
	
	// Handle background check
	if cmd.RecorderOpt.Background {
		if sessionPID := cmd.getRunningSessionPID(); sessionPID > 0 {
			return fmt.Errorf("another perfmonger is already running in background mode")
		}
	}
	
	// Execute normal recording with minimal processing
	return cmd.executeRecord()
}

// killSession kills a running background session (Ruby-compatible)
func (cmd *recordCommand) killSession() error {
	fmt.Fprintln(os.Stderr, "kill functionality not yet implemented")
	return fmt.Errorf("not implemented")
}

// showStatus shows status of running session (Ruby-compatible)  
func (cmd *recordCommand) showStatus() error {
	fmt.Fprintln(os.Stderr, "status functionality not yet implemented")
	return fmt.Errorf("not implemented")
}

// getRunningSessionPID returns PID of running session, 0 if none
func (cmd *recordCommand) getRunningSessionPID() int {
	// TODO: Implement Ruby-compatible session detection
	return 0
}

// executeRecord runs the actual recording using direct API with minimal processing
func (cmd *recordCommand) executeRecord() error {
	// Apply Ruby-specific logic (minimal processing only)
	cmd.applyRubySpecificLogic()
	
	if !cmd.RecorderOpt.Background {
		fmt.Printf("[recording to %s]\n", cmd.RecorderOpt.Output)
	}
	
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running recorder with options: %+v\n", cmd.RecorderOpt)
	}
	
	// Direct API call - no conversion needed
	recorder.RunWithOption(cmd.RecorderOpt)
	return nil
}

// applyRubySpecificLogic applies minimal Ruby-specific logic (replaces convertCobraToRecorderOptions)
func (cmd *recordCommand) applyRubySpecificLogic() {
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


// newRecordCommand creates the record subcommand with direct cobra setting
func newRecordCommand() *cobra.Command {
	recCmd := newRecordCommandStruct()
	
	cmd := &cobra.Command{
		Use:   "record [options]",
		Short: "Record system performance information",
		Long:  `Record system performance information`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validation moved to PreRunE for cobra integration
			return recCmd.validateOptions()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Direct execution - no parseArgs needed
			return recCmd.run()
		},
	}
	
	// Direct cobra flag setting to RecorderOption fields (no conversion needed)
	cmd.Flags().StringSliceVarP(&recCmd.RecorderOpt.DevsParts, "disk", "d", recCmd.RecorderOpt.DevsParts, 
		"Device name to be monitored (e.g. sda, sdb, md0, dm-1).")
	cmd.Flags().StringVarP(&recCmd.RecorderOpt.Output, "logfile", "l", recCmd.RecorderOpt.Output, 
		"Output file name")
	
	// Ruby-compatible duration setting (accepts both float64 seconds and duration format)
	cmd.Flags().VarP(&secondsDurationValue{target: &recCmd.RecorderOpt.Interval}, "interval", "i", 
		"Amount of time between each measurement report. Floating point is o.k.")
	cmd.Flags().VarP(&secondsDurationValue{target: &recCmd.RecorderOpt.StartDelay}, "start-delay", "s", 
		"Amount of wait time before starting measurement. Floating point is o.k.")
	cmd.Flags().VarP(&secondsDurationValue{target: &recCmd.RecorderOpt.Timeout}, "timeout", "t", 
		"Amount of measurement time. Floating point is o.k.")
	
	// Control flags (Ruby-specific)
	cmd.Flags().BoolVar(&recCmd.Kill, "kill", recCmd.Kill, 
		"Stop currently running perfmonger-record")
	cmd.Flags().BoolVar(&recCmd.Status, "status", recCmd.Status, 
		"Show currently running perfmonger-record status")
	cmd.Flags().BoolVar(&recCmd.RecorderOpt.Background, "background", recCmd.RecorderOpt.Background, 
		"Run in background")
	
	// Feature flags (direct setting to RecorderOption)
	cmd.Flags().BoolVar(&recCmd.RecordIntr, "record-intr", recCmd.RecordIntr, 
		"Record per core interrupts count (experimental)")
	cmd.Flags().BoolVar(&recCmd.RecorderOpt.NoCPU, "no-cpu", recCmd.RecorderOpt.NoCPU, 
		"Suppress recording CPU usage.")
	cmd.Flags().BoolVar(&recCmd.RecorderOpt.NoNet, "no-net", recCmd.RecorderOpt.NoNet, 
		"Suppress recording network usage")
	cmd.Flags().BoolVar(&recCmd.RecorderOpt.NoMem, "no-mem", recCmd.RecorderOpt.NoMem, 
		"Suppress recording memory usage")
	cmd.Flags().BoolVar(&recCmd.NoGzip, "no-gzip", recCmd.NoGzip, 
		"Do not save a logfile in gzipped format")
	cmd.Flags().BoolVar(&recCmd.RecorderOpt.NoIntervalBackoff, "no-interval-backoff", recCmd.RecorderOpt.NoIntervalBackoff, 
		"Prevent interval to be set longer every after 100 records.")
	
	// Debug flags  
	cmd.Flags().BoolVarP(&recCmd.Verbose, "verbose", "v", recCmd.Verbose, 
		"Verbose output")
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}