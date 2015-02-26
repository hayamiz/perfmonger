package subsystem

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	RdIops    float64
	WrIops    float64
	RdSecps   float64
	WrSecps   float64
	RdLatency float64
	WrLatency float64
	AvgRdSize float64
	AvgWrSize float64
	ReqQlen   float64
}

type DiskUsage map[string]*DiskUsageEntry

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
		`{"riops":%.2f,"wiops":%.2f,"rkbyteps":%.2f,"wkbytesps":%.2f,"rlatency":%.3f,"wlatency":%.3f,"rsize":%.2f,"wsize":%.2f,"qlen":%.2f}`,
		duentry.RdIops, duentry.WrIops, duentry.RdSecps/2.0, duentry.WrSecps/2.0,
		duentry.RdLatency, duentry.WrLatency,
		duentry.AvgRdSize, duentry.AvgWrSize, duentry.ReqQlen)
}

func (dusage *DiskUsage) WriteJsonTo(buf *bytes.Buffer) {
	var devices []string

	buf.WriteString(`{`)
	cnt := 0
	for device, usage := range *dusage {
		if cnt > 0 {
			buf.WriteString(`,`)
		}
		cnt++
		fmt.Fprintf(buf, `"%s":`, device)
		usage.WriteJsonTo(buf)

		if device != "total" {
			devices = append(devices, device)
		}
	}
	bytes, err := json.Marshal(devices)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(buf, `,"devices":%s}`, string(bytes))
}

func avgDelta(v int64, w int64, interval float64) float64 {
	ret := float64(w-v) / interval
	return ret
}

func GetDiskUsage(t1 time.Time, d1 *DiskStat, t2 time.Time, d2 *DiskStat) (*DiskUsage, error) {
	interval := float64(t2.Sub(t1).Seconds())

	if interval <= 0.0 {
		return nil, errors.New("negative interval")
	}

	if len(d1.Entries) == 0 || len(d2.Entries) == 0 {
		return nil, errors.New("no DiskEntry")
	}

	usage := new(DiskUsage)
	(*usage) = make(DiskUsage)
	total := new(DiskUsageEntry)

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

		cnt++
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
			avgDelta(entry1.RdIos, entry2.RdIos, interval),
			avgDelta(entry1.WrIos, entry2.WrIos, interval),
			avgDelta(entry1.RdSectors, entry2.RdSectors, interval),
			avgDelta(entry1.WrSectors, entry2.WrSectors, interval),
			rd_latency,
			wr_latency,
			avg_rd_sz,
			avg_wr_sz,
			float64(entry2.ReqTicks-entry1.ReqTicks) / interval / 1.0e3,
		}
		(*usage)[name] = entry

		total.RdIops += entry.RdIops
		total.WrIops += entry.WrIops
		total.RdSecps += entry.RdSecps
		total.WrSecps += entry.WrSecps
		total.RdLatency += entry.RdLatency
		total.WrLatency += entry.WrLatency
		total.AvgRdSize += entry.AvgRdSize
		total.AvgWrSize += entry.AvgWrSize
		total.ReqQlen += entry.ReqQlen
	}

	total.RdLatency /= float64(cnt)
	total.WrLatency /= float64(cnt)
	total.AvgRdSize /= float64(cnt)
	total.AvgWrSize /= float64(cnt)

	(*usage)["total"] = total

	return usage, nil
}

func usageItem(v1 int64, v2 int64, itv int64) float64 {
	return float64(v2-v1) / float64(itv) * 100.0
}
