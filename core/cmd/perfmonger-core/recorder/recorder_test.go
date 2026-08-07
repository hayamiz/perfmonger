package recorder

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
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

// TestGzipBufWriterFlushesBufferOnPanic is a regression test for the bug where
// the bufio.Writer wrapping the gzip writer was not flushed before
// gzip.Writer.Close() ran on panic paths, silently discarding buffered bytes and
// corrupting the output. newGzipBufWriter must return a cleanup that flushes the
// bufio buffer into the gzip writer BEFORE closing it, so that on panic the
// buffered data still reaches the gzip writer before its footer is written. The
// cleanup must also re-raise the original panic. The recovered output must
// therefore decompress to all bytes written before the panic.
func TestGzipBufWriterFlushesBufferOnPanic(t *testing.T) {
	payload := []byte("data written before the panic must survive the unwinding\n")

	var compressed bytes.Buffer

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected body panic to propagate out of the gzip cleanup")
			}
		}()

		out, _, cleanup := newGzipBufWriter(&compressed)
		defer cleanup()

		// Write into the bufio buffer but do NOT flush; the bytes are held in
		// the buffer when the panic unwinds.
		if _, err := out.Write(payload); err != nil {
			t.Fatalf("unexpected write error: %v", err)
		}
		panic("boom in recording body")
	}()

	gz, err := gzip.NewReader(&compressed)
	if err != nil {
		t.Fatalf("gzip output is corrupt/incomplete after panic: %v", err)
	}
	got, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to decompress gzip output after panic: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("buffered data was lost on panic path: got %q, want %q", got, payload)
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

// closeTrackingWriteCloser is an io.WriteCloser that records whether Close was
// called. It stands in for the player process's stdin pipe so tests can assert
// the pipe is released on error paths.
type closeTrackingWriteCloser struct {
	closed bool
}

func (c *closeTrackingWriteCloser) Write(p []byte) (int, error) { return len(p), nil }

func (c *closeTrackingWriteCloser) Close() error {
	c.closed = true
	return nil
}

// fakePlayerPipeSource implements playerPipeSource so setupPlayerPipes can be
// exercised without spawning a real subprocess. StdoutPipe can be made to fail
// to reproduce the fd-leak error path.
type fakePlayerPipeSource struct {
	stdin           *closeTrackingWriteCloser
	stdoutPipeError error
}

func (f *fakePlayerPipeSource) StdinPipe() (io.WriteCloser, error) {
	return f.stdin, nil
}

func (f *fakePlayerPipeSource) StdoutPipe() (io.ReadCloser, error) {
	if f.stdoutPipeError != nil {
		return nil, f.stdoutPipeError
	}
	return io.NopCloser(bytes.NewReader(nil)), nil
}

// TestSetupPlayerPipesClosesStdinWhenStdoutPipeFails is a regression test for the
// bug where, when StdoutPipe() fails during recorder setup, the already-acquired
// stdin pipe was set to nil without being closed, leaking the underlying write-end
// pipe file descriptor. setupPlayerPipes must call Close on the stdin pipe before
// abandoning it on the StdoutPipe failure path.
func TestSetupPlayerPipesClosesStdinWhenStdoutPipeFails(t *testing.T) {
	stdin := &closeTrackingWriteCloser{}
	src := &fakePlayerPipeSource{
		stdin:           stdin,
		stdoutPipeError: errors.New("simulated StdoutPipe failure"),
	}

	gotStdin, gotStdout, err := setupPlayerPipes(src)

	if err == nil {
		t.Fatalf("setupPlayerPipes returned nil error when StdoutPipe failed")
	}
	if gotStdin != nil {
		t.Fatalf("setupPlayerPipes returned a non-nil stdin on the failure path; want nil")
	}
	if gotStdout != nil {
		t.Fatalf("setupPlayerPipes returned a non-nil stdout on the failure path; want nil")
	}
	if !stdin.closed {
		t.Fatalf("stdin pipe was not closed when StdoutPipe failed; the pipe fd is leaked")
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

// TestRunDirectSetupErrorOnUncreatableOutput is a regression test for the bug
// where RunDirect panicked when os.Create failed on the output path. RunDirect
// must instead return a *SetupError (nothing was written), which the record
// subcommand maps onto exit code 1. The error must be distinguishable from a
// *CollectionError.
func TestRunDirectSetupErrorOnUncreatableOutput(t *testing.T) {
	// A path under a non-existent directory cannot be created by os.Create.
	badPath := path.Join(t.TempDir(), "does-not-exist", "out.pgr")

	option := NewRecorderOption()
	option.Output = badPath
	option.Timeout = 10 * time.Millisecond
	option.Interval = 5 * time.Millisecond
	option.NoIntervalBackoff = true

	err := RunDirect(option)
	if err == nil {
		t.Fatalf("RunDirect returned nil for an uncreatable output path; expected a setup error")
	}

	var setupErr *SetupError
	if !errors.As(err, &setupErr) {
		t.Fatalf("RunDirect error is not a *SetupError: got %T (%v)", err, err)
	}

	var collErr *CollectionError
	if errors.As(err, &collErr) {
		t.Fatalf("setup failure was misclassified as a *CollectionError: %v", err)
	}

	if _, statErr := os.Stat(badPath); statErr == nil {
		t.Fatalf("output file %s was created despite the setup error", badPath)
	}
}

// TestRunDirectNormalPathReturnsNil verifies that the normal recording path
// (short timeout, output to a real file) completes without panicking, returns a
// nil error, and leaves a non-empty file on disk (the data has been flushed and
// fsync'd before close).
func TestRunDirectNormalPathReturnsNil(t *testing.T) {
	tmpfile := path.Join(t.TempDir(), "out.pgr")

	option := NewRecorderOption()
	option.Output = tmpfile
	option.Timeout = 20 * time.Millisecond
	option.Interval = 5 * time.Millisecond
	option.NoIntervalBackoff = true
	// Avoid touching interrupt stats which may not be readable in all CI envs.
	option.NoIntr = true

	if err := RunDirect(option); err != nil {
		t.Fatalf("RunDirect returned an error on the normal path: %v", err)
	}

	info, statErr := os.Stat(tmpfile)
	if statErr != nil {
		t.Fatalf("output file was not written: %v", statErr)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty; recorded data was not flushed to disk")
	}
}

// TestRecorderErrorTypesAreDistinct documents the classification contract the
// record subcommand relies on: a *SetupError and a *CollectionError wrapping the
// same underlying error must be matchable by errors.As to their own type only,
// so record.go can choose exit code 1 vs 2 unambiguously.
func TestRecorderErrorTypesAreDistinct(t *testing.T) {
	base := errors.New("underlying failure")

	setup := &SetupError{Err: base}
	coll := &CollectionError{Err: base}

	var s *SetupError
	var c *CollectionError

	if !errors.As(setup, &s) || errors.As(setup, &c) {
		t.Fatalf("SetupError did not classify cleanly as *SetupError only")
	}
	if !errors.As(coll, &c) || errors.As(coll, &s) {
		t.Fatalf("CollectionError did not classify cleanly as *CollectionError only")
	}
	if !errors.Is(setup, base) || !errors.Is(coll, base) {
		t.Fatalf("wrapped error types must unwrap to the underlying error")
	}
}

// TestRunDirectReadyFdEmitsEventAndClosesFd verifies the --ready-fd contract:
// after the first sample is flushed, RunDirect writes exactly one
// newline-terminated ready JSON object carrying the expected keys, and then
// closes the fd so a reader observes EOF.
func TestRunDirectReadyFdEmitsEventAndClosesFd(t *testing.T) {
	readyR, readyW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create ready pipe: %v", err)
	}
	defer readyR.Close()

	tmpfile := path.Join(t.TempDir(), "out.pgr")
	option := NewRecorderOption()
	option.Output = tmpfile
	option.Timeout = 200 * time.Millisecond
	option.Interval = 5 * time.Millisecond
	option.NoIntervalBackoff = true
	option.NoIntr = true
	option.ReadyFd = int(readyW.Fd())

	runErr := make(chan error, 1)
	go func() {
		runErr <- RunDirect(option)
	}()

	// Read everything the recorder writes to the ready fd. RunDirect closes its
	// wrapper over the fd after emitting, so ReadAll returns on EOF.
	data, readErr := io.ReadAll(readyR)
	if readErr != nil {
		t.Fatalf("reading ready fd failed: %v", readErr)
	}
	// The recorder also holds the write end via the fd number; close ours so a
	// lingering descriptor does not keep the pipe open in edge cases.
	readyW.Close()

	if err := <-runErr; err != nil {
		t.Fatalf("RunDirect returned an error: %v", err)
	}

	if len(data) == 0 {
		t.Fatalf("no ready event was written to the ready fd")
	}
	// Exactly one newline-terminated line.
	if data[len(data)-1] != '\n' {
		t.Fatalf("ready event is not newline-terminated: %q", data)
	}
	if n := bytes.Count(data, []byte("\n")); n != 1 {
		t.Fatalf("expected exactly one ready line, got %d: %q", n, data)
	}

	var ev struct {
		Event     string `json:"event"`
		Version   int    `json:"version"`
		Pid       int    `json:"pid"`
		Output    string `json:"output"`
		StartTime string `json:"start_time"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(data), &ev); err != nil {
		t.Fatalf("ready event is not valid JSON: %v (%q)", err, data)
	}
	if ev.Event != "ready" {
		t.Fatalf("ready event has event=%q, want \"ready\"", ev.Event)
	}
	if ev.Version != 1 {
		t.Fatalf("ready event has version=%d, want 1", ev.Version)
	}
	if ev.Pid != os.Getpid() {
		t.Fatalf("ready event has pid=%d, want %d", ev.Pid, os.Getpid())
	}
	if ev.Output != tmpfile {
		t.Fatalf("ready event has output=%q, want %q", ev.Output, tmpfile)
	}
	if _, perr := time.Parse(time.RFC3339Nano, ev.StartTime); perr != nil {
		t.Fatalf("ready event start_time %q is not RFC3339Nano: %v", ev.StartTime, perr)
	}
}

// TestRunDirectStopFdClosesStopsRecording verifies that closing the write end of
// the stop fd stops recording gracefully: RunDirect takes the normal exit path
// and returns nil (exit 0), leaving a valid output file.
func TestRunDirectStopFdClosesStopsRecording(t *testing.T) {
	stopR, stopW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stop pipe: %v", err)
	}
	defer stopR.Close()

	tmpfile := path.Join(t.TempDir(), "out.pgr")
	option := NewRecorderOption()
	option.Output = tmpfile
	// No timeout: the only thing that can stop recording is the stop fd closing.
	option.Interval = 5 * time.Millisecond
	option.NoIntervalBackoff = true
	option.NoIntr = true
	option.StopFd = int(stopR.Fd())

	runErr := make(chan error, 1)
	go func() {
		runErr <- RunDirect(option)
	}()

	// Let a few samples be recorded, then close the write end so the recorder's
	// stop goroutine observes EOF and closes StopCh.
	time.Sleep(50 * time.Millisecond)
	stopW.Close()

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("RunDirect returned an error after stop fd close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("RunDirect did not stop within 5s after the stop fd was closed")
	}

	info, statErr := os.Stat(tmpfile)
	if statErr != nil {
		t.Fatalf("output file was not written: %v", statErr)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty; recorded data was not flushed")
	}
}

// TestRunDirectReadyOrStopFdWithBackgroundIsSetupError verifies that combining
// --ready-fd or --stop-fd with --background is rejected as a *SetupError (exit 1
// at the CLI), because daemonization detaches inherited descriptors.
func TestRunDirectReadyOrStopFdWithBackgroundIsSetupError(t *testing.T) {
	for _, tc := range []struct {
		name    string
		readyFd int
		stopFd  int
	}{
		{"ready-fd", 3, -1},
		{"stop-fd", -1, 3},
		{"both", 3, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			option := NewRecorderOption()
			option.Background = true
			option.ReadyFd = tc.readyFd
			option.StopFd = tc.stopFd

			err := RunDirect(option)
			if err == nil {
				t.Fatalf("RunDirect returned nil; expected a setup error")
			}
			var setupErr *SetupError
			if !errors.As(err, &setupErr) {
				t.Fatalf("RunDirect error is not a *SetupError: got %T (%v)", err, err)
			}
			var collErr *CollectionError
			if errors.As(err, &collErr) {
				t.Fatalf("--background conflict misclassified as *CollectionError: %v", err)
			}
		})
	}
}

// TestBuildTargetDisks verifies that the helper extracting the TargetDisks map
// from a comma-separated disk list populates the map correctly, and returns nil
// for an empty list (meaning "record all devices"). This is the logic shared by
// parseArgs and the direct-API path used by the live/record subcommands.
func TestBuildTargetDisks(t *testing.T) {
	// Empty list means "record all devices": map must be nil.
	if got := BuildTargetDisks(""); got != nil {
		t.Fatalf("BuildTargetDisks(\"\") = %v, want nil", got)
	}

	got := BuildTargetDisks("sda,sdb")
	if got == nil {
		t.Fatalf("BuildTargetDisks(\"sda,sdb\") = nil, want populated map")
	}
	m := *got
	if len(m) != 2 || !m["sda"] || !m["sdb"] {
		t.Fatalf("BuildTargetDisks(\"sda,sdb\") = %v, want {sda:true, sdb:true}", m)
	}

	single := BuildTargetDisks("sda")
	if single == nil || len(*single) != 1 || !(*single)["sda"] {
		t.Fatalf("BuildTargetDisks(\"sda\") = %v, want {sda:true}", single)
	}
}
