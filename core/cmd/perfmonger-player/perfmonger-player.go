//usr/bin/env go run $0 $@ ; exit

package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"os"

	projson "github.com/hayamiz/go-projson"
	ss "github.com/hayamiz/perfmonger/core/subsystem"
    core "github.com/hayamiz/perfmonger/core"
)

type PlayerOption struct {
	logfile string
	color   bool
	pretty  bool
}

var option PlayerOption
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

func showDiskStat(printer *projson.JsonPrinter, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	dusage, err := ss.GetDiskUsage(
		prev_rec.Time, prev_rec.Disk,
		cur_rec.Time, cur_rec.Disk)
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

func showStat(printer *projson.JsonPrinter, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	printer.Reset()
	if option.pretty {
		printer.SetStyle(projson.SmartStyle)
	}
	if option.color {
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
		err := showDiskStat(printer, prev_rec, cur_rec)
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

func parseArgs() {
	flag.BoolVar(&option.color, "color", false, "Use colored JSON output")
	flag.BoolVar(&option.pretty, "pretty", false, "Use human readable JSON output")

	flag.Parse()

	option.logfile = flag.Arg(0)
}

func main() {
	var in *os.File
	var out *bufio.Writer

	parseArgs()

	if option.logfile == "" {
		in = os.Stdin
	} else {
		f, err := os.Open(option.logfile)
		if err != nil {
			panic(err)
		}
		in = f
		defer f.Close()
	}
	input_reader := core.NewPerfmongerLogReader(in)
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

		err = showStat(printer, prev_rec, cur_rec)
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
