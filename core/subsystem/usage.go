package subsystem

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
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

func (ccusage *CpuCoreUsage) WriteJsonTo(buf *bytes.Buffer) {
	buf.WriteString(
		fmt.Sprintf(`{"usr":%.2f,"nice":%.2f,"sys":%.2f,"idle":%.2f,"iowait":%.2f,"hardirq":%.2f,"softirq":%.2f,"steal":%.2f,"guest":%.2f,"guestnice":%.2f}`,
			ccusage.User, ccusage.Nice, ccusage.Sys, ccusage.Idle, ccusage.Iowait,
			ccusage.Hardirq, ccusage.Softirq, ccusage.Steal, ccusage.Guest, ccusage.GuestNice))
}

func (cusage *CpuUsage) WriteJsonTo(buf *bytes.Buffer) {
	buf.WriteString(
		fmt.Sprintf(`{"num_core":%d,"all":`, cusage.NumCore))
	cusage.All.WriteJsonTo(buf)
	buf.WriteString(`,"cores":[`)
	for idx, ccusage := range cusage.CoreUsages {
		if idx > 0 {
			buf.WriteString(",")
		}
		ccusage.WriteJsonTo(buf)
	}
	buf.WriteString(`]}`)
}

func GetCpuCoreUsage(c1 *CpuCoreStat, c2 *CpuCoreStat) (*CpuCoreUsage, error) {
	usage := new(CpuCoreUsage)
	itv := c2.Uptime() - c1.Uptime()

	if itv == 0 {
		return nil, errors.New("uptime difference is zero")
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

func (duentry *DiskUsageEntry) WriteJsonTo(buf *bytes.Buffer) {
	fmt.Fprintf(buf,
		`{"riops":%.2f,"wiops":%.2f,"rkbyteps":%.2f,"wkbyteps":%.2f,"rlatency":%.3f,"wlatency":%.3f,"rsize":%.2f,"wsize":%.2f,"qlen":%.2f}`,
		duentry.RdIops, duentry.WrIops, duentry.RdSecps/2.0, duentry.WrSecps/2.0,
		duentry.RdLatency, duentry.WrLatency,
		duentry.AvgRdSize, duentry.AvgWrSize, duentry.ReqQlen)
}

func (dusage *DiskUsage) WriteJsonTo(buf *bytes.Buffer) {
	var devices []string

	for device, _ := range *dusage {
		if device != "total" {
			devices = append(devices, device)
		}
	}
	sort.Strings(devices)

	bytes, err := json.Marshal(devices)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(buf, `{"devices":%s`, string(bytes))

	if len(devices) > 1 {
		devices = append(devices, "total")
	}

	for _, device := range devices {
		usage := (*dusage)[device]
		buf.WriteString(`,"`)
		buf.WriteString(device)
		buf.WriteString(`":`)
		usage.WriteJsonTo(buf)
	}

	buf.WriteByte('}')
}

func avgDelta(v int64, w int64, interval float64) float64 {
	ret := float64(w-v) / interval
	return ret
}

func GetDiskUsage(t1 time.Time, d1 *DiskStat, t2 time.Time, d2 *DiskStat) (*DiskUsage, error) {
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
	cnt := 0

	for _, entry1 := range d1.Entries {
		name := entry1.Name
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

		cnt++

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

	if cnt > 1 {
		(*usage)["total"] = total
	}

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

	cnt := 0

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

		cnt++
	}

	if cnt > 1 {
		(*net_usage)["total"] = total
	}

	return net_usage, nil
}

func (nusage *NetUsage) WriteJsonTo(buf *bytes.Buffer) {
	var devices []string

	for device, _ := range *nusage {
		if device != "total" {
			devices = append(devices, device)
		}
	}
	sort.Strings(devices)

	bytes, err := json.Marshal(devices)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(buf, `{"devices":%s`, string(bytes))

	if len(devices) > 1 {
		devices = append(devices, "total")
	}

	for _, device := range devices {
		usage := (*nusage)[device]
		buf.WriteString(`,"`)
		buf.WriteString(device)
		buf.WriteString(`":`)
		usage.WriteJsonTo(buf)
	}

	buf.WriteByte('}')
}

func (entry *NetUsageEntry) WriteJsonTo(buf *bytes.Buffer) {
	buf.WriteString(
		fmt.Sprintf(`{"rxkbyteps":%.2f,"rxpktps":%.2f,"rxerrps":%.2f,"rxdropps":%.2f,"txkbyteps":%.2f,"txpktps":%.2f,"txerrps":%.2f,"txdropps":%.2f}`,
			entry.RxBytesPerSec/1024.0, entry.RxPacketsPerSec,
			entry.RxErrorsPerSec, entry.RxDropsPerSec,
			entry.TxBytesPerSec/1024.0, entry.TxPacketsPerSec,
			entry.TxErrorsPerSec, entry.TxDropsPerSec))
}
