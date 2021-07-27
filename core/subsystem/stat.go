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

type InterruptStatEntry struct {
	IrqNo   int    // >0 if associated with devices, -1 if not
	IrqType string // set intr name if IrqNo == -1

	NumCore    int
	IntrCounts []int
	Descr      string
}

type InterruptStat struct {
	NumEntries uint
	Entries    []*InterruptStatEntry
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

type NetStatEntry struct {
	Name         string
	RxBytes      int64
	RxPackets    int64
	RxErrors     int64
	RxDrops      int64
	RxFifo       int64
	RxFrame      int64
	RxCompressed int64
	RxMulticast  int64
	TxBytes      int64
	TxPackets    int64
	TxErrors     int64
	TxDrops      int64
	TxFifo       int64
	TxFrame      int64
	TxCompressed int64
	TxMulticast  int64
}

type NetStat struct {
	Entries []*NetStatEntry
}

// all values are recorded in KB
type MemStat struct {
	MemTotal        int64
	MemFree         int64
	Buffers         int64
	Cached          int64
	SwapCached      int64
	Active          int64
	Inactive        int64
	SwapTotal       int64
	SwapFree        int64
	Dirty           int64
	Writeback       int64
	AnonPages       int64
	Mapped          int64
	Shmem           int64
	Slab            int64
	SReclaimable    int64
	SUnreclaim      int64
	KernelStack     int64
	PageTables      int64
	NFS_Unstable    int64
	Bounce          int64
	CommitLimit     int64
	Committed_AS    int64
	AnonHugePages   int64
	HugePages_Total int64
	HugePages_Free  int64
	HugePages_Rsvd  int64
	HugePages_Surp  int64
	Hugepagesize    int64
}

type StatRecord struct {
	Time      time.Time
	Cpu       *CpuStat
	Interrupt *InterruptStat
	Proc      *ProcStat
	Disk      *DiskStat
	Softirq   *SoftIrqStat
	Net       *NetStat
	Mem       *MemStat
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
	for idx, _ := range cpu_stat.CoreStats {
		cpu_stat.CoreStats[idx].Clear()
	}
}

func NewInterruptStat() *InterruptStat {
	intr_stat := new(InterruptStat)
	intr_stat.NumEntries = 0
	intr_stat.Entries = make([]*InterruptStatEntry, 0)

	return intr_stat
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

func NewNetStatEntry() *NetStatEntry {
	return new(NetStatEntry)
}

func (entry *NetStatEntry) Clear() {
	entry.Name = ""
	entry.RxBytes = 0
	entry.RxPackets = 0
	entry.RxErrors = 0
	entry.RxDrops = 0
	entry.TxBytes = 0
	entry.TxPackets = 0
	entry.TxErrors = 0
	entry.TxDrops = 0
}

func NewNetStat() *NetStat {
	return new(NetStat)
}

func NewMemStat() *MemStat {
	return new(MemStat)
}

func (entry *MemStat) Clear() {
	entry.MemTotal = 0
	entry.MemFree = 0
	entry.Buffers = 0
	entry.Cached = 0
	entry.SwapCached = 0
	entry.Active = 0
	entry.Inactive = 0
	entry.SwapTotal = 0
	entry.SwapFree = 0
	entry.Dirty = 0
	entry.Writeback = 0
	entry.AnonPages = 0
	entry.Mapped = 0
	entry.Shmem = 0
	entry.Slab = 0
	entry.SReclaimable = 0
	entry.NFS_Unstable = 0
	entry.SUnreclaim = 0
	entry.KernelStack = 0
	entry.PageTables = 0
	entry.NFS_Unstable = 0
	entry.Bounce = 0
	entry.CommitLimit = 0
	entry.Committed_AS = 0
	entry.AnonHugePages = 0
	entry.HugePages_Total = 0
	entry.HugePages_Free = 0
	entry.HugePages_Rsvd = 0
	entry.HugePages_Surp = 0
	entry.Hugepagesize = 0
}

func NewStatRecord() *StatRecord {
	return &StatRecord{
		time.Now(),
		nil,
		nil,
		nil,
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
