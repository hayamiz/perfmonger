package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/summarizer"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// summaryCommand represents the summary command with direct SummaryOption setting
type summaryCommand struct {
	// Direct field (no embedding) for maximum efficiency
	SummaryOpt *summarizer.SummaryOption

	// Ruby-specific options only
	Pager   string
	NoPager bool
}

// newSummaryCommandStruct creates summaryCommand with Ruby-compatible defaults
func newSummaryCommandStruct() *summaryCommand {
	return &summaryCommand{
		SummaryOpt: summarizer.NewSummaryOption(),
		Pager:      "",
		NoPager:    false,
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

// resolvePager determines which pager command to use.
// Priority:
//  1. --pager flag value (explicit non-empty value)
//  2. PAGER environment variable
//
// An empty return value means "do not use a pager".
func (cmd *summaryCommand) resolvePager() string {
	if cmd.Pager != "" {
		return cmd.Pager
	}
	return os.Getenv("PAGER")
}

// isStdoutTTY reports whether stdout is connected to a terminal.
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// run executes the summary command with direct API calls
func (cmd *summaryCommand) run() error {
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running summarizer with options: %+v\n", cmd.SummaryOpt)
	}

	return cmd.runWithPager()
}

// runWithPager runs the summarizer, optionally piping output through a pager.
// The pager is invoked only when all of the following are true:
//   - --no-pager is NOT specified
//   - a non-empty pager command is resolved (--pager flag or PAGER env)
//   - stdout is a TTY
//
// Otherwise, the summary is written directly to stdout.
// If the pager fails to start, a warning is emitted and output falls back to stdout.
func (cmd *summaryCommand) runWithPager() error {
	// --no-pager short-circuits everything: always write to stdout.
	if cmd.NoPager {
		return summarizer.RunDirect(cmd.SummaryOpt, os.Stdout)
	}

	pager := cmd.resolvePager()
	if pager == "" || !isStdoutTTY() {
		return summarizer.RunDirect(cmd.SummaryOpt, os.Stdout)
	}

	// Split the pager string into argv. This supports things like
	// PAGER="less -R" without needing a shell.
	fields := strings.Fields(pager)
	if len(fields) == 0 {
		return summarizer.RunDirect(cmd.SummaryOpt, os.Stdout)
	}

	pagerCmd := exec.Command(fields[0], fields[1:]...)
	pagerCmd.Stdout = os.Stdout
	pagerCmd.Stderr = os.Stderr

	stdinPipe, err := pagerCmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Warning: failed to open pager stdin %q: %v. Falling back to stdout.\n",
			pager, err)
		return summarizer.RunDirect(cmd.SummaryOpt, os.Stdout)
	}

	if err := pagerCmd.Start(); err != nil {
		// Closing the pipe is best-effort here; the process never started.
		_ = stdinPipe.Close()
		fmt.Fprintf(os.Stderr,
			"Warning: failed to start pager %q: %v. Falling back to stdout.\n",
			pager, err)
		return summarizer.RunDirect(cmd.SummaryOpt, os.Stdout)
	}

	// Stream summary output into the pager. A broken pipe (user quit the
	// pager, e.g. pressed 'q') is treated as non-fatal.
	runErr := summarizer.RunDirect(cmd.SummaryOpt, stdinPipe)
	// Close the writing end so the pager sees EOF.
	_ = stdinPipe.Close()

	if runErr != nil && !isBrokenPipe(runErr) {
		// Still wait for the pager to exit to avoid leaving a zombie.
		_ = pagerCmd.Wait()
		return runErr
	}

	if err := pagerCmd.Wait(); err != nil {
		// If the user quit early with 'q', the pager may report a non-zero
		// exit code; do not bubble that up as a perfmonger error.
		return nil
	}
	return nil
}

// isBrokenPipe reports whether err originates from a closed pipe write
// (typically because the pager process terminated before consuming all input).
func isBrokenPipe(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	// fmt.Fprintf and friends wrap syscall EPIPE errors in fs.PathError /
	// os.SyscallError on some platforms; match the message as a fallback.
	msg := err.Error()
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "file already closed")
}

// newSummaryCommand creates the summary subcommand with direct cobra setting
func newSummaryCommand() *cobra.Command {
	summaryCmd := newSummaryCommandStruct()

	cmd := &cobra.Command{
		Use:   "summary [options] LOG_FILE",
		Short: "Summarize system performance data",
		Long:  `Show a summary of a perfmonger log file`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return summaryCmd.validateAndSetLogfile(args)
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
		"Use the given command as pager for summary output (overrides PAGER)")
	cmd.Flags().BoolVar(&summaryCmd.NoPager, "no-pager", summaryCmd.NoPager,
		"Disable paging; write summary directly to stdout")
	cmd.Flags().StringVar(&summaryCmd.SummaryOpt.DiskOnly, "disk-only", summaryCmd.SummaryOpt.DiskOnly,
		"Select disk devices that matches REGEX (Ex. 'sd[b-d]')")

	// Add aliases
	cmd.Aliases = []string{"summarize"}

	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}
