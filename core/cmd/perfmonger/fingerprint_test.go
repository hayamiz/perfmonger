package main

import (
	"testing"
)

// TestRunCommandRecordsErrorOnFailure verifies that runCommand surfaces command
// failures by appending them to f.errors and returning a non-nil error, instead
// of silently discarding them (ticket 0031).
func TestRunCommandRecordsErrorOnFailure(t *testing.T) {
	f := &fingerprintCommand{errors: []error{}}

	// Use a command that is guaranteed not to exist so exec.Command fails.
	_, err := f.runCommand("perfmonger-nonexistent-command-xyz")
	if err == nil {
		t.Errorf("expected non-nil error for a failing command, got nil")
	}
	if len(f.errors) != 1 {
		t.Errorf("expected the command error to be recorded in f.errors, got %d errors", len(f.errors))
	}
}

// TestRunCommandSucceeds verifies that a successful command returns output and no
// error, and does not pollute f.errors.
func TestRunCommandSucceeds(t *testing.T) {
	f := &fingerprintCommand{errors: []error{}}

	output, err := f.runCommand("true")
	if err != nil {
		t.Errorf("expected nil error for a successful command, got %v", err)
	}
	if len(f.errors) != 0 {
		t.Errorf("expected no recorded errors for a successful command, got %d", len(f.errors))
	}
	_ = output
}
