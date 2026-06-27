package recorder

import (
	"os"
	"path"
	"strconv"
	"testing"
	"time"
)

// TestWriteSessionFileSuccess verifies that WriteSessionFile writes the current
// PID under a writable TMPDIR and returns the session file path with no error.
func TestWriteSessionFileSuccess(t *testing.T) {
	tmpdir := t.TempDir()
	t.Setenv("TMPDIR", tmpdir)

	sf, err := WriteSessionFile()
	if err != nil {
		t.Fatalf("WriteSessionFile returned an unexpected error: %v", err)
	}
	defer RemoveSessionFile()

	data, err := os.ReadFile(sf)
	if err != nil {
		t.Fatalf("session file was not written: %v", err)
	}
	if got := string(data); got != strconv.Itoa(os.Getpid()) {
		t.Fatalf("session file contains %q, want PID %d", got, os.Getpid())
	}
}

// TestWriteSessionFileLockFailureDoesNotWritePID is a regression test for the
// bug where WriteSessionFile discarded the syscall.Open error and proceeded to
// write the PID file without holding the lock. The TMPDIR here is writable (so
// the session PID path is reachable), but the lock file is made unreadable so
// that syscall.Open fails. WriteSessionFile must then return an error and must
// NOT write the session PID file.
func TestWriteSessionFileLockFailureDoesNotWritePID(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("file permissions are not enforced for root; cannot force lock open failure")
	}

	tmpdir := t.TempDir()
	t.Setenv("TMPDIR", tmpdir)

	// Pre-create the lock file with no permissions so syscall.Open(O_RDONLY)
	// fails with EACCES, simulating an unacquirable lock.
	lockfile := path.Join(tmpdir, ".perfmonger.lock")
	if err := os.WriteFile(lockfile, []byte(""), 0000); err != nil {
		t.Fatalf("failed to create unreadable lock file: %v", err)
	}

	sf := SessionFilePath()

	_, err := WriteSessionFile()
	if err == nil {
		t.Fatalf("WriteSessionFile returned nil error when the lock could not be acquired")
	}

	if _, statErr := os.Stat(sf); statErr == nil {
		t.Fatalf("session PID file %s was written despite lock acquisition failure", sf)
	}
}

// TestRunDirectStopsSignalNotify is a regression test for the bug where
// RunDirect called signal.Notify on a fresh channel but never paired it with
// signal.Stop, leaking the registration past the function's return. The test
// installs observable seams over the signal package and asserts that the exact
// channel passed to signal.Notify is later handed to signal.Stop before
// RunDirect returns.
func TestRunDirectStopsSignalNotify(t *testing.T) {
	var notifiedCh chan<- os.Signal
	var stoppedCh chan<- os.Signal

	origNotify := signalNotify
	origStop := signalStop
	t.Cleanup(func() {
		signalNotify = origNotify
		signalStop = origStop
	})

	signalNotify = func(c chan<- os.Signal, sig ...os.Signal) {
		notifiedCh = c
		// Do not register with the real signal package during the test.
	}
	signalStop = func(c chan<- os.Signal) {
		stoppedCh = c
	}

	// Write to a temp file so RunDirect does not touch real stdout, and use a
	// short timeout so the recording loop exits promptly.
	tmpfile := path.Join(t.TempDir(), "out.pgr")
	option := NewRecorderOption()
	option.Output = tmpfile
	option.Timeout = 10 * time.Millisecond
	option.Interval = 5 * time.Millisecond
	option.NoIntervalBackoff = true

	RunDirect(option)

	if notifiedCh == nil {
		t.Fatalf("signal.Notify was never called by RunDirect")
	}
	if stoppedCh == nil {
		t.Fatalf("signal.Stop was never called by RunDirect; the SIGINT registration leaks past return")
	}
	if notifiedCh != stoppedCh {
		t.Fatalf("signal.Stop was called with a different channel than signal.Notify; registration not torn down")
	}
}
