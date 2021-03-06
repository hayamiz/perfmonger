//usr/bin/env go run $0 $@ ; exit

package main

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

	ss "github.com/hayamiz/perfmonger/core/subsystem"
)

type RecorderOption struct {
	interval            time.Duration
	no_interval_backoff bool
	timeout             time.Duration
	start_delay         time.Duration
	devsParts           []string
	output              string
	no_cpu              bool
	no_intr             bool
	no_disk             bool
	no_net              bool
	no_mem              bool
	debug               bool
	listDevices         bool
	player_bin          string
	disks               string
	targetDisks         *map[string]bool
	background          bool
	gzip                bool
	color               bool
	pretty              bool
}

var option RecorderOption

// By default, measurement interval backoff is enabled.
// Minimum resoluton guaranteed: BACKOFF_RATIO / BACKOFF_THRESH
const (
	BACKOFF_THRESH = 1000.0
	BACKOFF_RATIO  = 2.0
)

func parseArgs() {
	// set options
	flag.DurationVar(&option.interval, "interval",
		time.Second, "Measurement interval")
	flag.BoolVar(&option.no_interval_backoff, "no-interval-backoff",
		false, "Disable interval backoff")
	flag.DurationVar(&option.timeout, "timeout",
		time.Second*0, "Measurement timeout")
	flag.DurationVar(&option.start_delay, "start-delay",
		time.Second*0, "Wait time before measurement")
	flag.StringVar(&option.output, "output",
		"-", "Output file name")
	flag.BoolVar(&option.no_cpu, "no-cpu",
		false, "Do not record CPU usage")
	flag.BoolVar(&option.no_intr, "no-intr",
		false, "Do not record interrupts count")
	flag.BoolVar(&option.no_disk, "no-disk",
		false, "Do not record disk usage")
	flag.BoolVar(&option.no_net, "no-net",
		false, "Do not record net usage")
	flag.BoolVar(&option.no_mem, "no-mem",
		false, "Do not record memory usage")
	flag.BoolVar(&option.debug, "debug",
		false, "Enable debug mode")
	flag.BoolVar(&option.listDevices, "list-devices",
		false, "List devices and exits")
	flag.BoolVar(&option.background, "background",
		false, "Run in background mode")
	flag.StringVar(&option.disks, "disks",
		"", "Disk devices to be monitored")
	flag.StringVar(&option.player_bin, "player-bin",
		"", "Run perfmonger-player to show JSON output")
	flag.BoolVar(&option.gzip, "gzip",
		false, "Save a logfile in gzipped format")
	flag.BoolVar(&option.color, "color",
		false, "Colored output (for live subcmd)")
	flag.BoolVar(&option.pretty, "pretty",
		false, "Pretty output (for live subcmd)")

	flag.Parse()

	if option.player_bin == "" && terminal.IsTerminal(int(os.Stdout.Fd())) &&
		option.output == "-" {
		fmt.Fprintf(os.Stderr, "[recording to data.pgr]\n")
		option.output = "data.pgr"
	}

	if option.disks == "" {
		option.targetDisks = nil
	} else {
		option.targetDisks = new(map[string]bool)
		*option.targetDisks = make(map[string]bool)
		for _, dev := range strings.Split(option.disks, ",") {
			(*option.targetDisks)[dev] = true
		}
	}

	if option.debug {
		os.Stderr.WriteString(
			fmt.Sprintf(
				`== option
  - output   : %s
  - interval : %s
  - debug    : %t
  - remainings: %s
`,
				option.output,
				option.interval.String(),
				option.debug,
				fmt.Sprint(flag.Args())))
	}
}

func main() {
	var enc *gob.Encoder
	var out *bufio.Writer
	var err error

	// Need to check '-background' before parsing args
	is_background := false
	for _, arg := range os.Args {
		if arg == "-background" {
			is_background = true
		}
	}

	if is_background {
		// Find existing session, or create new one
		user, err := user.Current()
		if err != nil {
			panic(err)
		}
		session_file := path.Join(os.TempDir(),
			fmt.Sprintf("perfmonger-%s-session.pid", user.Username))

		lockfile := path.Join(os.TempDir(), ".perfmonger.lock")

		// make lock file if not exists
		session_exists := false

		if _, err := os.Stat(lockfile); err != nil {
			ioutil.WriteFile(lockfile, []byte(""), 0644)
		}
		fd, _ := syscall.Open(lockfile, syscall.O_RDONLY, 0000)
		syscall.Flock(fd, syscall.LOCK_EX)

		if _, err := os.Stat(session_file); err == nil {
			pidstr, err := ioutil.ReadFile(session_file)
			pid, err := strconv.Atoi(string(pidstr))
			if err != nil {
				goto MakeNewSession
			}

			// check if PID in session file is valid
			proc, err := os.FindProcess(pid)
			err = proc.Signal(syscall.Signal(0))

			if err == nil {
				session_exists = true
				goto Unlock
			}
		}
	MakeNewSession:
		err = ioutil.WriteFile(session_file, []byte(strconv.Itoa(os.Getpid())), 0644)
		if err != nil {
			panic(err)
		}
		defer os.Remove(session_file)

	Unlock:
		syscall.Flock(fd, syscall.LOCK_UN)
		syscall.Close(fd)

		if session_exists {
			fmt.Fprintf(os.Stderr, "[ERROR] another perfmonger is already running in background mode\n")
			return
		}
	}

	parseArgs()

	hostname, _ := os.Hostname()
	cheader := &ss.CommonHeader{Platform: ss.Linux, Hostname: hostname, StartTime: time.Now()}

	platform_header := ss.NewPlatformHeader()

	if option.listDevices {
		for _, name := range platform_header.DevsParts {
			os.Stderr.WriteString(name + "\n")
		}
		return
	}

	var player_cmd *exec.Cmd = nil
	var player_stdin io.WriteCloser = nil
	var player_stdout io.ReadCloser = nil

	if option.player_bin != "" {
		if option.color {
			if option.pretty {
				player_cmd = exec.Command(option.player_bin, "-color", "-pretty")
			} else {
				player_cmd = exec.Command(option.player_bin, "-color")
			}
		} else {
			player_cmd = exec.Command(option.player_bin)
		}
		player_stdin, err = player_cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get stdin of %s", option.player_bin)
			player_cmd = nil
			player_stdin = nil
		}
		player_stdout, err = player_cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get stdout of %s", option.player_bin)
			player_cmd = nil
			player_stdin = nil
			player_stdout = nil
		}

		err = player_cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start %s", option.player_bin)
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

	if option.output == "-" {
		out = bufio.NewWriter(os.Stdout)
		if player_stdin != nil {
			out = bufio.NewWriter(player_stdin)
		}
	} else {
		file, err := os.Create(option.output)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		if player_stdin != nil {
			out = bufio.NewWriter(io.MultiWriter(file, player_stdin))
		} else {
			if option.gzip {
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
	time.Sleep(option.start_delay)

	var timeout_ch <-chan time.Time
	var timeout_time time.Time
	if option.timeout == time.Second*0 {
		// dummy channel
		timeout_ch = make(<-chan time.Time)
		timeout_time = time.Now()
	} else {
		timeout_ch = time.After(option.timeout)
		timeout_time = time.Now().Add(option.timeout)
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

		if !option.no_cpu {
			ss.ReadCpuStat(record)
		}
		if !option.no_intr {
			ss.ReadInterruptStat(record)
		}
		if !option.no_disk {
			ss.ReadDiskStats(record, option.targetDisks)
		}
		if !option.no_net {
			ss.ReadNetStat(record)
		}
		if !option.no_mem {
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

		if !option.no_interval_backoff {
			backoff_counter++
			if backoff_counter >= BACKOFF_THRESH {
				backoff_counter -= BACKOFF_THRESH

				option.interval *= BACKOFF_RATIO
				if option.interval.Seconds() > 3600.0 {
					option.interval = time.Hour
				}
			}
		}

		next_time = next_time.Add(option.interval)

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
		if option.timeout != time.Second*0 &&
			timeout_time.Sub(next_time).Seconds() < 0.01 {
			running = false
		}
	}

	out.Flush()

	if player_stdin != nil {
		player_stdin.Close()
		_ = player_cmd.Wait()
	}

	return
}
