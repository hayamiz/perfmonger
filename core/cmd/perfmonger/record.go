package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/recorder"
)

// Environment variable used as a sentinel for the re-exec daemonization pattern.
// When set, the process knows it is the background child and should not fork again.
const daemonEnvKey = "PERFMONGER_DAEMON_CHILD"

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
	opt.Output = "perfmonger.pgr.gz"  // default logfile name
	opt.NoNet = true                  // default: don't record network
	opt.Gzip = true                   // default: use gzip
	
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

// sessionFilePath returns the path to the session PID file
func sessionFilePath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("perfmonger-%s-session.pid", u.Username)), nil
}

// lockFilePath returns the path to the lock file
func lockFilePath() string {
	return filepath.Join(os.TempDir(), ".perfmonger.lock")
}

// killSession kills a running background session (Ruby-compatible)
func (cmd *recordCommand) killSession() error {
	pid := cmd.getRunningSessionPID()
	if pid == 0 {
		fmt.Fprintln(os.Stderr, "[ERROR] No perfmonger record session is running.")
		return fmt.Errorf("no session running")
	}

	// Send SIGINT and wait with exponential backoff (Ruby-compatible)
	sleepTime := 50 * time.Millisecond
	for try := 0; try < 5; try++ {
		if err := syscall.Kill(pid, syscall.SIGINT); err != nil {
			// Process already gone — success
			break
		}
		time.Sleep(sleepTime)
		sleepTime *= 2
	}

	// Final check: is the process still alive?
	if err := syscall.Kill(pid, 0); err == nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Cannot stop perfmonger record session correctly. PID=%d\n", pid)
		return fmt.Errorf("failed to stop session PID=%d", pid)
	}

	// Clean up session file
	sf, err := sessionFilePath()
	if err == nil {
		os.Remove(sf)
	}

	return nil
}

// showStatus shows status of running session (Ruby-compatible)
func (cmd *recordCommand) showStatus() error {
	pid := cmd.getRunningSessionPID()
	if pid == 0 {
		fmt.Fprintln(os.Stderr, "[ERROR] No perfmonger record session is running.")
		return fmt.Errorf("no session running")
	}

	// Read /proc/<PID>/cmdline
	cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] No perfmonger-recorder is running.")
		return fmt.Errorf("cannot read process info")
	}

	parts := strings.Split(string(cmdlineBytes), "\x00")
	exe := ""
	args := ""
	if len(parts) > 0 {
		exe = parts[0]
	}
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	// Get start time from /proc/<PID> mtime
	procInfo, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	if err != nil {
		fmt.Fprintln(os.Stderr, "[ERROR] No perfmonger-recorder is running.")
		return fmt.Errorf("cannot stat process")
	}
	startTime := procInfo.ModTime()
	elapsed := int(time.Since(startTime).Seconds())

	fmt.Printf("==== perfmonger record is running (PID: %d) ====\n\n", pid)
	fmt.Printf("* Running executable: %s\n", exe)
	fmt.Printf("* Arguments: %s\n", args)
	fmt.Printf("* Started at %s (running %d sec)\n\n", startTime.Format(time.RFC3339), elapsed)

	return nil
}

// getRunningSessionPID returns PID of running session, 0 if none
func (cmd *recordCommand) getRunningSessionPID() int {
	sf, err := sessionFilePath()
	if err != nil {
		return 0
	}

	data, err := os.ReadFile(sf)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}

	// Check if the process is actually alive
	if err := syscall.Kill(pid, 0); err != nil {
		// Process is dead — clean up stale session file
		lf := lockFilePath()
		fd, err := syscall.Open(lf, syscall.O_RDONLY|syscall.O_CREAT, 0644)
		if err == nil {
			syscall.Flock(fd, syscall.LOCK_EX)
			os.Remove(sf)
			syscall.Flock(fd, syscall.LOCK_UN)
			syscall.Close(fd)
		}
		return 0
	}

	return pid
}

// executeRecord runs the actual recording using direct API with minimal processing
func (cmd *recordCommand) executeRecord() error {
	// Apply Ruby-specific logic (minimal processing only)
	cmd.applyRubySpecificLogic()

	if os.Getenv("PERFMONGER_DEBUG") != "" {
		cmd.RecorderOpt.Debug = true
	}

	// Resolve output path to absolute BEFORE daemonize (which changes cwd to /)
	if cmd.RecorderOpt.Output != "-" && !filepath.IsAbs(cmd.RecorderOpt.Output) {
		absPath, err := filepath.Abs(cmd.RecorderOpt.Output)
		if err != nil {
			return fmt.Errorf("failed to resolve output path: %v", err)
		}
		cmd.RecorderOpt.Output = absPath
	}

	// Handle background mode: re-exec as a detached child process.
	// Go's runtime is not fork-safe (goroutines, GC), so we use the
	// re-exec pattern: the parent launches a copy of itself with a
	// sentinel env var, then exits. The child detects the sentinel,
	// skips daemonization, and runs the recorder directly.
	if cmd.RecorderOpt.Background {
		if os.Getenv(daemonEnvKey) == "" {
			// Parent: launch child and exit
			return cmd.launchDaemonChild()
		}
		// Child: continue to recording (don't print banner)
	} else {
		fmt.Printf("[recording to %s]\n", cmd.RecorderOpt.Output)
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

// launchDaemonChild re-execs the current binary as a detached background process.
// The parent returns nil after launching the child; the caller should then exit.
func (cmd *recordCommand) launchDaemonChild() error {
	// Build argument list that mirrors the current invocation but with the
	// output path already resolved to absolute.
	args := []string{"record", "--background"}
	args = append(args, "--timeout", fmt.Sprintf("%g", cmd.RecorderOpt.Timeout.Seconds()))
	args = append(args, "--interval", fmt.Sprintf("%g", cmd.RecorderOpt.Interval.Seconds()))
	if cmd.RecorderOpt.StartDelay > 0 {
		args = append(args, "--start-delay", fmt.Sprintf("%g", cmd.RecorderOpt.StartDelay.Seconds()))
	}
	args = append(args, "-l", cmd.RecorderOpt.Output)
	if cmd.RecorderOpt.NoCPU {
		args = append(args, "--no-cpu")
	}
	if cmd.RecorderOpt.NoNet {
		args = append(args, "--no-net")
	}
	if cmd.RecorderOpt.NoMem {
		args = append(args, "--no-mem")
	}
	if cmd.RecorderOpt.NoIntr || !cmd.RecordIntr {
		args = append(args, "--record-intr=false")
	}
	if cmd.RecorderOpt.NoIntervalBackoff {
		args = append(args, "--no-interval-backoff")
	}
	if cmd.NoGzip {
		args = append(args, "--no-gzip")
	}
	for _, d := range cmd.RecorderOpt.DevsParts {
		args = append(args, "-d", d)
	}

	selfBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find own executable: %v", err)
	}

	child := exec.Command(selfBin, args...)
	child.Dir = "/"
	child.Env = append(os.Environ(), daemonEnvKey+"=1")
	// Detach stdin/stdout/stderr → /dev/null
	child.Stdin = nil
	child.Stdout = nil
	child.Stderr = nil
	// Create new process group / session so the child survives parent exit
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := child.Start(); err != nil {
		return fmt.Errorf("failed to start background process: %v", err)
	}

	// Parent is done — the child runs independently
	return nil
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