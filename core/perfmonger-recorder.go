//usr/bin/env go run $0 $@ ; exit

package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	ss "github.com/hayamiz/perfmonger/core/subsystem"
)

type RecorderOption struct {
	interval    time.Duration
	devsParts   []string
	output      string
	debug       bool
	listDevices bool
}

var option RecorderOption

func readStat(record *ss.StatRecord) error {
	f, ferr := os.Open("/proc/stat")
	if ferr != nil {
		return ferr
	}
	defer f.Close()

	// 0-fill StatRecord
	record.Cpu.Clear()
	record.Proc.Clear()

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		var err error
		var cpu string
		line := scan.Text()
		if line[0:4] == "cpu " {
			_, err = fmt.Sscanf(line,
				"%s %d %d %d %d %d %d %d %d %d %d",
				&cpu,
				&record.Cpu.All.User,
				&record.Cpu.All.Nice,
				&record.Cpu.All.Sys,
				&record.Cpu.All.Idle,
				&record.Cpu.All.Iowait,
				&record.Cpu.All.Hardirq,
				&record.Cpu.All.Softirq,
				&record.Cpu.All.Steal,
				&record.Cpu.All.Guest,
				&record.Cpu.All.GuestNice)
			if err != nil {
				return err
			}
		} else if line[0:3] == "cpu" {
			var n_core int
			var core_stat *ss.CpuCoreStat
			// assume n_core < 10000
			_, err = fmt.Sscanf(line[3:7], "%d", &n_core)
			if err != nil {
				return err
			}

			core_stat = &record.Cpu.CoreStats[n_core]
			_, err = fmt.Sscanf(line,
				"%s %d %d %d %d %d %d %d %d %d %d",
				&cpu,
				&core_stat.User,
				&core_stat.Nice,
				&core_stat.Sys,
				&core_stat.Idle,
				&core_stat.Iowait,
				&core_stat.Hardirq,
				&core_stat.Softirq,
				&core_stat.Steal,
				&core_stat.Guest,
				&core_stat.GuestNice)
			if err != nil {
				return err
			}
		} else if line[0:5] == "ctxt " {
			_, err = fmt.Sscanf(line[4:], "%d", &record.Proc.ContextSwitch)
			if err != nil {
				return err
			}
		} else if line[0:10] == "processes " {
			_, err = fmt.Sscanf(line[10:], "%d", &record.Proc.Fork)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func readDiskStats(record *ss.StatRecord) error {
	f, ferr := os.Open("/proc/diskstats")
	if ferr != nil {
		panic(ferr)
	}
	defer f.Close()

	record.Disk.Clear()

	scan := bufio.NewScanner(f)

	var num_items int
	var err error
	for scan.Scan() {
		var rdmerge_or_rdsec int64
		var rdsec_or_wrios int64
		var rdticks_or_wrsec int64

		line := scan.Text()
		entry := ss.NewDiskStatEntry()

		num_items, err = fmt.Sscanf(line,
			"%d %d %s %d %d %d %d %d %d %d %d %d %d %d",
			&entry.Major, &entry.Minor, &entry.Name,
			&entry.RdIos, &rdmerge_or_rdsec, &rdsec_or_wrios, &rdticks_or_wrsec,
			&entry.WrIos, &entry.WrMerges, &entry.WrSectors, &entry.WrTicks,
			&entry.IosPgr, &entry.TotalTicks, &entry.ReqTicks)
		if err != nil {
			return err
		}

		if num_items == 14 {
			entry.RdMerges = rdmerge_or_rdsec
			entry.RdSectors = rdsec_or_wrios
			entry.RdTicks = rdticks_or_wrsec
		} else if num_items == 7 {
			entry.RdSectors = rdmerge_or_rdsec
			entry.WrIos = rdsec_or_wrios
			entry.WrSectors = rdticks_or_wrsec
		} else {
			continue
		}

		if entry.RdIos == 0 && entry.WrIos == 0 {
			continue
		}

		record.Disk.Entries = append(record.Disk.Entries, entry)
	}

	return nil
}

func parseArgs() {
	// set options
	flag.DurationVar(&option.interval, "interval",
		time.Second, "Measurement interval")
	flag.StringVar(&option.output, "output",
		"-", "Output file name")
	flag.BoolVar(&option.debug, "debug",
		false, "Enable debug mode")
	flag.BoolVar(&option.listDevices, "list-devices",
		false, "List devices and exits")

	flag.Parse()

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

	parseArgs()

	hostname, _ := os.Hostname()
	cheader := &ss.CommonHeader{ss.Linux, hostname, time.Now()}

	platform_header := ss.NewPlatformHeader()

	if option.listDevices {
		for _, name := range platform_header.DevsParts {
			os.Stderr.WriteString(name + "\n")
		}
		return
	}

	if option.output == "-" {
		out = bufio.NewWriter(os.Stdout)
	} else {
		file, err := os.Create(option.output)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		out = bufio.NewWriter(file)
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

	// Loops
	sigint_ch := make(chan os.Signal, 1)
	running := true
	next_time := time.Now()
	record := ss.NewStatRecord()

	// cause SIGINT to break loop
	signal.Notify(sigint_ch, os.Interrupt)

	for {
		record.Time = time.Now()
		readStat(record)
		readDiskStats(record)

		err = enc.Encode(record)
		out.Flush()
		if err != nil {
			break
		}

		if !running {
			break
		}

		next_time = next_time.Add(option.interval)
		// wait for next iteration
		select {
		case <-sigint_ch:
			running = false
			break
		case <-time.After(next_time.Sub(time.Now())):
			break
		}
	}

	os.Exit(0)
}
