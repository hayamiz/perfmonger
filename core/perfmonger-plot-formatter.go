//usr/bin/env go run $0 $@ ; exit

package main

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	ss "github.com/hayamiz/perfmonger/core/subsystem"
)

type CmdOption struct {
	DiskFile       string
	CpuFile        string
	PerfmongerFile string
}

func parseArgs() *CmdOption {
	opt := new(CmdOption)

	flag.StringVar(&opt.DiskFile, "diskfile", "./disk.dat", "Disk performance data file")
	flag.StringVar(&opt.CpuFile, "cpufile", "./cpu.dat", "CPU performance data file")
	flag.StringVar(&opt.PerfmongerFile, "perfmonger", "", "Perfmonger log file")

	flag.Parse()

	if opt.PerfmongerFile == "" {
		os.Stderr.WriteString("[ERROR] perfmonger log file is required.\n")
		os.Exit(1)
	}

	return opt
}

type DiskMetaEntry struct {
	Name string `json:"name"`
	Idx  int    `json:"idx"`
}

type DiskMeta struct {
	Devices []DiskMetaEntry `json:"devices"`
}

type CpuMeta struct {
	NumCore int `json:"num_core"`
}

type PlotMeta struct {
	Disk DiskMeta `json:"disk"`
	Cpu  CpuMeta  `json:"cpu"`
}

type DiskDatTmpFile struct {
	Name   string
	Path   string
	File   *os.File
	Writer *bufio.Writer
	Idx    int
}

type CpuDatTmpFile struct {
	CoreId int
	Path   string
	File   *os.File
	Writer *bufio.Writer
}

func makeDiskDatTmpFile(dname string, idx int) *DiskDatTmpFile {
	ret := new(DiskDatTmpFile)
	ret.Name = dname
	ret.Idx = idx

	f, err := ioutil.TempFile("", "perfmonger-"+dname)
	if err != nil {
		panic(err)
	}
	ret.File = f
	ret.Path = f.Name()
	ret.Writer = bufio.NewWriter(f)

	return ret
}

func makeCpuDatTmpFile(coreid int) *CpuDatTmpFile {
	ret := new(CpuDatTmpFile)
	ret.CoreId = coreid

	f, err := ioutil.TempFile("", fmt.Sprintf("perfmonger-core%d-", coreid))
	if err != nil {
		panic(err)
	}
	ret.File = f
	ret.Path = f.Name()
	ret.Writer = bufio.NewWriter(f)

	return ret
}

func printCoreUsage(writer *bufio.Writer, elapsed_time float64, coreusage *ss.CpuCoreUsage) {
	writer.WriteString(
		fmt.Sprintf("%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\n",
			elapsed_time,
			coreusage.User,
			coreusage.Nice,
			coreusage.Sys,
			coreusage.Iowait,
			coreusage.Hardirq,
			coreusage.Softirq,
			coreusage.Steal,
			coreusage.Guest,
			coreusage.Idle))
}

func main() {
	opt := parseArgs()

	var in *os.File

	f, err := os.Open(opt.PerfmongerFile)
	if err != nil {
		panic(err)
	}
	in = f
	defer f.Close()

	dec := gob.NewDecoder(bufio.NewReader(in))

	var cheader ss.CommonHeader
	var pheader ss.PlatformHeader
	var records = make([]ss.StatRecord, 2)
	curr := 0

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
	t0 := records[curr].Time
	curr ^= 1

	meta := PlotMeta{}
	meta_set := false

	disk_dat_files := map[string]*DiskDatTmpFile{}
	cpu_dat_files := make([]*CpuDatTmpFile, records[0].Cpu.NumCore)
	meta.Cpu.NumCore = records[0].Cpu.NumCore

	f, err = os.Create(opt.CpuFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cpu_writer := bufio.NewWriter(f)

	cpu_writer.WriteString("# All cpu usage\n")
	cpu_writer.WriteString("# elapsed_time	%usr	%nice	%sys	%iowait	%hardirq	%softirq	%steal	%guest	%idle\n")

	for {
		prev_rec := &records[curr^1]
		cur_rec := &records[curr]

		err := dec.Decode(cur_rec)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		// Disk usage
		dusage, err := ss.GetDiskUsage(prev_rec.Time, prev_rec.Disk, cur_rec.Time, cur_rec.Disk)
		if err != nil {
			panic(err)
		}
		didx := 0

		for dname, dusage_entry := range *dusage {
			if !meta_set {
				meta.Disk.Devices =
					append(meta.Disk.Devices,
						DiskMetaEntry{Name: dname, Idx: didx})
			}

			disk_dat, ok := disk_dat_files[dname]
			if !ok {
				disk_dat = makeDiskDatTmpFile(dname, didx)
				disk_dat_files[dname] = disk_dat
				defer disk_dat.File.Close()

				disk_dat.Writer.WriteString("\n\n\n")
				disk_dat.Writer.WriteString("# device: " + disk_dat.Name + "\n")
				disk_dat.Writer.WriteString(fmt.Sprintln(
					"# elapsed_time	r_iops	w_iops	r_MB/s	w_MB/s	r_latency	w_latency	r_avgsz	w_avgsz	qdepth"))
			}

			elapsed_time := prev_rec.Time.Sub(t0).Seconds()
			disk_dat.Writer.WriteString(
				fmt.Sprintf("%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\t%f\n",
					elapsed_time,
					dusage_entry.RdIops,
					dusage_entry.WrIops,
					dusage_entry.RdSecps*512.0/1024.0/1024.0,
					dusage_entry.WrSecps*512.0/1024.0/1024.0,
					dusage_entry.RdLatency,
					dusage_entry.WrLatency,
					dusage_entry.AvgRdSize,
					dusage_entry.AvgWrSize,
					dusage_entry.ReqQlen))

			didx += 1
		}

		// Cpu usage
		cusage, err := ss.GetCpuUsage(prev_rec.Cpu, cur_rec.Cpu)
		if err != nil {
			panic(err)
		}
		for coreid, coreusage := range cusage.CoreUsages {
			cpu_dat := cpu_dat_files[coreid]
			if cpu_dat == nil {
				cpu_dat = makeCpuDatTmpFile(coreid)
				cpu_dat_files[coreid] = cpu_dat
				defer cpu_dat.File.Close()

				cpu_dat.Writer.WriteString(fmt.Sprintf("\n\n\n# core: %d\n", coreid))
				cpu_dat.Writer.WriteString("# elapsed_time	%usr	%nice	%sys	%iowait	%hardirq	%softirq	%steal	%guest	%idle\n")
			}

			printCoreUsage(cpu_dat.Writer, prev_rec.Time.Sub(t0).Seconds(), coreusage)
		}
		printCoreUsage(cpu_writer, cur_rec.Time.Sub(t0).Seconds(), cusage.All)

		curr ^= 1
		meta_set = true
	}

	for _, disk_dat := range disk_dat_files {
		disk_dat.Writer.Flush()
		disk_dat.File.Close()
	}

	f, err = os.Create(opt.DiskFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	df_writer := bufio.NewWriter(f)

	for _, dev := range meta.Disk.Devices {
		disk_dat, ok := disk_dat_files[dev.Name]
		if !ok {
			panic(dev.Name)
		}

		content, err := ioutil.ReadFile(disk_dat.Path)
		if err != nil {
			panic(err)
		}

		df_writer.Write(content)

		os.Remove(disk_dat.Path)
	}
	df_writer.Flush()

	for _, cpu_dat := range cpu_dat_files {
		cpu_dat.Writer.Flush()
		cpu_dat.File.Close()

		content, err := ioutil.ReadFile(cpu_dat.Path)
		if err != nil {
			panic(err)
		}

		cpu_writer.Write(content)
		os.Remove(cpu_dat.Path)
	}
	cpu_writer.Flush()

	json_enc := json.NewEncoder(os.Stdout)
	json_enc.Encode(meta)
}
