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

func showDiskStat(buffer *bytes.Buffer, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) {
	dusage := ss.GetDiskUsage(
		prev_rec.Time, prev_rec.Disk,
		cur_rec.Time, cur_rec.Disk,
	)

	buffer.WriteString(`,"disk":`)
	dusage.WriteJsonTo(buffer)
}

func showCpuStat(buffer *bytes.Buffer, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) {
	cusage := ss.GetCpuUsage(prev_rec.Cpu, cur_rec.Cpu)
	buffer.WriteString(`,"cpu":`)
	cusage.WriteJsonTo(buffer)
}

func showStat(buffer *bytes.Buffer, prev_rec *ss.StatRecord, cur_rec *ss.StatRecord) {
	buffer.WriteString(fmt.Sprintf(`{"time":%.3f`, float64(cur_rec.Time.UnixNano())/1e9))
	showDiskStat(buffer, prev_rec, cur_rec)
	showCpuStat(buffer, prev_rec, cur_rec)
	buffer.WriteString("}\n")
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

		showStat(buffer, prev_rec, cur_rec)

		_, err = out.WriteString(buffer.String())
		err = out.Flush()
		if err != nil {
			// stdout is closed
			break
		}

		buffer.Reset()

		// fmt.Println("time: ", cur_rec.Time)
		// fmt.Println("cpu: ", cur_rec.Cpu)
		// fmt.Println("proc: ", cur_rec.Proc)
		// fmt.Println("disk: ", cur_rec.Disk)
		// fmt.Println("softirq: ", cur_rec.Softirq)

		curr ^= 1
	}

	os.Exit(0)
}
