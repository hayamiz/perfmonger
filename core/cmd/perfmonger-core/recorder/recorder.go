package recorder

import (
	"bufio"
	"compress/gzip"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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
	StopCh             chan struct{} // External stop signal (closed to stop recording)
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
	fmt.Fprintf(os.Stderr, "=====================\n")
}

// RunWithOption executes the recorder with the provided options
// This is the preferred API that uses direct execution (no double argument parsing)
func RunWithOption(option *RecorderOption) {
	// Call the direct execution function (no args conversion needed)
	RunDirect(option)
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
// the buffered writer together with a cleanup function. The caller MUST invoke
// the cleanup with `defer cleanup()` immediately. The cleanup flushes the bufio
// buffer into the gzip writer BEFORE closing the gzip writer, so that on panic
// paths the buffered bytes reach the gzip writer before its footer is written,
// producing a complete, valid gzip stream instead of a truncated one.
//
// A flush error must never mask a panic that is already unwinding: cleanup
// captures any in-flight panic, flushes, then re-raises the original panic. Only
// when there was no panic is a flush error surfaced (via panic).
func newGzipBufWriter(file io.Writer) (out *bufio.Writer, cleanup func()) {
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
		if flushErr != nil {
			panic(flushErr)
		}
		if closeErr != nil {
			panic(closeErr)
		}
	}
	return out, cleanup
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
	
	// Call the direct execution function
	RunDirect(option)
}

// RunDirect executes the recorder with the provided RecorderOption directly
// This avoids the double conversion: RecorderOption -> args -> parseArgs -> RecorderOption
func RunDirect(option *RecorderOption) {
	// Session file management for background mode.
	// The CLI layer handles duplicate-session detection before launching;
	// here we just write and clean up the PID file.
	if option.Background {
		if _, err := WriteSessionFile(); err != nil {
			panic(err)
		}
		defer RemoveSessionFile()
	}

	var out *bufio.Writer
	var enc *gob.Encoder
	var err error

	if option.Debug {
		showRecorderOption(option)
	}

	hostname, _ := os.Hostname()
	cheader := &ss.CommonHeader{Platform: ss.Linux, Hostname: hostname, StartTime: time.Now()}

	platform_header := ss.NewPlatformHeader()

	if option.ListDevices {
		for _, name := range platform_header.DevsParts {
			os.Stderr.WriteString(name + "\n")
		}
		return
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
		file, err := os.Create(option.Output)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		if player_stdin != nil {
			out = bufio.NewWriter(io.MultiWriter(file, player_stdin))
		} else {
			if option.Gzip {
				// newGzipBufWriter registers its flush-before-close ordering in
				// the returned cleanup, so a panic anywhere in the recording
				// body still drains the bufio buffer into the gzip writer before
				// the gzip footer is written. The normal-path explicit Flush
				// further down makes the cleanup's flush a no-op on success.
				var cleanup func()
				out, cleanup = newGzipBufWriter(file)
				defer cleanup()
			} else {
				out = bufio.NewWriter(file)
			}
		}
	}

	enc = gob.NewEncoder(out)

	// Write the beginning sections
	err = enc.Encode(cheader)
	if err != nil {
		panic(err)
	}

	err = enc.Encode(platform_header)
	if err != nil {
		panic(err)
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
		panic(err)
	}

	if flushErr := out.Flush(); flushErr != nil {
		panic(flushErr)
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