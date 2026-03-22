package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/recorder"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/summarizer"
)

// statCommand represents the stat command with direct option setting
type statCommand struct {
	// Direct fields (no embedding) for maximum efficiency
	RecorderOpt  *recorder.RecorderOption
	SummaryOpt   *summarizer.SummaryOption
	
	// Ruby-specific options only (inherited from record command)
	Kill       bool
	Status     bool
	RecordIntr bool
	NoGzip     bool
	Verbose    bool
	
	// Stat-specific options
	Command []string
}

// newStatCommandStruct creates statCommand with Ruby-compatible defaults
func newStatCommandStruct() *statCommand {
	recOpt := recorder.NewRecorderOption()
	sumOpt := summarizer.NewSummaryOption()
	
	// Ruby defaults for stat command
	recOpt.Output = "./perfmonger.pgr"  // stat saves to file by default
	recOpt.NoNet = true                 // Ruby default: don't record network
	recOpt.NoIntr = true                // Ruby default: don't record interrupts
	recOpt.Gzip = true                  // Ruby default: use gzip
	
	return &statCommand{
		RecorderOpt: recOpt,
		SummaryOpt:  sumOpt,
		Kill:        false,
		Status:      false,
		RecordIntr:  false,
		NoGzip:      false,
		Verbose:     false,
		Command:     []string{},
	}
}

// validateAndSetCommand validates the command arguments using cobra's PreRunE approach
func (cmd *statCommand) validateAndSetCommand(args []string) error {
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
		cmd.Command = args
	} else {
		// Command is everything after --
		if dashIndex+1 < len(args) {
			cmd.Command = args[dashIndex+1:]
		}
	}
	
	// Validate that command is provided
	if len(cmd.Command) == 0 {
		return fmt.Errorf("no command given")
	}
	
	return nil
}

// validateOptions performs validation using cobra's PreRunE approach
func (cmd *statCommand) validateOptions() error {
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

// applyStatSpecificLogic applies minimal Ruby-specific logic for stat command
func (cmd *statCommand) applyStatSpecificLogic() {
	// Convert DevsParts slice to comma-separated Disks string (only if needed)
	if len(cmd.RecorderOpt.DevsParts) > 0 {
		cmd.RecorderOpt.Disks = strings.Join(cmd.RecorderOpt.DevsParts, ",")
	}
	
	// Handle Ruby-specific logic (inherited from record)
	if cmd.NoGzip {
		cmd.RecorderOpt.Gzip = false
	}
	
	// Handle record interrupts (Ruby --record-intr vs Go --no-intr)
	if !cmd.RecordIntr {
		cmd.RecorderOpt.NoIntr = true
	}
	
	// Set summary options
	cmd.SummaryOpt.Logfile = cmd.RecorderOpt.Output
	cmd.SummaryOpt.Title = fmt.Sprintf("Command: %s", strings.Join(cmd.Command, " "))
}

// run executes the stat command with direct API calls
func (cmd *statCommand) run() error {
	// Handle kill/status commands (inherited from record)
	if cmd.Kill {
		return cmd.killSession()
	}

	if cmd.Status {
		return cmd.showStatus()
	}

	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running stat command with: %v\n", cmd.Command)
	}

	// Apply Ruby-specific logic for both recorder and summary
	cmd.applyStatSpecificLogic()

	// Set up a stop channel so we can signal the recorder to stop
	stopCh := make(chan struct{})
	cmd.RecorderOpt.StopCh = stopCh

	// Create a temporary goroutine to run the recorder in background
	recorderDone := make(chan bool, 1)

	go func() {
		defer func() { recorderDone <- true }()
		// Start recording using the direct API
		recorder.RunWithOption(cmd.RecorderOpt)
	}()

	// Give recorder a moment to start
	time.Sleep(100 * time.Millisecond)

	// Run the user's command
	userCmd := exec.Command(cmd.Command[0], cmd.Command[1:]...)
	userCmd.Stdin = os.Stdin
	userCmd.Stdout = os.Stdout
	userCmd.Stderr = os.Stderr

	cmdErr := userCmd.Run()

	// Signal recorder to stop by closing the stop channel
	close(stopCh)

	// Wait for recorder to finish (with generous timeout)
	select {
	case <-recorderDone:
		// Recorder finished cleanly
	case <-time.After(5 * time.Second):
		fmt.Fprintf(os.Stderr, "Warning: recorder may still be running\n")
	}

	// Handle any error from the user command (but continue to show summary)
	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %v\n", cmdErr)
	}

	// Show summary using direct API
	fmt.Fprintf(os.Stderr, "\n== Performance Summary ==\n\n")
	summarizer.RunWithOption(cmd.SummaryOpt)

	return nil
}

// killSession kills a running background session (Ruby-compatible, inherited from record)
func (cmd *statCommand) killSession() error {
	fmt.Fprintln(os.Stderr, "kill functionality not yet implemented")
	return fmt.Errorf("not implemented")
}

// showStatus shows status of running session (Ruby-compatible, inherited from record)
func (cmd *statCommand) showStatus() error {
	fmt.Fprintln(os.Stderr, "status functionality not yet implemented") 
	return fmt.Errorf("not implemented")
}


// newStatCommand creates the stat subcommand with direct cobra setting
func newStatCommand() *cobra.Command {
	statCmd := newStatCommandStruct()
	
	cmd := &cobra.Command{
		Use:   "stat [options] -- <command>",
		Short: "Run a command and show performance summary",
		Long:  `Run a command and gather performance information during its execution.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validation moved to PreRunE for cobra integration
			if err := statCmd.validateAndSetCommand(args); err != nil {
				return err
			}
			return statCmd.validateOptions()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Direct execution - no additional validation needed
			return statCmd.run()
		},
	}
	
	// Direct cobra flag setting to RecorderOption fields (no conversion needed) - same as record
	cmd.Flags().StringSliceVarP(&statCmd.RecorderOpt.DevsParts, "disk", "d", statCmd.RecorderOpt.DevsParts, 
		"Device name to be monitored (e.g. sda, sdb, md0, dm-1).")
	cmd.Flags().StringVarP(&statCmd.RecorderOpt.Output, "logfile", "l", statCmd.RecorderOpt.Output, 
		"Output file name")
	
	// Ruby-compatible duration setting (accepts both float64 seconds and duration format)
	cmd.Flags().VarP(&secondsDurationValue{target: &statCmd.RecorderOpt.Interval}, "interval", "i", 
		"Amount of time between each measurement report. Floating point is o.k.")
	cmd.Flags().VarP(&secondsDurationValue{target: &statCmd.RecorderOpt.StartDelay}, "start-delay", "s", 
		"Amount of wait time before starting measurement. Floating point is o.k.")
	cmd.Flags().VarP(&secondsDurationValue{target: &statCmd.RecorderOpt.Timeout}, "timeout", "t", 
		"Amount of measurement time. Floating point is o.k.")
	
	// Control flags (Ruby-specific)
	cmd.Flags().BoolVar(&statCmd.Kill, "kill", statCmd.Kill, 
		"Stop currently running perfmonger-record")
	cmd.Flags().BoolVar(&statCmd.Status, "status", statCmd.Status, 
		"Show currently running perfmonger-record status")
	
	// Feature flags (direct setting to RecorderOption) - same as record
	cmd.Flags().BoolVar(&statCmd.RecordIntr, "record-intr", statCmd.RecordIntr, 
		"Record per core interrupts count (experimental)")
	cmd.Flags().BoolVar(&statCmd.RecorderOpt.NoCPU, "no-cpu", statCmd.RecorderOpt.NoCPU, 
		"Suppress recording CPU usage.")
	cmd.Flags().BoolVar(&statCmd.RecorderOpt.NoNet, "no-net", statCmd.RecorderOpt.NoNet, 
		"Suppress recording network usage")
	cmd.Flags().BoolVar(&statCmd.RecorderOpt.NoMem, "no-mem", statCmd.RecorderOpt.NoMem, 
		"Suppress recording memory usage")
	cmd.Flags().BoolVar(&statCmd.NoGzip, "no-gzip", statCmd.NoGzip, 
		"Do not save a logfile in gzipped format")
	cmd.Flags().BoolVar(&statCmd.RecorderOpt.NoIntervalBackoff, "no-interval-backoff", statCmd.RecorderOpt.NoIntervalBackoff, 
		"Prevent interval to be set longer every after 100 records.")
	
	// Stat-specific flags (direct setting to SummaryOption)
	cmd.Flags().BoolVar(&statCmd.SummaryOpt.JSON, "json", statCmd.SummaryOpt.JSON,
		"Output summary in JSON")
	
	// Debug flags  
	cmd.Flags().BoolVarP(&statCmd.Verbose, "verbose", "v", statCmd.Verbose, 
		"Verbose output")
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}