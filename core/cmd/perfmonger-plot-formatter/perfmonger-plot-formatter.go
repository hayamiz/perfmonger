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
	"regexp"
	"sort"
	"strings"

	"github.com/hayamiz/perfmonger/core"
	ss "github.com/hayamiz/perfmonger/core/subsystem"
)

type CmdOption struct {
	DiskFile        string
	CpuFile         string
	MemFile         string
	PerfmongerFile  string
	disk_only       string
	disk_only_regex *regexp.Regexp
}

func parseArgs() *CmdOption {
	var err error

	opt := new(CmdOption)

	flag.StringVar(&opt.DiskFile, "diskfile", "./disk.dat", "Disk usage data file for gnuplot")
	flag.StringVar(&opt.CpuFile, "cpufile", "./cpu.dat", "CPU usage data file for gnuplot")
	flag.StringVar(&opt.MemFile, "memfile", "./mem.dat", "Memory usage data file for gnuplot")
	flag.StringVar(&opt.PerfmongerFile, "perfmonger", "", "Perfmonger log file")
	flag.StringVar(&opt.disk_only, "disk-only",
		"", "Select disk devices by regex")

	flag.Parse()

	if opt.PerfmongerFile == "" {
		os.Stderr.WriteString("[ERROR] perfmonger log file is required.\n")
		os.Exit(1)
	}

	opt.disk_only_regex, err = regexp.Compile(opt.disk_only)
	if err != nil {
		panic(err)
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
	Disk      DiskMeta `json:"disk"`
	Cpu       CpuMeta  `json:"cpu"`
	StartTime float64  `json:"start_time"`
	EndTime   float64  `json:"end_time"`
}

type DiskDatTmpFile struct {
	Name   string
	Path   string
	File   *os.File
	Writer *bufio.Writer
	Idx    int
}

type MemDatTmpFile struct {
	Name   string
	Path   string
	File   *os.File
	Writer *bufio.Writer
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

func makeMemDatTmpFile() *MemDatTmpFile {
	ret := new(MemDatTmpFile)

	f, err := ioutil.TempFile("", "perfmonger-mem")
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

func printMemUsage(writer *bufio.Writer, elapsed_time float64, mem *ss.MemStat) {
	if mem == nil {
		writer.WriteString("#")
		writer.WriteString(
			strings.Join([]string{
				"elapsed_time",     // 1
				"mem_total",        // 2
				"mem_used",         // 3
				"mem_free",         // 4
				"buffers",          // 5
				"cached",           // 6
				"swap_cached",      // 7
				"active",           // 8
				"inactive",         // 9
				"swap_total",       // 10
				"swap_free",        // 11
				"dirty",            // 12
				"writeback",        // 13
				"anon_pages",       // 14
				"mapped",           // 15
				"shmem",            // 16
				"slab",             // 17
				"s_reclaimable",    // 18
				"s_unreclaim",      // 19
				"kernel_stack",     // 20
				"page_tables",      // 21
				"nfs_unstable",     // 22
				"bounce",           // 23
				"commit_limit",     // 24
				"committed_as",     // 25
				"anon_huge_pages",  // 26
				"huge_pages_total", // 27
				"huge_pages_free",  // 28
				"huge_pages_rsvd",  // 29
				"huge_pages_surp",  // 30
				"hugepagesize"},    // 31
				"\t"))
		writer.WriteString("\n")
	} else {
		writer.WriteString(fmt.Sprintf("%f\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",
			elapsed_time,
			mem.MemTotal,
			mem.MemTotal-mem.MemFree-mem.Buffers-mem.Cached-mem.SReclaimable,
			mem.MemFree,
			mem.Buffers,
			mem.Cached,
			mem.SwapCached,
			mem.Active,
			mem.Inactive,
			mem.SwapTotal,
			mem.SwapFree,
			mem.Dirty,
			mem.Writeback,
			mem.AnonPages,
			mem.Mapped,
			mem.Shmem,
			mem.Slab,
			mem.SReclaimable,
			mem.SUnreclaim,
			mem.KernelStack,
			mem.PageTables,
			mem.NFS_Unstable,
			mem.Bounce,
			mem.CommitLimit,
			mem.Committed_AS,
			mem.AnonHugePages,
			mem.HugePages_Total,
			mem.HugePages_Free,
			mem.HugePages_Rsvd,
			mem.HugePages_Surp,
			mem.Hugepagesize))
	}
}

func main() {
	opt := parseArgs()

	f, err := os.Open(opt.PerfmongerFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	input_reader := core.NewPerfmongerLogReader(f)
	dec := gob.NewDecoder(input_reader)

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

	meta_set := false
	meta := PlotMeta{}
	meta.StartTime = float64(records[0].Time.UnixNano()) / 1.0e9

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

	f, err = os.Create(opt.MemFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	mem_writer := bufio.NewWriter(f)

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
		dusage, err := ss.GetDiskUsage1(prev_rec.Time, prev_rec.Disk,
			cur_rec.Time, cur_rec.Disk,
			opt.disk_only_regex)
		if err != nil {
			panic(err)
		}
		didx := 0

		var dnames []string
		for dname, _ := range *dusage {
			if dname != "total" {
				dnames = append(dnames, dname)
			}
		}
		sort.Strings(dnames)
		dnames = append(dnames, "total")

		// for dname, dusage_entry := range *dusage {
		for _, dname := range dnames {
			dusage_entry, ok := (*dusage)[dname]
			if !ok {
				panic("device '" + dname + "' not found")
			}

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
		printCoreUsage(cpu_writer, prev_rec.Time.Sub(t0).Seconds(), cusage.All)

		if !meta_set {
			// print column labels
			printMemUsage(mem_writer, prev_rec.Time.Sub(t0).Seconds(), nil)
		}
		printMemUsage(mem_writer, prev_rec.Time.Sub(t0).Seconds(), cur_rec.Mem)

		curr ^= 1
		meta_set = true
	}

	meta.EndTime = float64(records[curr^1].Time.UnixNano()) / 1.0e9

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
	mem_writer.Flush()

	json_enc := json.NewEncoder(os.Stdout)
	json_enc.Encode(meta)
}
