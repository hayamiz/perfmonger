// +build linux

package subsystem

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type PlatformHeader LinuxHeader

func NewPlatformHeader() *LinuxHeader {
	header := new(LinuxHeader)
	header.Devices = make(map[string]LinuxDevice)

	header.getDevsParts()

	return header
}

func (header *LinuxHeader) getDevsParts() {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		var major, minor int
		var name string
		c, err := fmt.Sscanf(scan.Text(), "%d %d %s", &major, &minor, &name)
		if err != nil {
			panic(err)
		}
		if c != 3 {
			continue
		}

		header.DevsParts = append(header.DevsParts, name)

		if isDevice(name) {
			header.Devices[name] = LinuxDevice{
				name, getPartitions(name),
			}
		}
	}
}

func isDevice(name string) bool {
	stat, err := os.Stat(fmt.Sprintf("/sys/block/%s", name))
	if err == nil && stat.IsDir() {
		return true
	}

	return false
}

func getPartitions(name string) []string {
	var dir *os.File
	var fis []os.FileInfo
	var err error
	var parts = []string{}

	dir, err = os.Open(fmt.Sprintf("/sys/block/%s", name))
	if err != nil {
		panic(err)
	}
	fis, err = dir.Readdir(0)
	if err != nil {
		panic(err)
	}
	for _, fi := range fis {
		_, err := os.Stat(fmt.Sprintf("/sys/block/%s/%s/stat", name, fi.Name()))
		if err == nil {
			// partition exists
			parts = append(parts, fi.Name())
		}
	}

	return parts
}

func ReadCpuStat(record *StatRecord) error {
	f, ferr := os.Open("/proc/stat")
	if ferr != nil {
		return ferr
	}
	defer f.Close()

	if record.Cpu == nil {
		num_core := 0
		out, err := exec.Command("nproc", "--all").Output()
		out_str := strings.TrimSpace(string(out))

		if err == nil {
			num_core, err = strconv.Atoi(out_str)

			if err != nil {
				num_core = 0
			}
		}

		if num_core == 0 {
			num_core = runtime.NumCPU()
		}

		record.Cpu = NewCpuStat(num_core)
	} else {
		record.Cpu.Clear()
	}

	if record.Proc == nil {
		record.Proc = NewProcStat()
	} else {
		record.Proc.Clear()
	}

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		var err error
		var cpu string
		line := scan.Text()
		if line[0:4] == "cpu " {
			// Linux 2.6.33 or later
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
			if err == io.EOF {
				// Linux 2.6.24 or later
				_, err = fmt.Sscanf(line,
					"%s %d %d %d %d %d %d %d %d %d",
					&cpu,
					&record.Cpu.All.User,
					&record.Cpu.All.Nice,
					&record.Cpu.All.Sys,
					&record.Cpu.All.Idle,
					&record.Cpu.All.Iowait,
					&record.Cpu.All.Hardirq,
					&record.Cpu.All.Softirq,
					&record.Cpu.All.Steal,
					&record.Cpu.All.Guest)
				record.Cpu.All.GuestNice = 0
			}
			if err != nil {
				panic(err)
			}
		} else if line[0:3] == "cpu" {
			var n_core int
			var core_stat *CpuCoreStat
			// assume n_core < 10000
			_, err = fmt.Sscanf(line[3:7], "%d", &n_core)
			if err != nil {
				panic(err)
			}

			core_stat = &record.Cpu.CoreStats[n_core]
			// Linux 2.6.33 or later
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
			if err == io.EOF {
				// Linux 2.6.24 or later
				_, err = fmt.Sscanf(line,
					"%s %d %d %d %d %d %d %d %d %d",
					&cpu,
					&core_stat.User,
					&core_stat.Nice,
					&core_stat.Sys,
					&core_stat.Idle,
					&core_stat.Iowait,
					&core_stat.Hardirq,
					&core_stat.Softirq,
					&core_stat.Steal,
					&core_stat.Guest)
			}
			if err != nil {
				panic(err)
			}
		} else if line[0:5] == "ctxt " {
			_, err = fmt.Sscanf(line[4:], "%d", &record.Proc.ContextSwitch)
			if err != nil {
				panic(err)
			}
		} else if line[0:10] == "processes " {
			_, err = fmt.Sscanf(line[10:], "%d", &record.Proc.Fork)
			if err != nil {
				panic(err)
			}
		}
	}

	return nil
}

func parseInterruptStatEntry(line string, num_core int) (*InterruptStatEntry, error) {
	entry := new(InterruptStatEntry)

	entry.NumCore = num_core
	entry.IntrCounts = make([]int, num_core)

	tokens := strings.Fields(line)

	idx := 0

	tok := tokens[0]
	tok = strings.TrimRight(tok, ":")
	if irqno, err := strconv.Atoi(tok); err == nil {
		entry.IrqNo = irqno
		entry.IrqType = ""
	} else {
		entry.IrqNo = -1
		entry.IrqType = tok
	}

	for idx := 1; idx < num_core+1; idx += 1 {
		var c int
		var err error

		if idx >= len(tokens) {
			break
		}

		tok = tokens[idx]
		if c, err = strconv.Atoi(tok); err != nil {
			return nil, errors.New("Invalid string for IntrCounts element: " + tok)
		}

		entry.IntrCounts[idx-1] = c
	}

	idx = num_core + 1
	if idx < len(tokens) {
		entry.Descr = strings.Join(tokens[idx:], " ")
	} else {
		entry.Descr = ""
	}

	return entry, nil
}

func ReadInterruptStat(record *StatRecord) error {
	intr_stat := NewInterruptStat()

	if record == nil {
		return errors.New("Valid *StatRecord is required.")
	}

	f, err := os.Open("/proc/interrupts")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scan := bufio.NewScanner(f)

	if !scan.Scan() {
		return errors.New("/proc/interrupts seems to be empty")
	}

	cores := strings.Fields(scan.Text())
	num_core := len(cores)

	for scan.Scan() {
		entry, err := parseInterruptStatEntry(scan.Text(), num_core)

		if err != nil {
			return err
		}

		intr_stat.Entries = append(intr_stat.Entries, entry)
		intr_stat.NumEntries += 1
	}

	record.Interrupt = intr_stat

	return nil
}

func ReadDiskStats(record *StatRecord, targets *map[string]bool) error {
	if record == nil {
		return errors.New("Valid *StatRecord is required.")
	}

	f, ferr := os.Open("/proc/diskstats")
	if ferr != nil {
		panic(ferr)
	}
	defer f.Close()

	if record.Disk == nil {
		record.Disk = NewDiskStat()
	} else {
		record.Disk.Clear()
	}

	scan := bufio.NewScanner(f)

	var num_items int
	var err error
	for scan.Scan() {
		var rdmerge_or_rdsec int64
		var rdsec_or_wrios int64
		var rdticks_or_wrsec int64

		line := scan.Text()
		entry := NewDiskStatEntry()

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

		if targets != nil {
			if _, ok := (*targets)[entry.Name]; !ok {
				// device not in targets
				continue
			}
		} else {
			if !isDevice(entry.Name) {
				continue
			}
		}

		record.Disk.Entries = append(record.Disk.Entries, entry)
	}

	return nil
}

func ReadNetStat(record *StatRecord) error {
	if record == nil {
		return errors.New("Valid *StatRecord is required.")
	}

	net_stat := NewNetStat()

	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line[0:7] == "Inter-|":
			continue
		case line[0:7] == " face |":
			continue
		}
		line = strings.Replace(line, ":", " ", -1)

		e := NewNetStatEntry()

		var devname string
		n, err := fmt.Sscanf(line,
			"%s %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d %d", &devname,
			&e.RxBytes, &e.RxPackets, &e.RxErrors, &e.RxDrops,
			&e.RxFifo, &e.RxFrame, &e.RxCompressed, &e.RxMulticast,
			&e.TxBytes, &e.TxPackets, &e.TxErrors, &e.TxDrops,
			&e.TxFifo, &e.TxFrame, &e.TxCompressed, &e.TxMulticast)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if n != 17 {
			continue
		}

		// trim trailing ":" from devname
		if devname[len(devname)-1] == ':' {
			devname = devname[0 : len(devname)-1]
		}
		e.Name = devname

		net_stat.Entries = append(net_stat.Entries, e)
	}

	record.Net = net_stat

	return nil
}

func ReadMemStat(record *StatRecord) error {
	if record == nil {
		return errors.New("Valid *StatRecord is required.")
	}

	mem_stat := NewMemStat()

	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		var key string
		var val int64
		line := scanner.Text()

		n, err := fmt.Sscanf(line, "%s %d", &key, &val)

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if n != 2 {
			continue
		}

		switch key {
		case "HugePages_Surp:":
			mem_stat.HugePages_Surp = val
		case "HugePages_Rsvd:":
			mem_stat.HugePages_Rsvd = val
		case "HugePages_Free:":
			mem_stat.HugePages_Free = val
		case "HugePages_Total:":
			mem_stat.HugePages_Total = val
		case "Hugepagesize:":
			mem_stat.Hugepagesize = val
		case "AnonHugePages:":
			mem_stat.AnonHugePages = val
		case "Committed_AS:":
			mem_stat.Committed_AS = val
		case "CommitLimit:":
			mem_stat.CommitLimit = val
		case "Bounce:":
			mem_stat.Bounce = val
		case "NFS_Unstable:":
			mem_stat.NFS_Unstable = val
		case "Shmem:":
			mem_stat.Shmem = val
		case "Slab:":
			mem_stat.Slab = val
		case "SReclaimable:":
			mem_stat.SReclaimable = val
		case "SUnreclaim:":
			mem_stat.SUnreclaim = val
		case "KernelStack:":
			mem_stat.KernelStack = val
		case "PageTables:":
			mem_stat.PageTables = val
		case "Mapped:":
			mem_stat.Mapped = val
		case "AnonPages:":
			mem_stat.AnonPages = val
		case "Writeback:":
			mem_stat.Writeback = val
		case "Dirty:":
			mem_stat.Dirty = val
		case "SwapFree:":
			mem_stat.SwapFree = val
		case "SwapTotal:":
			mem_stat.SwapTotal = val
		case "Inactive:":
			mem_stat.Inactive = val
		case "Active:":
			mem_stat.Active = val
		case "SwapCached:":
			mem_stat.SwapCached = val
		case "Cached:":
			mem_stat.Cached = val
		case "Buffers:":
			mem_stat.Buffers = val
		case "MemFree:":
			mem_stat.MemFree = val
		case "MemTotal:":
			mem_stat.MemTotal = val
		}
	}

	record.Mem = mem_stat

	return nil
}
