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
	Disks              string
	TargetDisks        *map[string]bool
	Background         bool
	Gzip               bool
	Color              bool
	Pretty             bool
}

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

	if option.Disks == "" {
		option.TargetDisks = nil
	} else {
		option.TargetDisks = new(map[string]bool)
		*option.TargetDisks = make(map[string]bool)
		for _, dev := range strings.Split(option.Disks, ",") {
			(*option.TargetDisks)[dev] = true
		}
	}

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
func WriteSessionFile() string {
	sf := SessionFilePath()
	lockfile := path.Join(os.TempDir(), ".perfmonger.lock")

	if _, err := os.Stat(lockfile); err != nil {
		ioutil.WriteFile(lockfile, []byte(""), 0644)
	}
	fd, _ := syscall.Open(lockfile, syscall.O_RDONLY, 0000)
	syscall.Flock(fd, syscall.LOCK_EX)

	ioutil.WriteFile(sf, []byte(strconv.Itoa(os.Getpid())), 0644)

	syscall.Flock(fd, syscall.LOCK_UN)
	syscall.Close(fd)

	return sf
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
		WriteSessionFile()
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

	if option.PlayerBin != "" {
		if option.Color {
			if option.Pretty {
				player_cmd = exec.Command(option.PlayerBin, "-color", "-pretty")
			} else {
				player_cmd = exec.Command(option.PlayerBin, "-color")
			}
		} else {
			player_cmd = exec.Command(option.PlayerBin)
		}
		player_stdin, err = player_cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get stdin of %s", option.PlayerBin)
			player_cmd = nil
			player_stdin = nil
		}
		player_stdout, err = player_cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get stdout of %s", option.PlayerBin)
			player_cmd = nil
			player_stdin = nil
			player_stdout = nil
		}

		err = player_cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start %s", option.PlayerBin)
			player_cmd = nil
			player_stdin = nil
			player_stdout = nil
		}

		// read stdout of player and write to stdout
		go func() {
			var buf = make([]byte, 4096)
			for {
				n, err := player_stdout.Read(buf)
				if err == io.EOF {
					break
				} else if err != nil {
					panic(err)
				}

				if n == 0 {
					continue
				}
				os.Stdout.Write(buf[0:n])
			}
		}()
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
				gzwriter := gzip.NewWriter(file)
				defer gzwriter.Close()

				out = bufio.NewWriter(gzwriter)
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

	// cause SIGINT to break loop
	signal.Notify(sigint_ch, os.Interrupt)

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

		err = enc.Encode(record)
		if err != nil {
			break
		}
		out.Flush()

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

		// wait for next iteration
		select {
		case <-sigint_ch:
			running = false
			break
		case <-timeout_ch:
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

	out.Flush()

	if player_stdin != nil {
		player_stdin.Close()
		_ = player_cmd.Wait()
	}
}