package recorder

import (
	"os"
	"path"
	"strconv"
	"testing"
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
