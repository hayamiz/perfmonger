package subsystem

import "time"

//
// Record
//

type CpuCoreStat struct {
	User      int64
	Nice      int64
	Sys       int64
	Idle      int64
	Iowait    int64
	Hardirq   int64
	Softirq   int64
	Steal     int64
	Guest     int64
	GuestNice int64
}

type CpuStat struct {
	All       CpuCoreStat
	NumCore   int
	CoreStats []CpuCoreStat
}

type ProcStat struct {
	ContextSwitch int64
	Fork          int64
}

type SoftIrqStat struct {
	Hi          int64
	Timer       int64
	NetTx       int64
	NetRx       int64
	Block       int64
	BlockIopoll int64
	Tasklet     int64
	Sched       int64
	Hrtimer     int64
	Rcu         int64
}

type DiskStatEntry struct {
	Major      uint
	Minor      uint
	Name       string
	RdIos      int64
	RdMerges   int64
	RdSectors  int64
	RdTicks    int64
	WrIos      int64
	WrMerges   int64
	WrSectors  int64
	WrTicks    int64
	IosPgr     int64
	TotalTicks int64
	ReqTicks   int64
}

type DiskStat struct {
	Entries []*DiskStatEntry
}

type StatRecord struct {
	Time    time.Time
	Cpu     *CpuStat
	Proc    *ProcStat
	Disk    *DiskStat
	Softirq *SoftIrqStat
}

func (core_stat *CpuCoreStat) Clear() {
	core_stat.User = 0
	core_stat.Nice = 0
	core_stat.Sys = 0
	core_stat.Iowait = 0
	core_stat.Steal = 0
	core_stat.Hardirq = 0
	core_stat.Softirq = 0
	core_stat.Guest = 0
	core_stat.GuestNice = 0
	core_stat.Idle = 0
}

func (core_stat *CpuCoreStat) Uptime() int64 {
	// Don't take Guest because User include Guest
	return core_stat.User +
		core_stat.Nice +
		core_stat.Sys +
		core_stat.Iowait +
		core_stat.Steal +
		core_stat.Hardirq +
		core_stat.Softirq +
		core_stat.Idle
}

func NewCpuStat(num_core int) *CpuStat {
	if num_core < 1 {
		return nil
	}

	cpu_stat := new(CpuStat)

	cpu_stat.NumCore = num_core
	cpu_stat.CoreStats = make([]CpuCoreStat, num_core)

	return cpu_stat
}

func (cpu_stat *CpuStat) Clear() {
	cpu_stat.All.Clear()
	for _, core_stat := range cpu_stat.CoreStats {
		core_stat.Clear()
	}
}

func NewProcStat() *ProcStat {
	return &ProcStat{0, 0}
}

func (proc_stat *ProcStat) Clear() {
	proc_stat.ContextSwitch = 0
	proc_stat.Fork = 0
}

func NewDiskStatEntry() *DiskStatEntry {
	return new(DiskStatEntry)
}

func NewDiskStat() *DiskStat {
	return &DiskStat{[]*DiskStatEntry{}}
}

func (disk_stat *DiskStat) Clear() {
	disk_stat.Entries = []*DiskStatEntry{}
}

func (sirq_stat *SoftIrqStat) Clear() {
	sirq_stat.Hi = 0
	sirq_stat.Timer = 0
	sirq_stat.NetTx = 0
	sirq_stat.NetRx = 0
	sirq_stat.Block = 0
	sirq_stat.BlockIopoll = 0
	sirq_stat.Tasklet = 0
	sirq_stat.Sched = 0
	sirq_stat.Hrtimer = 0
	sirq_stat.Rcu = 0
}

func NewStatRecord() *StatRecord {
	return &StatRecord{
		time.Now(),
		nil,
		nil,
		nil,
		nil,
	}
}

func (rec *StatRecord) Clear() {
	rec.Cpu.Clear()
	rec.Proc.Clear()
	rec.Disk.Clear()
}
