package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/summarizer"
)

// statOptions represents all options for the stat command (inherits from record)
type statOptions struct {
	// Record options (similar to recordOptions)
	Disks     []string
	Logfile   string
	Interval  float64
	StartDelay float64
	Timeout   float64
	NoCPU     bool
	NoDisk    bool
	NoNet     bool
	NoIntr    bool
	NoMem     bool
	NoGzip    bool
	NoIntervalBackoff bool
	Debug     bool
	
	// Stat-specific options
	JSON      bool
	
	// Command to run
	Command   []string
}

// newStatOptions creates statOptions with Ruby-compatible defaults
func newStatOptions() *statOptions {
	return &statOptions{
		Disks:             []string{},
		Logfile:           "./perfmonger.pgr",
		Interval:          1.0,
		StartDelay:        0.0,
		Timeout:           0.0,
		NoCPU:             false,
		NoDisk:            false,
		NoNet:             true,  // Ruby default
		NoIntr:            true,  // Ruby default: don't record interrupts by default
		NoMem:             false,
		NoGzip:            false, // Ruby default: use gzip
		NoIntervalBackoff: false,
		Debug:             false,
		JSON:              false,
		Command:           []string{},
	}
}

// parseArgs validates and processes the parsed arguments
func (opts *statOptions) parseArgs(args []string, cmd *cobra.Command) error {
	// Find the -- separator
	dashIndex := -1
	for i, arg := range args {
		if arg == "--" {
			dashIndex = i
			break
		}
	}
	
	// Extract command after --
	if dashIndex == -1 {
		// If no --, all args are the command
		opts.Command = args
	} else {
		// Command is everything after --
		if dashIndex+1 < len(args) {
			opts.Command = args[dashIndex+1:]
		}
	}
	
	// Validate that command is provided
	if len(opts.Command) == 0 {
		return fmt.Errorf("no command given")
	}
	
	// Validate timing parameters
	if opts.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	
	if opts.StartDelay < 0 {
		return fmt.Errorf("start-delay cannot be negative")
	}
	
	if opts.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	
	return nil
}

// run executes the stat command logic
func (opts *statOptions) run() error {
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running stat command with: %v\n", opts.Command)
	}
	
	// Build recorder arguments (similar to record command)
	recorderArgs := opts.buildRecorderArgs()
	
	// Start recorder in background
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	
	// Use same binary with record subcommand
	recorderCmd := exec.Command(execPath, append([]string{"record"}, recorderArgs...)...)
	recorderCmd.Stdout = nil
	recorderCmd.Stderr = os.Stderr
	
	if err := recorderCmd.Start(); err != nil {
		return fmt.Errorf("failed to start recorder: %v", err)
	}
	
	// Set up signal handling to kill recorder on interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Run the user's command
	startTime := time.Now()
	userCmd := exec.Command(opts.Command[0], opts.Command[1:]...)
	userCmd.Stdin = os.Stdin
	userCmd.Stdout = os.Stdout
	userCmd.Stderr = os.Stderr
	
	cmdErr := userCmd.Run()
	endTime := time.Now()
	
	// Stop recorder
	if recorderCmd.Process != nil {
		recorderCmd.Process.Signal(os.Interrupt)
		recorderCmd.Wait()
	}
	
	// Handle any error from the user command (but continue to show summary)
	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %v\n", cmdErr)
	}
	
	// Show summary
	fmt.Fprintf(os.Stderr, "\nElapsed time: %.3f seconds\n\n", endTime.Sub(startTime).Seconds())
	
	// Run summarizer
	summaryArgs := opts.buildSummaryArgs()
	summarizer.Run(summaryArgs)
	
	return nil
}

// buildRecorderArgs creates arguments for the recorder (similar to record command)
func (opts *statOptions) buildRecorderArgs() []string {
	var args []string
	
	// Interval (convert to milliseconds for internal recorder)
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
	if opts.NoNet {
		args = append(args, "-no-net")
	}
	if opts.NoIntr {
		args = append(args, "-no-intr")
	}
	if opts.NoMem {
		args = append(args, "-no-mem")
	}
	
	// Disks
	if len(opts.Disks) > 0 {
		args = append(args, "-disks", fmt.Sprintf("%v", opts.Disks))
	}
	
	// Output format
	logfile := opts.Logfile
	if !opts.NoGzip {
		args = append(args, "-gzip")
	}
	
	// Output file
	args = append(args, "-output", logfile)
	
	return args
}

// buildSummaryArgs creates arguments for the summarizer
func (opts *statOptions) buildSummaryArgs() []string {
	var args []string
	
	if opts.JSON {
		args = append(args, "-json")
	}
	
	// Add title with the command
	commandStr := ""
	for i, arg := range opts.Command {
		if i > 0 {
			commandStr += " "
		}
		commandStr += arg
	}
	args = append(args, "-title", commandStr)
	args = append(args, opts.Logfile)
	
	return args
}

// newStatCommand creates the stat subcommand with Ruby-compatible options
func newStatCommand() *cobra.Command {
	opts := newStatOptions()
	
	cmd := &cobra.Command{
		Use:   "stat [options] -- <command>",
		Short: "Run a command and show performance summary",
		Long:  `Run a command and gather performance information during its execution.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.parseArgs(args, cmd); err != nil {
				return err
			}
			return opts.run()
		},
	}
	
	// Ruby-compatible flags (similar to record but with stat-specific additions)
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
	
	// Feature flags
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
	
	// Stat-specific flags
	cmd.Flags().BoolVar(&opts.JSON, "json", opts.JSON,
		"Output summary in JSON")
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}