package subsystem

import "testing"

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
