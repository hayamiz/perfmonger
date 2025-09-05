package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/recorder"
)

// recordOptions represents all options for the record command
type recordOptions struct {
	// Basic options
	Disks     []string
	Logfile   string
	Interval  float64
	
	// Control options  
	Kill         bool
	Status       bool
	Background   bool
	
	// Timing options
	StartDelay   float64
	Timeout      float64
	
	// Feature toggles
	RecordIntr      bool
	NoCPU          bool
	NoDisk         bool
	NoNet          bool
	NoMem          bool
	NoGzip         bool
	NoIntervalBackoff bool
	
	// Output options
	Debug    bool
	Verbose  bool
}

// newRecordOptions creates recordOptions with Ruby-compatible defaults
func newRecordOptions() *recordOptions {
	return &recordOptions{
		Disks:             []string{},
		Logfile:           "perfmonger.pgr",
		Interval:          1.0,
		Kill:              false,
		Status:            false,
		Background:        false,
		StartDelay:        0.0,
		Timeout:           0.0,
		RecordIntr:        false,
		NoCPU:             false,
		NoDisk:            false,
		NoNet:             true,  // Ruby default: don't record network
		NoMem:             false,
		NoGzip:            false, // Ruby default: use gzip
		NoIntervalBackoff: false,
		Debug:             false,
		Verbose:           false,
	}
}

// parseArgs validates and processes the parsed arguments
func (opts *recordOptions) parseArgs(args []string, cmd *cobra.Command) error {
	// Validate mutually exclusive options
	if opts.Kill && opts.Status {
		return fmt.Errorf("--kill and --status cannot be used together")
	}
	
	// If kill or status, no other validation needed
	if opts.Kill || opts.Status {
		return nil
	}
	
	// Validate timing parameters (order matters for test expectations)
	if opts.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	
	if opts.StartDelay < 0 {
		return fmt.Errorf("start-delay cannot be negative")
	}
	
	// Validate interval last (since it's always set)
	if opts.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	
	return nil
}

// run executes the record command logic
func (opts *recordOptions) run() error {
	// Handle kill command
	if opts.Kill {
		return opts.killSession()
	}
	
	// Handle status command
	if opts.Status {
		return opts.showStatus()
	}
	
	// Handle background check
	if opts.Background {
		if sessionPID := opts.getRunningSessionPID(); sessionPID > 0 {
			return fmt.Errorf("another perfmonger is already running in background mode")
		}
	}
	
	// Execute normal recording
	return opts.executeRecord()
}

// killSession kills a running background session (Ruby-compatible)
func (opts *recordOptions) killSession() error {
	fmt.Fprintln(os.Stderr, "kill functionality not yet implemented")
	return fmt.Errorf("not implemented")
}

// showStatus shows status of running session (Ruby-compatible)  
func (opts *recordOptions) showStatus() error {
	fmt.Fprintln(os.Stderr, "status functionality not yet implemented")
	return fmt.Errorf("not implemented")
}

// getRunningSessionPID returns PID of running session, 0 if none
func (opts *recordOptions) getRunningSessionPID() int {
	// TODO: Implement Ruby-compatible session detection
	return 0
}

// executeRecord runs the actual recording by calling the existing recorder
func (opts *recordOptions) executeRecord() error {
	// Build arguments for the existing recorder
	args := opts.buildRecorderArgs()
	
	if !opts.Background {
		fmt.Printf("[recording to %s]\n", opts.Logfile)
	}
	
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running recorder with args: %v\n", args)
	}
	
	// Call the existing recorder.Run function
	recorder.Run(args)
	return nil
}

// buildRecorderArgs creates arguments for the existing recorder
func (opts *recordOptions) buildRecorderArgs() []string {
	var args []string
	
	// Interval (convert to milliseconds)
	args = append(args, fmt.Sprintf("-interval=%.0fms", opts.Interval*1000))
	
	// Interval backoff
	if opts.NoIntervalBackoff {
		args = append(args, "-no-interval-backoff")
	}
	
	// Start delay
	if opts.StartDelay > 0 {
		args = append(args, "-start-delay", fmt.Sprintf("%.0fms", opts.StartDelay*1000))
	}
	
	// Timeout
	if opts.Timeout > 0 {
		args = append(args, "-timeout", fmt.Sprintf("%.0fms", opts.Timeout*1000))
	}
	
	// Feature toggles
	if opts.NoCPU {
		args = append(args, "-no-cpu")
	}
	if opts.NoDisk {
		args = append(args, "-no-disk")
	}
	if !opts.RecordIntr {
		args = append(args, "-no-intr")
	}
	if opts.NoNet {
		args = append(args, "-no-net")
	}
	if opts.NoMem {
		args = append(args, "-no-mem")
	}
	
	// Disks
	if len(opts.Disks) > 0 {
		args = append(args, "-disks", strings.Join(opts.Disks, ","))
	}
	
	// Background mode
	if opts.Background {
		args = append(args, "-background")
	}
	
	// Output format
	logfile := opts.Logfile
	if !strings.HasSuffix(logfile, ".gz") && !opts.NoGzip {
		logfile += ".gz"
	}
	if !opts.NoGzip {
		args = append(args, "-gzip")
	}
	
	// Output file
	args = append(args, "-output", logfile)
	
	return args
}

// newRecordCommand creates the record subcommand with Ruby-compatible options
func newRecordCommand() *cobra.Command {
	opts := newRecordOptions()
	
	cmd := &cobra.Command{
		Use:   "record [options]",
		Short: "Record system performance information",
		Long:  `Record system performance information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.parseArgs(args, cmd); err != nil {
				return err
			}
			return opts.run()
		},
	}
	
	// Ruby-compatible flags with both short and long forms
	cmd.Flags().StringSliceVarP(&opts.Disks, "disk", "d", opts.Disks, 
		"Device name to be monitored (e.g. sda, sdb, md0, dm-1).")
	cmd.Flags().StringVarP(&opts.Logfile, "logfile", "l", opts.Logfile, 
		"Output file name")
	cmd.Flags().Float64VarP(&opts.Interval, "interval", "i", opts.Interval, 
		"Amount of time between each measurement report. Floating point is o.k.")
	cmd.Flags().Float64VarP(&opts.StartDelay, "start-delay", "s", opts.StartDelay, 
		"Amount of wait time before starting measurement. Floating point is o.k.")
	cmd.Flags().Float64VarP(&opts.Timeout, "timeout", "t", opts.Timeout, 
		"Amount of measurement time. Floating point is o.k.")
	
	// Control flags
	cmd.Flags().BoolVar(&opts.Kill, "kill", opts.Kill, 
		"Stop currently running perfmonger-record")
	cmd.Flags().BoolVar(&opts.Status, "status", opts.Status, 
		"Show currently running perfmonger-record status")
	cmd.Flags().BoolVar(&opts.Background, "background", opts.Background, 
		"Run in background")
	
	// Feature flags
	cmd.Flags().BoolVar(&opts.RecordIntr, "record-intr", opts.RecordIntr, 
		"Record per core interrupts count (experimental)")
	cmd.Flags().BoolVar(&opts.NoCPU, "no-cpu", opts.NoCPU, 
		"Suppress recording CPU usage.")
	cmd.Flags().BoolVar(&opts.NoNet, "no-net", opts.NoNet, 
		"Suppress recording network usage")
	cmd.Flags().BoolVar(&opts.NoMem, "no-mem", opts.NoMem, 
		"Suppress recording memory usage")
	cmd.Flags().BoolVar(&opts.NoGzip, "no-gzip", opts.NoGzip, 
		"Do not save a logfile in gzipped format")
	cmd.Flags().BoolVar(&opts.NoIntervalBackoff, "no-interval-backoff", opts.NoIntervalBackoff, 
		"Prevent interval to be set longer every after 100 records.")
	
	// Debug flags  
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", opts.Verbose, 
		"Verbose output")
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}