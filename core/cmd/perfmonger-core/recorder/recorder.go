package recorder

import (
	"bufio"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	ss "github.com/hayamiz/perfmonger/core/internal/perfmonger"
)

// RecorderOption represents all options for the recorder component
// This struct is now public for direct use by cobra commands
type RecorderOption struct {
	Interval           time.Duration
	NoIntervalBackoff  bool
	Timeout            time.Duration
	StartDelay         time.Duration
	DevsParts          []string
	Output             string
	NoCPU              bool
	NoIntr             bool
	NoDisk             bool
	NoNet              bool
	NoMem              bool
	Debug              bool
	ListDevices        bool
	PlayerBin          string
	PlayerArgs         []string
	Disks              string
	TargetDisks        *map[string]bool
	Background         bool
	Gzip               bool
	Color              bool
	Pretty             bool
	MetaArgs           []string      // Raw "key=value" metadata args from --meta (percent-encoded)
	StopCh             chan struct{} // External stop signal (closed to stop recording)
	// ReadyFd, when >= 0, is a writable file descriptor to which the recorder
	// writes a single newline-terminated "ready" JSON event after the first
	// sample is flushed, then closes it. -1 disables the ready signal.
	ReadyFd int
	// StopFd, when >= 0, is a file descriptor the recorder reads in a
	// background goroutine; when it reaches EOF (or errors) the recorder stops
	// gracefully by closing StopCh. -1 disables stop-on-fd-close.
	StopFd int
}

// signalNotify and signalStop wrap the os/signal package functions so that
// signal registration/teardown can be observed in tests.
var (
	signalNotify = signal.Notify
	signalStop   = signal.Stop
)

// By default, measurement interval backoff is enabled.
// Minimum resoluton guaranteed: BACKOFF_RATIO / BACKOFF_THRESH
const (
	BACKOFF_THRESH = 1000.0
	BACKOFF_RATIO  = 2.0
)

func parseArgs(args []string, option *RecorderOption) {
	// Create a new FlagSet to avoid conflicts with global flags
	fs := flag.NewFlagSet("recorder", flag.ExitOnError)
	
	// set options
	fs.DurationVar(&option.Interval, "interval",
		time.Second, "Measurement interval")
	fs.BoolVar(&option.NoIntervalBackoff, "no-interval-backoff",
		false, "Disable interval backoff")
	fs.DurationVar(&option.Timeout, "timeout",
		time.Second*0, "Measurement timeout")
	fs.DurationVar(&option.StartDelay, "start-delay",
		time.Second*0, "Wait time before measurement")
	fs.StringVar(&option.Output, "output",
		"-", "Output file name")
	fs.BoolVar(&option.NoCPU, "no-cpu",
		false, "Do not record CPU usage")
	fs.BoolVar(&option.NoIntr, "no-intr",
		false, "Do not record interrupts count")
	fs.BoolVar(&option.NoDisk, "no-disk",
		false, "Do not record disk usage")
	fs.BoolVar(&option.NoNet, "no-net",
		false, "Do not record net usage")
	fs.BoolVar(&option.NoMem, "no-mem",
		false, "Do not record memory usage")
	fs.BoolVar(&option.Debug, "debug",
		false, "Enable debug mode")
	fs.BoolVar(&option.ListDevices, "list-devices",
		false, "List devices and exits")
	fs.BoolVar(&option.Background, "background",
		false, "Run in background mode")
	fs.StringVar(&option.Disks, "disks",
		"", "Disk devices to be monitored")
	fs.StringVar(&option.PlayerBin, "player-bin",
		"", "Run perfmonger-player to show JSON output")
	fs.BoolVar(&option.Gzip, "gzip",
		false, "Save a logfile in gzipped format")
	fs.BoolVar(&option.Color, "color",
		false, "Colored output (for live subcmd)")
	fs.BoolVar(&option.Pretty, "pretty",
		false, "Pretty output (for live subcmd)")
	fs.IntVar(&option.ReadyFd, "ready-fd",
		-1, "Write a ready event to this fd after the first sample is flushed")
	fs.IntVar(&option.StopFd, "stop-fd",
		-1, "Stop recording gracefully when this fd reaches EOF")

	fs.Parse(args)

	if option.PlayerBin == "" && terminal.IsTerminal(int(os.Stdout.Fd())) &&
		option.Output == "-" {
		fmt.Fprintf(os.Stderr, "[recording to data.pgr]\n")
		option.Output = "data.pgr"
	}

	option.TargetDisks = BuildTargetDisks(option.Disks)

	if option.Debug {
		os.Stderr.WriteString(
			fmt.Sprintf(
				`== option
  - output   : %s
  - interval : %s
  - debug    : %t
  - remainings: %s
`,
				option.Output,
				option.Interval.String(),
				option.Debug,
				fmt.Sprint(fs.Args())))
	}
}

// BuildTargetDisks converts a comma-separated disk device list into the
// TargetDisks map used by ReadDiskStats for filtering. An empty list returns
// nil, which ReadDiskStats interprets as "record all devices". This logic is
// shared by parseArgs and the direct-API path (live/record subcommands) so that
// the -d/--disk flag is honored on both paths.
func BuildTargetDisks(disks string) *map[string]bool {
	if disks == "" {
		return nil
	}
	targets := new(map[string]bool)
	*targets = make(map[string]bool)
	for _, dev := range strings.Split(disks, ",") {
		(*targets)[dev] = true
	}
	return targets
}

// ParseMetaArgs parses the raw "key=value" strings supplied via --meta into a
// decoded metadata map. Each argument is split on the FIRST raw '=' so that a
// literal '=' inside a key or value can be carried as the percent escape %3D.
// Both the key and the value are percent-decoded per RFC 3986 using
// url.PathUnescape (not url.QueryUnescape, so a literal '+' is preserved rather
// than turned into a space).
//
// An argument with no '=' is malformed and produces an error, as does an
// argument whose key or value contains an invalid percent-escape sequence.
// Callers in the recorder wrap the returned error in *SetupError so the record
// subcommand maps it onto exit code 1 (nothing was written yet).
//
// It returns nil when args is empty, so a header written without --meta leaves
// CommonHeader.Meta nil.
func ParseMetaArgs(args []string) (map[string]string, error) {
	if len(args) == 0 {
		return nil, nil
	}

	meta := make(map[string]string, len(args))
	for _, arg := range args {
		idx := strings.IndexByte(arg, '=')
		if idx < 0 {
			return nil, fmt.Errorf("invalid --meta argument %q: expected key=value", arg)
		}
		rawKey := arg[:idx]
		rawValue := arg[idx+1:]

		key, err := url.PathUnescape(rawKey)
		if err != nil {
			return nil, fmt.Errorf("invalid --meta key %q: %w", rawKey, err)
		}
		value, err := url.PathUnescape(rawValue)
		if err != nil {
			return nil, fmt.Errorf("invalid --meta value %q: %w", rawValue, err)
		}

		meta[key] = value
	}

	return meta, nil
}

// NewRecorderOption creates a RecorderOption with default values
func NewRecorderOption() *RecorderOption {
	return &RecorderOption{
		Interval:           time.Second,
		NoIntervalBackoff:  false,
		Timeout:            time.Second * 0,
		StartDelay:         time.Second * 0,
		DevsParts:          []string{},
		Output:             "-",
		NoCPU:              false,
		NoIntr:             false,
		NoDisk:             false,
		NoNet:              false,
		NoMem:              false,
		Debug:              false,
		ListDevices:        false,
		PlayerBin:          "",
		PlayerArgs:         []string{},
		Disks:              "",
		TargetDisks:        nil,
		Background:         false,
		Gzip:               false,
		Color:              false,
		Pretty:             false,
		ReadyFd:            -1,
		StopFd:             -1,
	}
}

// showRecorderOption dumps all values in RecorderOption for debugging
func showRecorderOption(option *RecorderOption) {
	fmt.Fprintf(os.Stderr, "=== RecorderOption ===\n")
	fmt.Fprintf(os.Stderr, "Interval: %s\n", option.Interval.String())
	fmt.Fprintf(os.Stderr, "NoIntervalBackoff: %t\n", option.NoIntervalBackoff)
	fmt.Fprintf(os.Stderr, "Timeout: %s\n", option.Timeout.String())
	fmt.Fprintf(os.Stderr, "StartDelay: %s\n", option.StartDelay.String())
	fmt.Fprintf(os.Stderr, "DevsParts: %v\n", option.DevsParts)
	fmt.Fprintf(os.Stderr, "Output: %s\n", option.Output)
	fmt.Fprintf(os.Stderr, "NoCPU: %t\n", option.NoCPU)
	fmt.Fprintf(os.Stderr, "NoIntr: %t\n", option.NoIntr)
	fmt.Fprintf(os.Stderr, "NoDisk: %t\n", option.NoDisk)
	fmt.Fprintf(os.Stderr, "NoNet: %t\n", option.NoNet)
	fmt.Fprintf(os.Stderr, "NoMem: %t\n", option.NoMem)
	fmt.Fprintf(os.Stderr, "Debug: %t\n", option.Debug)
	fmt.Fprintf(os.Stderr, "ListDevices: %t\n", option.ListDevices)
	fmt.Fprintf(os.Stderr, "PlayerBin: %s\n", option.PlayerBin)
	fmt.Fprintf(os.Stderr, "Disks: %s\n", option.Disks)
	if option.TargetDisks != nil {
		fmt.Fprintf(os.Stderr, "TargetDisks: %v\n", *option.TargetDisks)
	} else {
		fmt.Fprintf(os.Stderr, "TargetDisks: nil\n")
	}
	fmt.Fprintf(os.Stderr, "Background: %t\n", option.Background)
	fmt.Fprintf(os.Stderr, "Gzip: %t\n", option.Gzip)
	fmt.Fprintf(os.Stderr, "Color: %t\n", option.Color)
	fmt.Fprintf(os.Stderr, "Pretty: %t\n", option.Pretty)
	fmt.Fprintf(os.Stderr, "ReadyFd: %d\n", option.ReadyFd)
	fmt.Fprintf(os.Stderr, "StopFd: %d\n", option.StopFd)
	fmt.Fprintf(os.Stderr, "=====================\n")
}

// SetupError wraps a failure that happened during recorder setup, before any
// sample was written to the output (e.g. the session file could not be locked,
// or the output path could not be created). Nothing was written, so callers map
// this onto exit code 1.
type SetupError struct {
	Err error
}

func (e *SetupError) Error() string { return e.Err.Error() }

func (e *SetupError) Unwrap() error { return e.Err }

// CollectionError wraps a failure that happened after recording started: during
// header/record encoding, the buffered flush, the fsync, or the file close
// (e.g. ENOSPC). The output may be incomplete, so callers map this onto exit
// code 2.
type CollectionError struct {
	Err error
}

func (e *CollectionError) Error() string { return e.Err.Error() }

func (e *CollectionError) Unwrap() error { return e.Err }

// readyEvent is the payload written to the --ready-fd descriptor once the first
// sample has been flushed and the measurement window is armed. It is serialized
// with encoding/json (rather than hand-formatted) so callers can parse it as a
// stable, well-formed JSON object.
type readyEvent struct {
	Event     string `json:"event"`
	Version   int    `json:"version"`
	Pid       int    `json:"pid"`
	Output    string `json:"output"`
	StartTime string `json:"start_time"`
}

// emitReadyEvent writes a single newline-terminated ready JSON object to f. The
// caller is responsible for closing f afterwards so a reader blocked on the fd
// observes EOF.
func emitReadyEvent(f *os.File, output string) error {
	ev := readyEvent{
		Event:     "ready",
		Version:   1,
		Pid:       os.Getpid(),
		Output:    output,
		StartTime: time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// fdFcntl issues an fcntl(2) with the given command against fd, returning the
// integer result. It is a thin wrapper over the raw syscall because the
// standard library's syscall package does not expose FcntlInt on Linux.
func fdFcntl(fd int, cmd int) (int, error) {
	r, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(cmd), 0)
	if errno != 0 {
		return 0, errno
	}
	return int(r), nil
}

// fdIsValid reports whether fd refers to an open file descriptor in this
// process. F_GETFD fails with EBADF for an unopened descriptor.
func fdIsValid(fd int) bool {
	_, err := fdFcntl(fd, syscall.F_GETFD)
	return err == nil
}

// fdIsWritable reports whether fd is open for writing (O_WRONLY or O_RDWR). It
// is used to reject a --ready-fd that cannot actually receive the ready event.
func fdIsWritable(fd int) bool {
	flags, err := fdFcntl(fd, syscall.F_GETFL)
	if err != nil {
		return false
	}
	accmode := flags & syscall.O_ACCMODE
	return accmode == syscall.O_WRONLY || accmode == syscall.O_RDWR
}

// RunWithOption executes the recorder with the provided options
// This is the preferred API that uses direct execution (no double argument parsing)
func RunWithOption(option *RecorderOption) error {
	// Call the direct execution function (no args conversion needed)
	return RunDirect(option)
}

// SessionFilePath returns the path to the background session PID file.
func SessionFilePath() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return path.Join(os.TempDir(),
		fmt.Sprintf("perfmonger-%s-session.pid", u.Username))
}

// WriteSessionFile atomically writes the current PID to the session file
// under flock protection. Returns the session file path for deferred cleanup.
// If the lock cannot be acquired, it returns an error and does not write the
// PID file, so that mutual exclusion is never silently lost.
func WriteSessionFile() (string, error) {
	sf := SessionFilePath()
	lockfile := path.Join(os.TempDir(), ".perfmonger.lock")

	if _, err := os.Stat(lockfile); err != nil {
		ioutil.WriteFile(lockfile, []byte(""), 0644)
	}
	fd, err := syscall.Open(lockfile, syscall.O_RDONLY, 0000)
	if err != nil {
		return "", fmt.Errorf("failed to open lock file %s: %w", lockfile, err)
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		syscall.Close(fd)
		return "", fmt.Errorf("failed to acquire lock on %s: %w", lockfile, err)
	}

	ioutil.WriteFile(sf, []byte(strconv.Itoa(os.Getpid())), 0644)

	if err := syscall.Flock(fd, syscall.LOCK_UN); err != nil {
		syscall.Close(fd)
		return "", fmt.Errorf("failed to release lock on %s: %w", lockfile, err)
	}
	if err := syscall.Close(fd); err != nil {
		return "", fmt.Errorf("failed to close lock file %s: %w", lockfile, err)
	}

	return sf, nil
}

// RemoveSessionFile removes the session PID file under flock protection.
func RemoveSessionFile() {
	sf := SessionFilePath()
	lockfile := path.Join(os.TempDir(), ".perfmonger.lock")

	fd, err := syscall.Open(lockfile, syscall.O_RDONLY, 0000)
	if err == nil {
		syscall.Flock(fd, syscall.LOCK_EX)
		os.Remove(sf)
		syscall.Flock(fd, syscall.LOCK_UN)
		syscall.Close(fd)
	} else {
		os.Remove(sf)
	}
}

// encodeAndFlush encodes a single record and flushes the buffered writer,
// returning the first error encountered. Propagating the Flush error is
// important: on a full disk (or a broken output file descriptor) the buffered
// data never reaches durable storage, and callers must stop recording instead
// of silently continuing to encode into a failed writer.
func encodeAndFlush(enc *gob.Encoder, out *bufio.Writer, record *ss.StatRecord) error {
	if err := enc.Encode(record); err != nil {
		return err
	}
	return out.Flush()
}

// newGzipBufWriter wraps file in a gzip.Writer and a bufio.Writer and returns
// the buffered writer, the underlying gzip writer, and a cleanup function. The
// caller MUST invoke the cleanup with `defer cleanup()` immediately. The cleanup
// flushes the bufio buffer into the gzip writer BEFORE closing the gzip writer,
// so that on panic paths the buffered bytes reach the gzip writer before its
// footer is written, producing a complete, valid gzip stream instead of a
// truncated one.
//
// A flush error must never mask a panic that is already unwinding: cleanup
// captures any in-flight panic, flushes, closes, then re-raises the original
// panic. On the non-panic path the cleanup is only a safety net; the caller is
// expected to flush the bufio buffer and close the returned gzip writer
// explicitly so that flush/close errors are surfaced as returned errors rather
// than panics. gzip.Writer.Close is idempotent, so the caller's explicit close
// followed by this cleanup's close is safe.
func newGzipBufWriter(file io.Writer) (out *bufio.Writer, gz *gzip.Writer, cleanup func()) {
	gzwriter := gzip.NewWriter(file)
	out = bufio.NewWriter(gzwriter)
	cleanup = func() {
		// Capture any in-flight panic so the buffer can be flushed first.
		p := recover()
		flushErr := out.Flush()
		// Close runs after the flush so the gzip footer is written only once
		// all buffered bytes have entered the gzip writer.
		closeErr := gzwriter.Close()
		if p != nil {
			// Preserve the original panic; flush/close errors must not mask it.
			panic(p)
		}
		// On the non-panic path any flush/close error has already been (or will
		// be) surfaced by the caller's explicit finalization, so it is ignored
		// here to keep cleanup a pure safety net that never panics on success.
		_ = flushErr
		_ = closeErr
	}
	return out, gzwriter, cleanup
}

// playerPipeSource abstracts the parts of *exec.Cmd used while wiring up the
// player subprocess's pipes. It exists so the pipe-setup logic can be exercised
// in tests without spawning a real subprocess.
type playerPipeSource interface {
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
}

// execCmdPipeSource adapts *exec.Cmd to the playerPipeSource interface, since
// exec.Cmd's pipe methods return concrete *os.File types rather than the io
// interface types.
type execCmdPipeSource struct {
	cmd *exec.Cmd
}

func (e execCmdPipeSource) StdinPipe() (io.WriteCloser, error) {
	return e.cmd.StdinPipe()
}

func (e execCmdPipeSource) StdoutPipe() (io.ReadCloser, error) {
	return e.cmd.StdoutPipe()
}

// setupPlayerPipes acquires the player subprocess's stdin and stdout pipes. On
// any failure it returns (nil, nil, err) after releasing pipes already acquired,
// so the caller can fall back without leaking file descriptors. In particular,
// if StdoutPipe() fails after StdinPipe() succeeded, the stdin pipe is closed
// before being abandoned.
func setupPlayerPipes(cmd playerPipeSource) (io.WriteCloser, io.ReadCloser, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		// Release the already-acquired stdin pipe before abandoning it,
		// otherwise its write-end fd is leaked for the process lifetime.
		stdin.Close()
		return nil, nil, err
	}

	return stdin, stdout, nil
}

func Run(args []string) {
	option := NewRecorderOption()

	parseArgs(args, option)

	// Call the direct execution function. This legacy entry point has no
	// caller-supplied error handling, so a failure is surfaced as a panic to
	// keep the previous fail-loud behavior.
	if err := RunDirect(option); err != nil {
		panic(err)
	}
}

// RunDirect executes the recorder with the provided RecorderOption directly
// This avoids the double conversion: RecorderOption -> args -> parseArgs -> RecorderOption
//
// It returns nil on normal completion (timeout, stop-fd close, SIGINT/SIGTERM).
// Setup failures (before any sample is written) are returned as *SetupError and
// collection/write failures (after recording started) as *CollectionError, so
// callers can map them onto the record subcommand's 0/1/2 exit-code contract.
func RunDirect(option *RecorderOption) error {
	// --ready-fd/--stop-fd rely on inherited file descriptors, which background
	// daemonization detaches during the re-exec. Reject the combination up front
	// as a setup error (nothing has been written yet).
	if option.Background && (option.ReadyFd >= 0 || option.StopFd >= 0) {
		return &SetupError{Err: fmt.Errorf("--ready-fd/--stop-fd cannot be used with --background")}
	}

	// Session file management for background mode.
	// The CLI layer handles duplicate-session detection before launching;
	// here we just write and clean up the PID file.
	if option.Background {
		if _, err := WriteSessionFile(); err != nil {
			return &SetupError{Err: err}
		}
		defer RemoveSessionFile()
	}

	var out *bufio.Writer
	var enc *gob.Encoder
	var err error
	// outFile is the real output file when writing to a path (nil for stdout /
	// player pipe). gzwriter is the gzip writer when Gzip output is enabled.
	// Both are finalized explicitly on the normal success path so the data is
	// flushed and fsync'd before the file is closed.
	var outFile *os.File
	var gzwriter *gzip.Writer

	if option.Debug {
		showRecorderOption(option)
	}

	// Parse --meta arguments before creating any output so a malformed value
	// fails as a setup error (exit 1) with nothing written to disk.
	meta, metaErr := ParseMetaArgs(option.MetaArgs)
	if metaErr != nil {
		return &SetupError{Err: metaErr}
	}

	hostname, _ := os.Hostname()
	cheader := &ss.CommonHeader{Platform: ss.Linux, Hostname: hostname, StartTime: time.Now(), Meta: meta}

	platform_header := ss.NewPlatformHeader()

	if option.ListDevices {
		for _, name := range platform_header.DevsParts {
			os.Stderr.WriteString(name + "\n")
		}
		return nil
	}

	// Set up the optional ready descriptor. It is validated here (setup phase,
	// before any output is written) so an invalid/unwritable fd is reported as a
	// *SetupError (exit 1). The event itself is emitted after the first sample is
	// flushed; the deferred Close guarantees a reader blocked on the fd observes
	// EOF even if the recorder fails before arming.
	var readyFile *os.File
	if option.ReadyFd >= 0 {
		if !fdIsValid(option.ReadyFd) || !fdIsWritable(option.ReadyFd) {
			return &SetupError{Err: fmt.Errorf("--ready-fd=%d is not a valid writable file descriptor", option.ReadyFd)}
		}
		readyFile = os.NewFile(uintptr(option.ReadyFd), "ready")
		if readyFile == nil {
			return &SetupError{Err: fmt.Errorf("--ready-fd=%d could not be opened", option.ReadyFd)}
		}
		defer readyFile.Close()
	}

	// Set up the optional stop descriptor. A background goroutine reads it and
	// closes StopCh when it reaches EOF (or errors), reusing the graceful-stop
	// path already wired into the sample loop's select.
	if option.StopFd >= 0 {
		if !fdIsValid(option.StopFd) {
			return &SetupError{Err: fmt.Errorf("--stop-fd=%d is not a valid file descriptor", option.StopFd)}
		}
		stopFile := os.NewFile(uintptr(option.StopFd), "stop")
		if stopFile == nil {
			return &SetupError{Err: fmt.Errorf("--stop-fd=%d could not be opened", option.StopFd)}
		}
		// Close on return so the reader goroutine unblocks and exits even when
		// recording ends via timeout or signal rather than the stop fd.
		defer stopFile.Close()

		if option.StopCh == nil {
			option.StopCh = make(chan struct{})
		}
		// Guard against a double close: the goroutine is the only closer of a
		// channel it created, but sync.Once keeps this safe if the read loop
		// were ever to fall through more than once.
		var stopOnce sync.Once
		closeStop := func() { stopOnce.Do(func() { close(option.StopCh) }) }
		go func() {
			buf := make([]byte, 256)
			for {
				if _, err := stopFile.Read(buf); err != nil {
					break
				}
			}
			closeStop()
		}()
	}

	var player_cmd *exec.Cmd = nil
	var player_stdin io.WriteCloser = nil
	var player_stdout io.ReadCloser = nil
	var player_drain_wg *sync.WaitGroup = nil

	if option.PlayerBin != "" {
		// Build player command arguments: base args + color/pretty flags
		playerArgs := append([]string{}, option.PlayerArgs...)
		if option.Color {
			playerArgs = append(playerArgs, "--color")
		}
		if option.Pretty {
			playerArgs = append(playerArgs, "--pretty")
		}
		player_cmd = exec.Command(option.PlayerBin, playerArgs...)
		player_stdin, player_stdout, err = setupPlayerPipes(execCmdPipeSource{cmd: player_cmd})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set up pipes of %s\n", option.PlayerBin)
			player_cmd = nil
			player_stdin = nil
			player_stdout = nil
		}

		if player_cmd != nil {
			err = player_cmd.Start()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to start %s\n", option.PlayerBin)
				player_cmd = nil
				player_stdin = nil
				player_stdout = nil
			}
		}

		if player_stdout != nil {
			// read stdout of player and write to stdout. The returned
			// WaitGroup is joined after player_cmd.Wait() so that all
			// buffered player output is flushed before RunDirect returns.
			player_drain_wg = startPlayerDrain(player_stdout, os.Stdout)
		}
	}

	if option.Output == "-" {
		out = bufio.NewWriter(os.Stdout)
		if player_stdin != nil {
			out = bufio.NewWriter(player_stdin)
		}
	} else {
		file, createErr := os.Create(option.Output)
		if createErr != nil {
			// The output could not be created; nothing has been written yet.
			return &SetupError{Err: createErr}
		}
		outFile = file
		// Safety net for early-return and panic paths. On the normal path the
		// file is closed explicitly (after fsync) below; the second Close here
		// then returns an already-closed error that is intentionally ignored.
		defer file.Close()

		if player_stdin != nil {
			out = bufio.NewWriter(io.MultiWriter(file, player_stdin))
		} else {
			if option.Gzip {
				// newGzipBufWriter registers its flush-before-close ordering in
				// the returned cleanup, so a panic anywhere in the recording
				// body still drains the bufio buffer into the gzip writer before
				// the gzip footer is written. The normal-path explicit Flush and
				// gzip Close further down make the cleanup a no-op on success.
				var cleanup func()
				out, gzwriter, cleanup = newGzipBufWriter(file)
				defer cleanup()
			} else {
				out = bufio.NewWriter(file)
			}
		}
	}

	enc = gob.NewEncoder(out)

	// Write the beginning sections. The output file already exists, so a failure
	// here means the output may be partially written: classify as a collection
	// error (exit 2) rather than a setup error.
	err = enc.Encode(cheader)
	if err != nil {
		return &CollectionError{Err: err}
	}

	err = enc.Encode(platform_header)
	if err != nil {
		return &CollectionError{Err: err}
	}

	// start delay
	time.Sleep(option.StartDelay)

	var timeout_ch <-chan time.Time
	var timeout_time time.Time
	if option.Timeout == time.Second*0 {
		// dummy channel
		timeout_ch = make(<-chan time.Time)
		timeout_time = time.Now()
	} else {
		timeout_ch = time.After(option.Timeout)
		timeout_time = time.Now().Add(option.Timeout)
	}

	// Loops
	sigint_ch := make(chan os.Signal, 1)
	running := true
	next_time := time.Now()
	record := ss.NewStatRecord()
	backoff_counter := 0

	// cause SIGINT or SIGTERM to break the loop. SIGTERM is the signal sent by
	// systemd, container runtimes, and a plain `kill <pid>`, so it must be
	// handled on the same graceful-shutdown path as SIGINT; otherwise the Go
	// runtime terminates the process without flushing the bufio buffer or
	// closing the gzip writer, corrupting the output file.
	signalNotify(sigint_ch, os.Interrupt, syscall.SIGTERM)
	// Deregister the handler on return so the channel is not leaked and does
	// not keep silently consuming signals after RunDirect exits.
	defer signalStop(sigint_ch)

	for {
		record.Time = time.Now()

		if !option.NoCPU {
			ss.ReadCpuStat(record)
		}
		if !option.NoIntr {
			ss.ReadInterruptStat(record)
		}
		if !option.NoDisk {
			ss.ReadDiskStats(record, option.TargetDisks)
		}
		if !option.NoNet {
			ss.ReadNetStat(record)
		}
		if !option.NoMem {
			ss.ReadMemStat(record)
		}

		// Encode the record and flush it to durable storage. If either the
		// encode or the flush fails (e.g. the disk is full), stop recording so
		// the process exits non-zero instead of silently dropping data.
		err = encodeAndFlush(enc, out, record)
		if err != nil {
			break
		}

		// Emit the ready event once, immediately after the first sample has been
		// flushed (baseline captured, measurement window armed), then close the
		// fd. readyFile is niled so this fires exactly once.
		if readyFile != nil {
			if emitErr := emitReadyEvent(readyFile, option.Output); emitErr != nil {
				return &CollectionError{Err: emitErr}
			}
			readyFile.Close()
			readyFile = nil
		}

		if !running {
			break
		}

		if !option.NoIntervalBackoff {
			backoff_counter++
			if backoff_counter >= BACKOFF_THRESH {
				backoff_counter -= BACKOFF_THRESH

				option.Interval *= BACKOFF_RATIO
				if option.Interval.Seconds() > 3600.0 {
					option.Interval = time.Hour
				}
			}
		}

		next_time = next_time.Add(option.Interval)

		// Build a nil-safe stop channel (nil channels block forever in select)
		var stopCh <-chan struct{}
		if option.StopCh != nil {
			stopCh = option.StopCh
		}

		// wait for next iteration
		select {
		case <-sigint_ch:
			running = false
			break
		case <-timeout_ch:
			running = false
			break
		case <-stopCh:
			running = false
			break
		case <-time.After(next_time.Sub(time.Now())):
			break
		}

		// If next_time and timeout_time is very close,
		// avoid recording twice in a very short time
		if option.Timeout != time.Second*0 &&
			timeout_time.Sub(next_time).Seconds() < 0.01 {
			running = false
		}
	}

	// If the loop exited because of a write/flush failure, the output is
	// already truncated or corrupted. Propagate the failure so the process
	// exits non-zero instead of reporting success on an unusable file.
	if err != nil {
		return &CollectionError{Err: err}
	}

	// Normal finalization: flush the buffered data, write the gzip trailer (if
	// any), fsync the file, then close it, so a successful return guarantees the
	// recorded data is on durable storage.
	if flushErr := out.Flush(); flushErr != nil {
		return &CollectionError{Err: flushErr}
	}

	if gzwriter != nil {
		// Close writes the gzip trailer. gzip.Writer.Close is idempotent, so
		// the deferred cleanup's later close is a harmless no-op.
		if closeErr := gzwriter.Close(); closeErr != nil {
			return &CollectionError{Err: closeErr}
		}
	}

	if outFile != nil {
		// fsync before close so a successful exit guarantees the data reached
		// durable storage. Sync is skipped for non-regular outputs (pipes,
		// devices) where it may fail even though the write succeeded.
		if syncErr := syncFile(outFile); syncErr != nil {
			return &CollectionError{Err: syncErr}
		}
		if closeErr := outFile.Close(); closeErr != nil {
			return &CollectionError{Err: closeErr}
		}
	}

	if player_stdin != nil {
		player_stdin.Close()
		_ = player_cmd.Wait()
		// Join the stdout draining goroutine so that all buffered player
		// output is flushed to stdout before RunDirect returns. Without this,
		// the process could exit while the goroutine is still copying, leading
		// to truncated output or a data race on os.Stdout at shutdown.
		if player_drain_wg != nil {
			player_drain_wg.Wait()
		}
	}

	return nil
}

// syncFile flushes the file's data to durable storage before it is closed. Sync
// is only meaningful for regular files; for pipes, character devices, or similar
// special outputs Sync may return an error (e.g. EINVAL) even though every write
// succeeded, so those errors are ignored to avoid failing an otherwise-good run.
func syncFile(f *os.File) error {
	if info, statErr := f.Stat(); statErr == nil && !info.Mode().IsRegular() {
		return nil
	}
	return f.Sync()
}

// startPlayerDrain spawns a goroutine that copies all data from the player's
// stdout reader to the destination writer (normally os.Stdout). It returns a
// *sync.WaitGroup whose counter is incremented before the goroutine starts and
// decremented when the goroutine finishes draining. Callers must Wait on the
// returned group before returning so that all buffered player output is flushed.
func startPlayerDrain(r io.Reader, w io.Writer) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var buf = make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				w.Write(buf[0:n])
			}
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}
		}
	}()
	return &wg
}