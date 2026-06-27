package recorder

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"os"
	"path"
	"strconv"
	"syscall"
	"testing"
	"time"

	ss "github.com/hayamiz/perfmonger/core/internal/perfmonger"
)

// slowReader emits its payload one byte at a time with a small delay between
// reads, then returns io.EOF. It simulates a player process whose stdout is
// still being drained when the player process itself has already exited, so a
// caller that does not join the draining goroutine would observe truncated
// output.
type slowReader struct {
	data []byte
	pos  int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	time.Sleep(time.Millisecond)
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

// TestStartPlayerDrainJoinsBeforeReturn is a regression test for the bug where
// the goroutine draining the player's stdout was not joined before RunDirect
// returned, so buffered player output could be truncated or lost. startPlayerDrain
// must return a join handle (sync.WaitGroup) such that, after waiting on it, all
// bytes from the reader have been copied to the destination writer.
func TestStartPlayerDrainJoinsBeforeReturn(t *testing.T) {
	payload := []byte("the quick brown fox jumps over the lazy dog\n")
	src := &slowReader{data: payload}
	var dst bytes.Buffer

	wg := startPlayerDrain(src, &dst)
	// Joining must guarantee the drain goroutine has fully completed.
	wg.Wait()

	if got := dst.Bytes(); !bytes.Equal(got, payload) {
		t.Fatalf("player output truncated/lost after join: got %q, want %q", got, payload)
	}
}

// failingWriter is an io.Writer that always returns an error, simulating a
// full disk or an otherwise broken output destination.
type failingWriter struct{}

var errDiskFull = errors.New("simulated disk full")

func (failingWriter) Write(p []byte) (int, error) {
	return 0, errDiskFull
}

// TestEncodeAndFlushPropagatesFlushError is a regression test for the bug where
// the recording loop ignored the error returned by out.Flush(). When the
// underlying writer fails (e.g. a full disk), encodeAndFlush must surface the
// error so the loop can break and the process can exit non-zero, instead of
// silently dropping data.
func TestEncodeAndFlushPropagatesFlushError(t *testing.T) {
	// A small bufio buffer ensures the encoded record is held in the buffer and
	// only forced out to the failing writer on Flush.
	out := bufio.NewWriter(failingWriter{})
	enc := gob.NewEncoder(out)
	record := ss.NewStatRecord()
	record.Time = time.Now()

	if err := encodeAndFlush(enc, out, record); err == nil {
		t.Fatalf("encodeAndFlush returned nil error when the underlying writer failed; the flush error was swallowed")
	}
}

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

// TestRunDirectHandlesSIGTERM is a regression test for the bug where RunDirect
// registered only os.Interrupt (SIGINT) with signal.Notify, leaving SIGTERM to
// terminate the process abruptly without flushing the bufio buffer or closing
// the gzip writer, corrupting the output file. RunDirect must register
// syscall.SIGTERM on the same graceful-shutdown path as SIGINT.
func TestRunDirectHandlesSIGTERM(t *testing.T) {
	var notifiedSignals []os.Signal

	origNotify := signalNotify
	origStop := signalStop
	t.Cleanup(func() {
		signalNotify = origNotify
		signalStop = origStop
	})

	signalNotify = func(c chan<- os.Signal, sig ...os.Signal) {
		notifiedSignals = append(notifiedSignals, sig...)
		// Do not register with the real signal package during the test.
	}
	signalStop = func(c chan<- os.Signal) {}

	tmpfile := path.Join(t.TempDir(), "out.pgr")
	option := NewRecorderOption()
	option.Output = tmpfile
	option.Timeout = 10 * time.Millisecond
	option.Interval = 5 * time.Millisecond
	option.NoIntervalBackoff = true

	RunDirect(option)

	sawInterrupt := false
	sawSIGTERM := false
	for _, sig := range notifiedSignals {
		if sig == os.Interrupt {
			sawInterrupt = true
		}
		if sig == syscall.SIGTERM {
			sawSIGTERM = true
		}
	}

	if !sawInterrupt {
		t.Fatalf("signal.Notify was not called with os.Interrupt; got %v", notifiedSignals)
	}
	if !sawSIGTERM {
		t.Fatalf("signal.Notify was not called with syscall.SIGTERM; SIGTERM is not handled gracefully (got %v)", notifiedSignals)
	}
}
