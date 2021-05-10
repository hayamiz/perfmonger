//usr/bin/env go run $0 $@ ; exit

package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"

	projson "github.com/hayamiz/go-projson"
	ss "github.com/hayamiz/perfmonger/core/subsystem"
)

type SummaryOption struct {
	logfile         string
	title           string
	json            bool
	disk_only       string
	disk_only_regex *regexp.Regexp
}

var option SummaryOption

func parseArgs() {
	var err error

	flag.BoolVar(&option.json, "json",
		false, "Show summary in JSON")
	flag.StringVar(&option.title, "title",
		"", "Title of summary")
	flag.StringVar(&option.disk_only, "disk-only",
		"", "Select disk devices by regex")

	flag.Parse()

	if len(flag.Args()) < 1 {
		os.Exit(1)
	}

	option.disk_only_regex, err = regexp.Compile(option.disk_only)
	if err != nil {
		panic(err)
	}

	option.logfile = flag.Args()[0]
}

func main() {
	var cheader ss.CommonHeader
	var pheader ss.PlatformHeader

	parseArgs()

	f, err := os.Open(option.logfile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	input_reader := newPerfmongerLogReader(f)
	dec := gob.NewDecoder(input_reader)

	err = dec.Decode(&cheader)
	if err == io.EOF {
		return
	}
	if err != nil {
		panic(err)
	}
	err = dec.Decode(&pheader)
	if err == io.EOF {
		return
	}
	if err != nil {
		panic(err)
	}

	var fst_record ss.StatRecord
	// read first record
	err = dec.Decode(&fst_record)
	if err == io.EOF {
		return
	} else if err != nil {
		panic(err)
	}

	// loop until last line
	var lst_records [2]ss.StatRecord
	idx := 0
	for {
		err = dec.Decode(&lst_records[idx])
		if err == io.EOF {
			idx ^= 1
			break
		} else if err != nil {
			panic(err)
		}

		idx ^= 1
	}

	lst_record := lst_records[idx]

	var cpu_usage *ss.CpuUsage = nil
	var intr_usage *ss.InterruptUsage = nil
	var disk_usage *ss.DiskUsage = nil
	var net_usage *ss.NetUsage = nil

	if fst_record.Cpu != nil && lst_record.Cpu != nil {
		cpu_usage, err = ss.GetCpuUsage(fst_record.Cpu, lst_record.Cpu)
	}

	if fst_record.Interrupt != nil && lst_record.Interrupt != nil {
		intr_usage, err = ss.GetInterruptUsage(
			fst_record.Time, fst_record.Interrupt,
			lst_record.Time, lst_record.Interrupt,
		)
	}

	if fst_record.Disk != nil && lst_record.Disk != nil {
		disk_usage, err = ss.GetDiskUsage1(
			fst_record.Time, fst_record.Disk,
			lst_record.Time, lst_record.Disk,
			option.disk_only_regex)
	}

	if fst_record.Net != nil && lst_record.Net != nil {
		net_usage, err = ss.GetNetUsage(
			fst_record.Time, fst_record.Net,
			lst_record.Time, lst_record.Net)
	}

	interval := lst_record.Time.Sub(fst_record.Time)

	if option.json {
		printer := projson.NewPrinter()

		printer.BeginObject()
		printer.PutKey("exectime")
		printer.PutFloatFmt(interval.Seconds(), "%.3f")
		if cpu_usage != nil {
			printer.PutKey("cpu")
			cpu_usage.WriteJsonTo(printer)
		}

		if intr_usage != nil {
			printer.PutKey("intr")
			intr_usage.WriteJsonTo(printer)
		}

		if disk_usage != nil {
			printer.PutKey("disk")
			disk_usage.WriteJsonTo(printer)
		}

		if net_usage != nil {
			printer.PutKey("net")
			net_usage.WriteJsonTo(printer)
		}

		printer.FinishObject()

		if str, err := printer.String(); err != nil {
			fmt.Println("skip by err")
		} else {
			fmt.Println(str)
		}
	} else {
		if option.title == "" {
			fmt.Println("== performance summary ==")
		} else {
			fmt.Printf("== performance summary of '%s' ==\n", option.title)
		}
		fmt.Printf(`
Duration: %.3f sec

`,
			interval.Seconds())
		if cpu_usage != nil {
			fmt.Printf(`* Average CPU usage (MAX: %d %%)
  * Non-idle usage: %.2f %%
       %%usr: %.2f %%
       %%sys: %.2f %%
       %%irq: %.2f %%
      %%soft: %.2f %%
     %%other: %.2f %%
  * Idle usage: %.2f %%
    %%iowait: %.2f %%
      %%idle: %.2f %%

`,
				100*cpu_usage.NumCore,
				100.0*float64(cpu_usage.NumCore)-cpu_usage.All.Idle-cpu_usage.All.Iowait,
				cpu_usage.All.User+cpu_usage.All.Nice,
				cpu_usage.All.Sys,
				cpu_usage.All.Hardirq,
				cpu_usage.All.Softirq,
				cpu_usage.All.Steal,
				cpu_usage.All.Idle+cpu_usage.All.Iowait,
				cpu_usage.All.Iowait, cpu_usage.All.Idle)
		}

		if disk_usage != nil {
			devices := []string{}

			for device, _ := range *disk_usage {
				if device != "total" {
					devices = append(devices, device)
				}
			}
			sort.Strings(devices)
			if len(devices) > 1 {
				devices = append(devices, "total")
			}

			for _, device := range devices {
				e := (*disk_usage)[device]
				fmt.Printf(`* Average DEVICE usage: %s
        read IOPS: %.2f
       write IOPS: %.2f
  read throughput: %.2f MB/s
 write throughput: %.2f MB/s
     read latency: %.1f usec
    write latency: %.1f usec
      read amount: %.2f MB
     write amount: %.2f MB

`,
					device,
					e.RdIops, e.WrIops,
					e.RdSecps*512.0/1024.0/1024.0, e.WrSecps*512.0/1024.0/1024.0,
					e.RdLatency*1000.0, e.WrLatency*1000.0,
					float64(e.RdSectors*512)/1024.0/1024.0,
					float64(e.WrSectors*512)/1024.0/1024.0)
			}
		}
	}
}
