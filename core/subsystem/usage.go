package subsystem

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	projson "github.com/hayamiz/go-projson"
)

type CpuCoreUsage struct {
	User      float64
	Nice      float64
	Sys       float64
	Idle      float64
	Iowait    float64
	Hardirq   float64
	Softirq   float64
	Steal     float64
	Guest     float64
	GuestNice float64
}

type CpuUsage struct {
	All        *CpuCoreUsage
	NumCore    int
	CoreUsages []*CpuCoreUsage
}

type CpuCoreIntrUsage struct {
	Device float64 // intr/sec for devices
	System float64 // intr/sec for system internal interrupts (timers, TLB miss, ...)
}

type InterruptUsage struct {
	Interval time.Duration

	NumEntries uint
	NumCore    int

	CoreIntrUsages []*CpuCoreIntrUsage
}

type DiskUsageEntry struct {
	Interval time.Duration

	RdIops    float64
	WrIops    float64
	RdSectors int64
	WrSectors int64
	RdSecps   float64 // sectors per second
	WrSecps   float64 // sectors per second
	RdLatency float64 // msec
	WrLatency float64 // msec
	AvgRdSize float64 // sectors
	AvgWrSize float64 // sectors
	ReqQlen   float64
}

type DiskUsage map[string]*DiskUsageEntry

type NetUsageEntry struct {
	Interval time.Duration

	RxBytesPerSec      float64
	RxPacketsPerSec    float64
	RxErrorsPerSec     float64
	RxDropsPerSec      float64
	RxFifoPerSec       float64
	RxFramePerSec      float64
	RxCompressedPerSec float64
	RxMulticastPerSec  float64
	TxBytesPerSec      float64
	TxPacketsPerSec    float64
	TxErrorsPerSec     float64
	TxDropsPerSec      float64
	TxFifoPerSec       float64
	TxFramePerSec      float64
	TxCompressedPerSec float64
	TxMulticastPerSec  float64
}

type NetUsage map[string]*NetUsageEntry

type MemUsage struct {
	mem *MemStat
}

var UseColor = false

func SetUseColor(use_color bool) {
	UseColor = use_color
}

func (ccusage *CpuCoreUsage) WriteJsonTo(printer *projson.JsonPrinter) {
	printer.BeginObject()
	printer.PutKey("usr")
	printer.PutFloatFmt(ccusage.User, "%.2f")
	printer.PutKey("nice")
	printer.PutFloatFmt(ccusage.Nice, "%.2f")
	printer.PutKey("sys")
	printer.PutFloatFmt(ccusage.Sys, "%.2f")
	printer.PutKey("idle")
	printer.PutFloatFmt(ccusage.Idle, "%.2f")
	printer.PutKey("iowait")
	printer.PutFloatFmt(ccusage.Iowait, "%.2f")
	printer.PutKey("hardirq")
	printer.PutFloatFmt(ccusage.Hardirq, "%.2f")
	printer.PutKey("softirq")
	printer.PutFloatFmt(ccusage.Softirq, "%.2f")
	printer.PutKey("steal")
	printer.PutFloatFmt(ccusage.Steal, "%.2f")
	printer.PutKey("guest")
	printer.PutFloatFmt(ccusage.Guest, "%.2f")
	printer.PutKey("guestnice")
	printer.PutFloatFmt(ccusage.GuestNice, "%.2f")
	printer.FinishObject()
}

func (cusage *CpuUsage) WriteJsonTo(printer *projson.JsonPrinter) {
	printer.BeginObject()
	printer.PutKey("num_core")
	printer.PutInt(cusage.NumCore)
	printer.PutKey("all")

	cusage.All.WriteJsonTo(printer)

	printer.PutKey("cores")
	printer.BeginArray()
	for _, ccusage := range cusage.CoreUsages {
		ccusage.WriteJsonTo(printer)
	}
	printer.FinishArray()
	printer.FinishObject()
}

func GetCpuCoreUsage(c1 *CpuCoreStat, c2 *CpuCoreStat) (*CpuCoreUsage, error) {
	usage := new(CpuCoreUsage)
	itv := c2.Uptime() - c1.Uptime()

	if itv == 0 {
		// return nil, errors.New("uptime difference is zero")
		usage.User = 0
		usage.Nice = 0
		usage.Sys = 0
		usage.Idle = 0
		usage.Iowait = 0
		usage.Hardirq = 0
		usage.Softirq = 0
		usage.Steal = 0
		usage.Guest = 0
		usage.GuestNice = 0

		return usage, nil
	} else if itv < 0 {
		return nil, errors.New("uptime difference is negative")
	}

	user := usageItem(c1.User-c1.Guest, c2.User-c2.Guest, itv)
	if user < 0.0 {
		user = 0.0
	}
	nice := usageItem(c1.Nice-c1.GuestNice, c2.Nice-c2.GuestNice, itv)
	if nice < 0.0 {
		nice = 0.0
	}

	usage.User = user
	usage.Nice = nice
	usage.Sys = usageItem(c1.Sys, c2.Sys, itv)
	usage.Idle = usageItem(c1.Idle, c2.Idle, itv)
	usage.Iowait = usageItem(c1.Iowait, c2.Iowait, itv)
	usage.Hardirq = usageItem(c1.Hardirq, c2.Hardirq, itv)
	usage.Softirq = usageItem(c1.Softirq, c2.Softirq, itv)
	usage.Steal = usageItem(c1.Steal, c2.Steal, itv)
	usage.Guest = usageItem(c1.Guest, c2.Guest, itv)
	usage.GuestNice = usageItem(c1.GuestNice, c2.GuestNice, itv)

	return usage, nil
}

func GetCpuUsage(c1 *CpuStat, c2 *CpuStat) (*CpuUsage, error) {
	var err error

	usage := new(CpuUsage)
	usage.NumCore = c1.NumCore

	usage.CoreUsages = make([]*CpuCoreUsage, usage.NumCore)
	for idx, _ := range usage.CoreUsages {
		usage.CoreUsages[idx], err = GetCpuCoreUsage(&c1.CoreStats[idx], &c2.CoreStats[idx])
		if err != nil {
			return nil, err
		}
	}
	usage.All, err = GetCpuCoreUsage(&c1.All, &c2.All)
	if err != nil {
		return nil, err
	}

	// scale: NumCore * 100% as maximum
	usage.All.User *= float64(usage.NumCore)
	usage.All.Nice *= float64(usage.NumCore)
	usage.All.Sys *= float64(usage.NumCore)
	usage.All.Idle *= float64(usage.NumCore)
	usage.All.Iowait *= float64(usage.NumCore)
	usage.All.Hardirq *= float64(usage.NumCore)
	usage.All.Softirq *= float64(usage.NumCore)
	usage.All.Steal *= float64(usage.NumCore)
	usage.All.Guest *= float64(usage.NumCore)
	usage.All.GuestNice *= float64(usage.NumCore)

	return usage, nil
}

func GetInterruptUsage(t1 time.Time, i1 *InterruptStat, t2 time.Time, i2 *InterruptStat) (*InterruptUsage, error) {
	num_core := i1.Entries[0].NumCore

	usage := new(InterruptUsage)
	usage.Interval = t2.Sub(t1)
	usage.NumEntries = i1.NumEntries
	usage.NumCore = num_core
	usage.CoreIntrUsages = make([]*CpuCoreIntrUsage, num_core)

	for coreid := 0; coreid < usage.NumCore; coreid += 1 {
		core_usage := new(CpuCoreIntrUsage)
		core_usage.Device = 0
		core_usage.System = 0

		core_dev_count := 0
		core_sys_count := 0

		for idx, istat_entry1 := range i1.Entries {
			istat_entry2 := i2.Entries[idx]

			if istat_entry1.IrqNo != istat_entry2.IrqNo ||
				istat_entry1.IrqType != istat_entry2.IrqType {
				return nil, errors.New("Intr stat format changed")
			}

			countup := istat_entry2.IntrCounts[coreid] - istat_entry1.IntrCounts[coreid]
			if istat_entry1.IrqNo != -1 {
				core_dev_count += countup
			} else {
				core_sys_count += countup
			}
		}

		core_usage.Device = float64(core_dev_count) / usage.Interval.Seconds()
		core_usage.System = float64(core_sys_count) / usage.Interval.Seconds()

		usage.CoreIntrUsages[coreid] = core_usage
	}

	return usage, nil
}

func (intr_usage *InterruptUsage) WriteJsonTo(printer *projson.JsonPrinter) {
	printer.BeginObject()
	printer.PutKey("core_dev_intr")
	printer.BeginArray()
	for _, core_usage := range intr_usage.CoreIntrUsages {
		printer.PutFloatFmt(core_usage.Device, "%.2f")
	}
	printer.FinishArray()

	printer.PutKey("core_sys_intr")
	printer.BeginArray()
	for _, core_usage := range intr_usage.CoreIntrUsages {
		printer.PutFloatFmt(core_usage.System, "%.2f")
	}
	printer.FinishArray()
	printer.FinishObject()
}

func (duentry *DiskUsageEntry) WriteJsonTo(printer *projson.JsonPrinter) {
	printer.BeginObject()
	printer.PutKey("riops")
	printer.PutFloatFmt(duentry.RdIops, "%.2f")
	printer.PutKey("wiops")
	printer.PutFloatFmt(duentry.WrIops, "%.2f")
	printer.PutKey("rkbyteps")
	printer.PutFloatFmt(duentry.RdSecps/2.0, "%.2f")
	printer.PutKey("wkbyteps")
	printer.PutFloatFmt(duentry.WrSecps/2.0, "%.2f")
	printer.PutKey("rlatency")
	printer.PutFloatFmt(duentry.RdLatency, "%.3f")
	printer.PutKey("wlatency")
	printer.PutFloatFmt(duentry.WrLatency, "%.3f")
	printer.PutKey("rsize")
	printer.PutFloatFmt(duentry.AvgRdSize, "%.2f")
	printer.PutKey("wsize")
	printer.PutFloatFmt(duentry.AvgWrSize, "%.2f")
	printer.PutKey("qlen")
	printer.PutFloatFmt(duentry.ReqQlen, "%.2f")
	printer.FinishObject()
}

func strarrayToString(arr []string) string {
	buf := bytes.NewBuffer([]byte{})

	fmt.Fprintf(buf, "[")
	for i, elem := range arr {
		if i > 0 {
			fmt.Fprintf(buf, ",")
		}

		if UseColor {
			fmt.Fprintf(buf, "\033[35m\"%s\"\033[0m", elem)
		} else {
			fmt.Fprintf(buf, "\"%s\"", elem)
		}
	}
	fmt.Fprintf(buf, "]")

	return buf.String()
}

func (dusage *DiskUsage) WriteJsonTo(printer *projson.JsonPrinter) {
	var devices []string

	for device, _ := range *dusage {
		if device != "total" {
			devices = append(devices, device)
		}
	}
	sort.Strings(devices)

	printer.BeginObject()
	printer.PutKey("devices")
	printer.BeginArray()
	for _, device := range devices {
		printer.PutString(device)
	}
	printer.FinishArray()

	devices = append(devices, "total")

	for _, device := range devices {
		usage := (*dusage)[device]

		printer.PutKey(device)
		usage.WriteJsonTo(printer)
	}

	printer.FinishObject()
}

func avgDelta(v int64, w int64, interval float64) float64 {
	ret := float64(w-v) / interval
	return ret
}

func GetDiskUsage(t1 time.Time, d1 *DiskStat, t2 time.Time, d2 *DiskStat) (*DiskUsage, error) {
	return GetDiskUsage1(t1, d1, t2, d2, nil)
}

func GetDiskUsage1(t1 time.Time, d1 *DiskStat, t2 time.Time, d2 *DiskStat,
	filter *regexp.Regexp) (*DiskUsage, error) {
	interval := t2.Sub(t1)
	itv := interval.Seconds()

	if itv <= 0.0 {
		return nil, errors.New("negative interval")
	}

	if len(d1.Entries) == 0 || len(d2.Entries) == 0 {
		return nil, errors.New("no DiskEntry")
	}

	usage := new(DiskUsage)
	(*usage) = make(DiskUsage)
	total := new(DiskUsageEntry)

	var total_rd_ios int64 = 0
	var total_wr_ios int64 = 0

	for _, entry1 := range d1.Entries {
		name := entry1.Name
		if filter != nil {
			if !filter.MatchString(name) {
				continue
			}
		}
		var entry2 *DiskStatEntry = nil
		for _, e := range d2.Entries {
			if e.Name == entry1.Name {
				entry2 = e
				break
			}
		}
		if entry2 == nil {
			continue
		}

		rd_latency := 0.0
		wr_latency := 0.0
		avg_rd_sz := 0.0
		avg_wr_sz := 0.0
		if entry2.RdIos != entry1.RdIos {
			rd_latency = float64(entry2.RdTicks-entry1.RdTicks) / float64(entry2.RdIos-entry1.RdIos)
			avg_rd_sz = float64(entry2.RdSectors-entry1.RdSectors) / float64(entry2.RdIos-entry1.RdIos)
		}
		if entry2.WrIos != entry1.WrIos {
			wr_latency = float64(entry2.WrTicks-entry1.WrTicks) / float64(entry2.WrIos-entry1.WrIos)
			avg_wr_sz = float64(entry2.WrSectors-entry1.WrSectors) / float64(entry2.WrIos-entry1.WrIos)
		}

		entry := &DiskUsageEntry{
			interval,
			avgDelta(entry1.RdIos, entry2.RdIos, itv),
			avgDelta(entry1.WrIos, entry2.WrIos, itv),
			entry2.RdSectors - entry1.RdSectors,
			entry2.WrSectors - entry1.WrSectors,
			avgDelta(entry1.RdSectors, entry2.RdSectors, itv),
			avgDelta(entry1.WrSectors, entry2.WrSectors, itv),
			rd_latency,
			wr_latency,
			avg_rd_sz,
			avg_wr_sz,
			float64(entry2.ReqTicks-entry1.ReqTicks) / itv / 1.0e3,
		}

		(*usage)[name] = entry

		total.RdIops += entry.RdIops
		total.WrIops += entry.WrIops
		total.RdSectors += entry.RdSectors
		total.WrSectors += entry.WrSectors
		total.RdSecps += entry.RdSecps
		total.WrSecps += entry.WrSecps
		total.RdLatency += entry.RdLatency * float64(entry2.RdIos-entry1.RdIos)
		total.WrLatency += entry.WrLatency * float64(entry2.WrIos-entry1.WrIos)
		total.AvgRdSize += entry.AvgRdSize * float64(entry2.RdIos-entry1.RdIos)
		total.AvgWrSize += entry.AvgWrSize * float64(entry2.WrIos-entry1.WrIos)
		total.ReqQlen += entry.ReqQlen

		total_rd_ios += entry2.RdIos - entry1.RdIos
		total_wr_ios += entry2.WrIos - entry1.WrIos
	}

	if total_rd_ios > 0 {
		total.RdLatency /= float64(total_rd_ios)
		total.AvgRdSize /= float64(total_rd_ios)
	}
	if total_wr_ios > 0 {
		total.WrLatency /= float64(total_wr_ios)
		total.AvgWrSize /= float64(total_wr_ios)
	}

	(*usage)["total"] = total

	return usage, nil
}

func usageItem(v1 int64, v2 int64, itv int64) float64 {
	return float64(v2-v1) / float64(itv) * 100.0
}

func GetNetUsage(t1 time.Time, d1 *NetStat, t2 time.Time, d2 *NetStat) (*NetUsage, error) {
	if len(d1.Entries) == 0 && len(d2.Entries) == 0 {
		return nil, errors.New("no entries")
	}

	interval := t2.Sub(t1)
	itv := interval.Seconds()

	if itv <= 0 {
		return nil, errors.New("Non-positive interval")
	}

	net_usage := new(NetUsage)
	(*net_usage) = make(NetUsage)
	total := new(NetUsageEntry)

	for _, d1_entry := range d1.Entries {
		devname := d1_entry.Name

		// find devname in d2
		var d2_entry *NetStatEntry = nil
		for _, e := range d2.Entries {
			if e.Name == devname {
				d2_entry = e
				break
			}
		}

		if d2_entry == nil {
			continue
		}

		ue := new(NetUsageEntry)

		ue.Interval = interval
		ue.RxBytesPerSec = avgDelta(d1_entry.RxBytes, d2_entry.RxBytes, itv)
		ue.RxPacketsPerSec = avgDelta(d1_entry.RxPackets, d2_entry.RxPackets, itv)
		ue.RxErrorsPerSec = avgDelta(d1_entry.RxErrors, d2_entry.RxErrors, itv)
		ue.RxDropsPerSec = avgDelta(d1_entry.RxDrops, d2_entry.RxDrops, itv)
		ue.RxFifoPerSec = avgDelta(d1_entry.RxFifo, d2_entry.RxFifo, itv)
		ue.RxFramePerSec = avgDelta(d1_entry.RxFrame, d2_entry.RxFrame, itv)
		ue.RxCompressedPerSec = avgDelta(d1_entry.RxCompressed, d2_entry.RxCompressed, itv)
		ue.RxMulticastPerSec = avgDelta(d1_entry.RxMulticast, d2_entry.RxMulticast, itv)
		ue.TxBytesPerSec = avgDelta(d1_entry.TxBytes, d2_entry.TxBytes, itv)
		ue.TxPacketsPerSec = avgDelta(d1_entry.TxPackets, d2_entry.TxPackets, itv)
		ue.TxErrorsPerSec = avgDelta(d1_entry.TxErrors, d2_entry.TxErrors, itv)
		ue.TxDropsPerSec = avgDelta(d1_entry.TxDrops, d2_entry.TxDrops, itv)
		ue.TxFifoPerSec = avgDelta(d1_entry.TxFifo, d2_entry.TxFifo, itv)
		ue.TxFramePerSec = avgDelta(d1_entry.TxFrame, d2_entry.TxFrame, itv)
		ue.TxCompressedPerSec = avgDelta(d1_entry.TxCompressed, d2_entry.TxCompressed, itv)
		ue.TxMulticastPerSec = avgDelta(d1_entry.TxMulticast, d2_entry.TxMulticast, itv)

		(*net_usage)[devname] = ue

		total.RxBytesPerSec += ue.RxBytesPerSec
		total.RxPacketsPerSec += ue.RxPacketsPerSec
		total.RxErrorsPerSec += ue.RxErrorsPerSec
		total.RxDropsPerSec += ue.RxDropsPerSec
		total.RxFifoPerSec += ue.RxFifoPerSec
		total.RxFramePerSec += ue.RxFramePerSec
		total.RxCompressedPerSec += ue.RxCompressedPerSec
		total.RxMulticastPerSec += ue.RxMulticastPerSec
		total.TxBytesPerSec += ue.TxBytesPerSec
		total.TxPacketsPerSec += ue.TxPacketsPerSec
		total.TxErrorsPerSec += ue.TxErrorsPerSec
		total.TxDropsPerSec += ue.TxDropsPerSec
		total.TxFifoPerSec += ue.TxFifoPerSec
		total.TxFramePerSec += ue.TxFramePerSec
		total.TxCompressedPerSec += ue.TxCompressedPerSec
		total.TxMulticastPerSec += ue.TxMulticastPerSec
	}

	(*net_usage)["total"] = total

	return net_usage, nil
}

func (nusage *NetUsage) WriteJsonTo(printer *projson.JsonPrinter) {
	var devices []string

	for device, _ := range *nusage {
		if device != "total" {
			devices = append(devices, device)
		}
	}
	sort.Strings(devices)

	printer.BeginObject()
	printer.PutKey("devices")
	printer.BeginArray()
	for _, device := range devices {
		printer.PutString(device)
	}
	printer.FinishArray()

	devices = append(devices, "total")

	for _, device := range devices {
		usage := (*nusage)[device]

		printer.PutKey(device)
		usage.WriteJsonTo(printer)
	}

	printer.FinishObject()
}

func GetMemUsage(mem *MemStat) (*MemUsage, error) {
	if mem == nil {
		return nil, errors.New("invalid memstat")
	}

	musage := new(MemUsage)
	musage.mem = mem

	return musage, nil
}

func (musage *MemUsage) WriteJsonTo(printer *projson.JsonPrinter) {
	printer.BeginObject()

	printer.PutKey("MemTotal")
	printer.PutInt64(musage.mem.MemTotal)
	printer.PutKey("MemFree")
	printer.PutInt64(musage.mem.MemFree)
	printer.PutKey("Buffers")
	printer.PutInt64(musage.mem.Buffers)
	printer.PutKey("Cached")
	printer.PutInt64(musage.mem.Cached)
	printer.PutKey("SwapCached")
	printer.PutInt64(musage.mem.SwapCached)
	printer.PutKey("Active")
	printer.PutInt64(musage.mem.Active)
	printer.PutKey("Inactive")
	printer.PutInt64(musage.mem.Inactive)
	printer.PutKey("SwapTotal")
	printer.PutInt64(musage.mem.SwapTotal)
	printer.PutKey("SwapFree")
	printer.PutInt64(musage.mem.SwapFree)
	printer.PutKey("Dirty")
	printer.PutInt64(musage.mem.Dirty)
	printer.PutKey("Writeback")
	printer.PutInt64(musage.mem.Writeback)
	printer.PutKey("AnonPages")
	printer.PutInt64(musage.mem.AnonPages)
	printer.PutKey("Mapped")
	printer.PutInt64(musage.mem.Mapped)
	printer.PutKey("Shmem")
	printer.PutInt64(musage.mem.Shmem)
	printer.PutKey("NFS_Unstable")
	printer.PutInt64(musage.mem.NFS_Unstable)
	printer.PutKey("Bounce")
	printer.PutInt64(musage.mem.Bounce)
	printer.PutKey("CommitLimit")
	printer.PutInt64(musage.mem.CommitLimit)
	printer.PutKey("Committed_AS")
	printer.PutInt64(musage.mem.Committed_AS)
	printer.PutKey("AnonHugePages")
	printer.PutInt64(musage.mem.AnonHugePages)
	printer.PutKey("HugePages_Total")
	printer.PutInt64(musage.mem.HugePages_Total)
	printer.PutKey("HugePages_Free")
	printer.PutInt64(musage.mem.HugePages_Free)
	printer.PutKey("HugePages_Rsvd")
	printer.PutInt64(musage.mem.HugePages_Rsvd)
	printer.PutKey("HugePages_Surp")
	printer.PutInt64(musage.mem.HugePages_Surp)

	printer.FinishObject()
}

func (entry *NetUsageEntry) WriteJsonTo(printer *projson.JsonPrinter) {
	printer.BeginObject()

	printer.PutKey("rxkbyteps")
	printer.PutFloatFmt(entry.RxBytesPerSec/1024.0, "%.2f")
	printer.PutKey("rxpktps")
	printer.PutFloatFmt(entry.RxPacketsPerSec, "%.2f")
	printer.PutKey("rxerrps")
	printer.PutFloatFmt(entry.RxErrorsPerSec, "%.2f")
	printer.PutKey("rxdropps")
	printer.PutFloatFmt(entry.RxDropsPerSec, "%.2f")
	printer.PutKey("txkbyteps")
	printer.PutFloatFmt(entry.TxBytesPerSec/1024.0, "%.2f")
	printer.PutKey("txpktps")
	printer.PutFloatFmt(entry.TxPacketsPerSec, "%.2f")
	printer.PutKey("txerrps")
	printer.PutFloatFmt(entry.TxErrorsPerSec, "%.2f")
	printer.PutKey("txdropps")
	printer.PutFloatFmt(entry.TxDropsPerSec, "%.2f")

	printer.FinishObject()
}
