package summarizer

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"

	projson "github.com/hayamiz/go-projson"
	ss "github.com/hayamiz/perfmonger/core/internal/perfmonger"
)

// SummaryOption represents all options for the summarizer component
// This struct is now public for direct use by cobra commands
type SummaryOption struct {
	Logfile       string
	Title         string
	JSON          bool
	DiskOnly      string
	DiskOnlyRegex *regexp.Regexp
}

func parseArgs(args []string, option *SummaryOption) {
	var err error

	fs := flag.NewFlagSet("summarizer", flag.ExitOnError)
	
	fs.BoolVar(&option.JSON, "json",
		false, "Show summary in JSON")
	fs.StringVar(&option.Title, "title",
		"", "Title of summary")
	fs.StringVar(&option.DiskOnly, "disk-only",
		"", "Select disk devices by regex")

	fs.Parse(args)

	if len(fs.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "Insufficient argument")
		os.Exit(1)
	}

	option.DiskOnlyRegex, err = regexp.Compile(option.DiskOnly)
	if err != nil {
		panic(err)
	}

	option.Logfile = fs.Args()[0]
}

// NewSummaryOption creates a SummaryOption with default values
func NewSummaryOption() *SummaryOption {
	return &SummaryOption{
		Logfile:       "",
		Title:         "",
		JSON:          false,
		DiskOnly:      "",
		DiskOnlyRegex: nil,
	}
}

// RunWithOption executes the summarizer with the provided options
// This is the new preferred API that avoids double argument parsing
func RunWithOption(option *SummaryOption) {
	// Create command line arguments from the option struct
	args := make([]string, 0, 10)
	
	if option.JSON {
		args = append(args, "-json")
	}
	if option.Title != "" {
		args = append(args, "-title", option.Title)
	}
	if option.DiskOnly != "" {
		args = append(args, "-disk-only", option.DiskOnly)
	}
	
	// Add logfile as positional argument
	if option.Logfile != "" {
		args = append(args, option.Logfile)
	}
	
	// Call the existing Run function with generated args
	Run(args)
}

func Run(args []string) {
	option := NewSummaryOption()
	var cheader ss.CommonHeader
	var pheader ss.PlatformHeader

	parseArgs(args, option)

	f, err := os.Open(option.Logfile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	input_reader := ss.NewPerfmongerLogReader(f)
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
			option.DiskOnlyRegex)
	}

	if fst_record.Net != nil && lst_record.Net != nil {
		net_usage, err = ss.GetNetUsage(
			fst_record.Time, fst_record.Net,
			lst_record.Time, lst_record.Net)
	}

	interval := lst_record.Time.Sub(fst_record.Time)

	if option.JSON {
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
		if option.Title == "" {
			fmt.Println("== performance summary ==")
		} else {
			fmt.Printf("== performance summary of '%s' ==\n", option.Title)
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