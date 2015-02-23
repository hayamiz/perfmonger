// +build linux

package subsystem

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
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
		record.Cpu = NewCpuStat(runtime.NumCPU())
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
			var core_stat *CpuCoreStat
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

func ReadDiskStats(record *StatRecord) error {
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

		record.Disk.Entries = append(record.Disk.Entries, entry)
	}

	return nil
}
