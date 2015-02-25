package subsystem

import "testing"

func TestCpuCoreStatUptime(t *testing.T) {
	var corestat *CpuCoreStat

	corestat = new(CpuCoreStat)

	if corestat.Uptime() != 0 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 0)
	}

	corestat.User += 3
	if corestat.Uptime() != 3 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 10)
	}

	corestat.Sys += 5
	if corestat.Uptime() != 8 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 8)
	}

	corestat.Iowait += 7
	if corestat.Uptime() != 15 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 15)
	}

	corestat.Iowait += 11
	if corestat.Uptime() != 26 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 26)
	}

	corestat.Steal += 13
	if corestat.Uptime() != 39 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 39)
	}

	corestat.Hardirq += 17
	if corestat.Uptime() != 56 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 56)
	}

	corestat.Softirq += 19
	if corestat.Uptime() != 75 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 75)
	}

	corestat.Idle += 23
	if corestat.Uptime() != 98 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 98)
	}

	corestat.Nice += 29
	if corestat.Uptime() != 127 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 127)
	}

	corestat.Guest += 31
	if corestat.Uptime() != 127 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 127)
	}

	corestat.GuestNice += 37
	if corestat.Uptime() != 127 {
		t.Errorf("corestat.Uptime() = %v, want %v", corestat.Uptime(), 127)
	}
}

func TestCpuCoreStatClear(t *testing.T) {
	var corestat *CpuCoreStat

	corestat = new(CpuCoreStat)

	corestat.User = 3
	corestat.Sys = 5
	corestat.Iowait = 7
	corestat.Idle = 11
	corestat.Steal = 13
	corestat.Hardirq = 17
	corestat.Softirq = 19
	corestat.Nice = 23
	corestat.Guest = 29
	corestat.GuestNice = 31

	if corestat.User == 0 ||
		corestat.Sys == 0 ||
		corestat.Iowait == 0 ||
		corestat.Idle == 0 ||
		corestat.Steal == 0 ||
		corestat.Hardirq == 0 ||
		corestat.Softirq == 0 ||
		corestat.Nice == 0 ||
		corestat.Guest == 0 ||
		corestat.GuestNice == 0 {
		t.Errorf("corestat = %v, want all fields not to be 0.",
			corestat)
	}

	corestat.Clear()

	if corestat.User != 0 ||
		corestat.Sys != 0 ||
		corestat.Iowait != 0 ||
		corestat.Idle != 0 ||
		corestat.Steal != 0 ||
		corestat.Hardirq != 0 ||
		corestat.Softirq != 0 ||
		corestat.Nice != 0 ||
		corestat.Guest != 0 ||
		corestat.GuestNice != 0 {
		t.Errorf("corestat = %v, want all fields to be 0.",
			corestat)
	}
}

func TestNewCpuStat(t *testing.T) {
	var cpustat *CpuStat
	var num_core int

	num_core = 0
	cpustat = NewCpuStat(num_core)
	if cpustat != nil {
		t.Errorf("NewCore(%d) = %v, want %v",
			num_core, cpustat, nil)
	}

	num_core = 1
	cpustat = NewCpuStat(num_core)
	if cpustat == nil {
		t.Errorf("NewCore(%d) = %v, expect != %v",
			num_core, cpustat, nil)
	}

	for num_core = 1; num_core < 1024; num_core++ {
		cpustat = NewCpuStat(num_core)
		if len(cpustat.CoreStats) != num_core {
			t.Errorf("len(cpustat.CoreStats) = %v, want %v",
				len(cpustat.CoreStats), num_core)
		}
	}
}

func TestCpuStatClear(t *testing.T) {
	num_core := 4
	cpustat := NewCpuStat(num_core)

	cores := []*CpuCoreStat{&cpustat.All}
	for i := 0; i < num_core; i++ {
		cores = append(cores, &cpustat.CoreStats[i])
	}

	for _, corestat := range cores {
		corestat.User = 3
		corestat.Sys = 5
		corestat.Iowait = 7
		corestat.Idle = 11
		corestat.Steal = 13
		corestat.Hardirq = 17
		corestat.Softirq = 19
		corestat.Nice = 23
		corestat.Guest = 29
		corestat.GuestNice = 31
	}

	for _, corestat := range cores {
		if corestat.User == 0 ||
			corestat.Sys == 0 ||
			corestat.Iowait == 0 ||
			corestat.Idle == 0 ||
			corestat.Steal == 0 ||
			corestat.Hardirq == 0 ||
			corestat.Softirq == 0 ||
			corestat.Nice == 0 ||
			corestat.Guest == 0 ||
			corestat.GuestNice == 0 {
			t.Errorf("corestat = %v, want all fields not to be 0.",
				corestat)
		}
	}

	cpustat.Clear()

	for _, corestat := range cores {
		if corestat.User != 0 ||
			corestat.Sys != 0 ||
			corestat.Iowait != 0 ||
			corestat.Idle != 0 ||
			corestat.Steal != 0 ||
			corestat.Hardirq != 0 ||
			corestat.Softirq != 0 ||
			corestat.Nice != 0 ||
			corestat.Guest != 0 ||
			corestat.GuestNice != 0 {
			t.Errorf("corestat = %v, want all fields to be 0.",
				corestat)
		}
	}
}
