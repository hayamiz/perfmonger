package player

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"

	projson "github.com/hayamiz/go-projson"
	ss "github.com/hayamiz/perfmonger/core/internal/perfmonger"
)

// PlayerOption represents all options for the player component
// This struct is now public for direct use by cobra commands
type PlayerOption struct {
	Logfile       string
	Color         bool
	Pretty        bool
	DiskOnly      string
	DiskOnlyRegex *regexp.Regexp
}

var init_rec ss.StatRecord

func showCpuStat(printer *projson.JsonPrinter, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	cusage, err := ss.GetCpuUsage(prev_rec.Cpu, cur_rec.Cpu)
	if err != nil {
		return err
	}

	printer.PutKey("cpu")
	cusage.WriteJsonTo(printer)

	return nil
}

func showInterruptStat(printer *projson.JsonPrinter, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	intr_usage, err := ss.GetInterruptUsage(
		prev_rec.Time, prev_rec.Interrupt,
		cur_rec.Time, cur_rec.Interrupt)
	if err != nil {
		return err
	}

	printer.PutKey("intr")
	intr_usage.WriteJsonTo(printer)

	return nil
}

func showDiskStat(printer *projson.JsonPrinter, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord, disk_only_regex *regexp.Regexp, option *PlayerOption) error {
	dusage, err := ss.GetDiskUsage1(
		prev_rec.Time, prev_rec.Disk,
		cur_rec.Time, cur_rec.Disk,
		option.DiskOnlyRegex)
	if err != nil {
		return err
	}

	printer.PutKey("disk")

	dusage.WriteJsonTo(printer)

	return nil
}

func showNetStat(printer *projson.JsonPrinter, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	dusage, err := ss.GetNetUsage(
		prev_rec.Time, prev_rec.Net,
		cur_rec.Time, cur_rec.Net,
	)
	if err != nil {
		return err
	}

	printer.PutKey("net")

	dusage.WriteJsonTo(printer)

	return nil
}

func showMemStat(printer *projson.JsonPrinter, cur_rec *ss.StatRecord) error {
	musage, err := ss.GetMemUsage(cur_rec.Mem)
	if err != nil {
		return err
	}

	printer.PutKey("mem")

	musage.WriteJsonTo(printer)

	return nil
}

func showStat(printer *projson.JsonPrinter, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord,
	disk_only_regex *regexp.Regexp, option *PlayerOption) error {

	printer.Reset()
	if option.Pretty {
		printer.SetStyle(projson.SmartStyle)
	}
	if option.Color {
		printer.SetColor(true)
	}

	printer.BeginObject()
	printer.PutKey("time")
	printer.PutFloatFmt(float64(cur_rec.Time.UnixNano())/1e9, "%.3f")
	printer.PutKey("elapsed_time")
	printer.PutFloatFmt((float64(cur_rec.Time.UnixNano())-float64(init_rec.Time.UnixNano()))/1e9,
		"%.3f")

	if cur_rec.Cpu != nil {
		err := showCpuStat(printer, prev_rec, cur_rec)
		if err != nil {
			return err
		}
	}
	if cur_rec.Interrupt != nil {
		err := showInterruptStat(printer, prev_rec, cur_rec)
		if err != nil {
			return err
		}
	}
	if cur_rec.Disk != nil {
		err := showDiskStat(printer, prev_rec, cur_rec, disk_only_regex, option)
		if err != nil {
			return err
		}
	}
	if cur_rec.Net != nil {
		err := showNetStat(printer, prev_rec, cur_rec)
		if err != nil {
			return err
		}
	}
	if cur_rec.Mem != nil {
		err := showMemStat(printer, cur_rec)
		if err != nil {
			return err
		}
	}

	printer.FinishObject()

	return nil
}

func parseArgs(args []string, option *PlayerOption) {
	fs := flag.NewFlagSet("player", flag.ExitOnError)
	
	fs.BoolVar(&option.Color, "color", false, "Use colored JSON output")
	fs.BoolVar(&option.Pretty, "pretty", false, "Use human readable JSON output")
	fs.StringVar(&option.DiskOnly, "disk-only", "", "Select disk devices by regex")

	fs.Parse(args)

	option.DiskOnlyRegex = nil

	if option.DiskOnly != "" {
		var err error
		option.DiskOnlyRegex, err = regexp.Compile(option.DiskOnly)
		if err != nil {
			panic(err)
		}
	}

	if len(fs.Args()) < 1 {
		option.Logfile = "-"
	} else {
		option.Logfile = fs.Arg(0)
	}
}

// NewPlayerOption creates a PlayerOption with default values
func NewPlayerOption() *PlayerOption {
	return &PlayerOption{
		Logfile:       "-",
		Color:         false,
		Pretty:        false,
		DiskOnly:      "",
		DiskOnlyRegex: nil,
	}
}

// RunWithOption executes the player with the provided options
// This is the new preferred API that avoids double argument parsing
func RunWithOption(option *PlayerOption) {
	// Create command line arguments from the option struct
	args := make([]string, 0, 10)
	
	if option.Color {
		args = append(args, "-color")
	}
	if option.Pretty {
		args = append(args, "-pretty")
	}
	if option.DiskOnly != "" {
		args = append(args, "-disk-only", option.DiskOnly)
	}
	
	// Add logfile as positional argument if not stdin
	if option.Logfile != "-" {
		args = append(args, option.Logfile)
	}
	
	// Call the existing Run function with generated args
	Run(args)
}

func Run(args []string) {
	option := NewPlayerOption()
	var in *os.File
	var out *bufio.Writer

	parseArgs(args, option)

	if option.Logfile == "-" {
		in = os.Stdin
	} else {
		f, err := os.Open(option.Logfile)
		if err != nil {
			panic(err)
		}
		in = f
		defer f.Close()
	}
	input_reader := ss.NewPerfmongerLogReader(in)
	dec := gob.NewDecoder(input_reader)

	out = bufio.NewWriter(os.Stdout)

	var cheader ss.CommonHeader
	var pheader ss.PlatformHeader
	var records = make([]ss.StatRecord, 2)
	curr := 0

	var err error

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

	// read first record
	err = dec.Decode(&records[curr])
	if err == io.EOF {
		return
	} else if err != nil {
		panic(err)
	}
	init_rec = records[curr]
	curr ^= 1

	printer := projson.NewPrinter()
	for {
		prev_rec := &records[curr^1]
		cur_rec := &records[curr]

		err = dec.Decode(cur_rec)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		err = showStat(printer, prev_rec, cur_rec, option.DiskOnlyRegex, option)
		if err != nil {
			printer.Reset()
			fmt.Fprintln(os.Stderr, "skip by err")
			continue
		}

		if str, err := printer.String(); err != nil {
			fmt.Println("error", err)
			fmt.Println(str)
		} else {
			_, err = out.WriteString(str + "\n")
		}
		err = out.Flush()
		if err != nil {
			// stdout is closed
			break
		}

		printer.Reset()

		curr ^= 1
	}

	return
}