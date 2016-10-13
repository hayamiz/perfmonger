//usr/bin/env go run $0 $@ ; exit

package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"

	ss "github.com/hayamiz/perfmonger/core/subsystem"
)

func showCpuStat(buffer *bytes.Buffer, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	cusage, err := ss.GetCpuUsage(prev_rec.Cpu, cur_rec.Cpu)
	if err != nil {
		return err
	}
	buffer.WriteString(`,"cpu":`)
	cusage.WriteJsonTo(buffer)

	return nil
}

func showDiskStat(buffer *bytes.Buffer, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	dusage, err := ss.GetDiskUsage(
		prev_rec.Time, prev_rec.Disk,
		cur_rec.Time, cur_rec.Disk)
	if err != nil {
		return err
	}

	buffer.WriteString(`,"disk":`)
	dusage.WriteJsonTo(buffer)

	return nil
}

func showNetStat(buffer *bytes.Buffer, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	dusage, err := ss.GetNetUsage(
		prev_rec.Time, prev_rec.Net,
		cur_rec.Time, cur_rec.Net,
	)
	if err != nil {
		return err
	}

	buffer.WriteString(`,"net":`)
	dusage.WriteJsonTo(buffer)

	return nil
}

func showStat(buffer *bytes.Buffer, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) error {
	buffer.WriteString(fmt.Sprintf(`{"time":%.3f`, float64(cur_rec.Time.UnixNano())/1e9))
	if cur_rec.Cpu != nil {
		err := showCpuStat(buffer, prev_rec, cur_rec)
		if err != nil {
			return err
		}
	}
	if cur_rec.Disk != nil {
		err := showDiskStat(buffer, prev_rec, cur_rec)
		if err != nil {
			return err
		}
	}
	if cur_rec.Net != nil {
		err := showNetStat(buffer, prev_rec, cur_rec)
		if err != nil {
			return err
		}
	}
	buffer.WriteString("}\n")

	return nil
}

func main() {
	args := os.Args
	var in *os.File
	var out *bufio.Writer

	if len(args) < 2 {
		in = os.Stdin
	} else {
		f, err := os.Open(args[1])
		if err != nil {
			panic(err)
		}
		in = f
		defer f.Close()
	}

	out = bufio.NewWriter(os.Stdout)

	dec := gob.NewDecoder(in)

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
	curr ^= 1

	buffer := bytes.NewBuffer([]byte{})
	for {
		prev_rec := &records[curr^1]
		cur_rec := &records[curr]

		err = dec.Decode(cur_rec)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		err = showStat(buffer, prev_rec, cur_rec)
		if err != nil {
			buffer.Reset()
			fmt.Fprintln(os.Stderr, "skip by err")
			continue
		}

		_, err = out.WriteString(buffer.String())
		err = out.Flush()
		if err != nil {
			// stdout is closed
			break
		}

		buffer.Reset()

		curr ^= 1
	}

	return
}
