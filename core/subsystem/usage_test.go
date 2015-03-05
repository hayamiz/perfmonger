package subsystem

import (
	"bytes"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func isValidJson(byt []byte) bool {
	var val interface{}
	err := json.Unmarshal(byt, &val)
	return err == nil
}

func jsonHasKey(byt []byte, key_path string) bool {
	var val interface{}
	err := json.Unmarshal(byt, &val)
	if err != nil {
		return false
	}

	keys := strings.Split(key_path, ".")

	array_re := regexp.MustCompile("^\\[(\\d+)\\]$")

	for _, key := range keys {
		var idx int = -1
		m := array_re.FindStringSubmatch(key)
		if len(m) > 0 {
			_i, err := strconv.ParseInt(m[1], 10, 32)
			if err != nil {
				return false
			}
			idx = int(_i)
		}

		switch val.(type) {
		case map[string]interface{}:
			v, ok := val.(map[string]interface{})[key]
			if !ok {
				return false
			}
			val = v
		case []interface{}:
			if idx < 0 {
				return false
			}
			val = val.([]interface{})[idx]
		default:
			return false
		}
	}

	return true
}

func floatEqWithin(val1, val2, epsilon float64) bool {
	return math.Abs(val1-val2) < epsilon
}

func TestGetCoreUsage(t *testing.T) {
	var err error

	c1 := new(CpuCoreStat)
	c2 := new(CpuCoreStat)

	c1.User = 10
	c2.User = 5
	c2.Sys = 5

	// should return error if uptime is the same
	_, err = GetCpuCoreUsage(c1, c2)
	if err == nil {
		t.Error("err == nil, want non-nil")
	}

	// should return error if c1.Uptime() is larger than c2.Uptime()
	c1.User = 100
	_, err = GetCpuCoreUsage(c1, c2)
	if err == nil {
		t.Error("err == nil, want non-nil")
	}

	// should return 75% usr and 25% sys usage
	var usage *CpuCoreUsage
	c1.Clear()
	c2.Clear()
	c1.User = 100
	c1.Sys = 50
	c2.User = c1.User + 300
	c2.Sys = c1.Sys + 100
	usage, err = GetCpuCoreUsage(c1, c2)
	if err != nil {
		t.Errorf("err == %v, want nil", err)
	}
	if usage == nil {
		t.Error("usage == nil, want non-nil")
	}
	if !floatEqWithin(usage.User, 75.0, 0.01) {
		t.Errorf("usage.User = %v, want 75.0", usage.User)
	}
	if !floatEqWithin(usage.Sys, 25.0, 0.01) {
		t.Errorf("usage.Sys = %v, want 25.0", usage.User)
	}

	buf := bytes.NewBuffer([]byte{})
	usage.WriteJsonTo(buf)
	if !isValidJson(buf.Bytes()) {
		t.Errorf("Invalid JSON: %s", buf.String())
	}
}

func TestGetCpuUsage(t *testing.T) {
	var err error
	var usage *CpuUsage

	num_core := 2
	c1 := NewCpuStat(num_core)
	c2 := NewCpuStat(num_core)

	usage, err = GetCpuUsage(c1, c2)
	if err == nil {
		t.Error("Error should be returned because no difference between c1 and c2")
	}
	if usage != nil {
		t.Error("Nil should be returned as usage")
	}

	c1.CoreStats[0].User = 100
	c1.CoreStats[0].Sys = 50
	c1.CoreStats[1].User = 100
	c1.CoreStats[1].Sys = 50

	c2.CoreStats[0].User = c1.CoreStats[0].User + 300
	c2.CoreStats[0].Sys = c1.CoreStats[0].Sys + 100
	c2.CoreStats[1].User = c1.CoreStats[0].User + 300
	c2.CoreStats[1].Sys = c1.CoreStats[0].Sys + 100

	usage, err = GetCpuUsage(c1, c2)
	if err == nil {
		t.Error("Error should be returned because no progress in .All uptime")
	}
	if usage != nil {
		t.Error("Nil should be returned as usage")
	}

	c1.All.User = 200
	c1.All.Sys = 100
	c2.All.User = c1.All.User + 600
	c2.All.Sys = c1.All.Sys + 200

	c1.CoreStats[0].Clear()
	c2.CoreStats[0].Clear()

	usage, err = GetCpuUsage(c1, c2)
	if err == nil {
		t.Error("Error should be returned because no progress in .CoreStats[9] uptime")
	}
	if usage != nil {
		t.Error("Nil should be returned as usage")
	}

	c1.CoreStats[0].User = 100
	c1.CoreStats[0].Sys = 50
	c2.CoreStats[0].User = c1.CoreStats[0].User + 300
	c2.CoreStats[0].Sys = c1.CoreStats[0].Sys + 100
	usage, err = GetCpuUsage(c1, c2)
	if err != nil {
		t.Error("Error should not be returned.")
	}
	if usage == nil {
		t.Error("usage == nil, want non-nil")
	}
	if !floatEqWithin(usage.All.User, 150.0, 0.01) {
		t.Errorf("usage.User = %v, want 150.0", usage.All.User)
	}
	if !floatEqWithin(usage.All.Sys, 50.0, 0.01) {
		t.Errorf("usage.Sys = %v, want 50.0", usage.All.User)
	}
	if !floatEqWithin(usage.CoreUsages[0].User, 75.0, 0.01) {
		t.Errorf("usage.User = %v, want 75.0", usage.CoreUsages[0].User)
	}
	if !floatEqWithin(usage.CoreUsages[0].Sys, 25.0, 0.01) {
		t.Errorf("usage.Sys = %v, want 25.0", usage.CoreUsages[0].User)
	}
	if !floatEqWithin(usage.CoreUsages[1].User, 75.0, 0.01) {
		t.Errorf("usage.User = %v, want 75.0", usage.CoreUsages[1].User)
	}
	if !floatEqWithin(usage.CoreUsages[1].Sys, 25.0, 0.01) {
		t.Errorf("usage.Sys = %v, want 25.0", usage.CoreUsages[1].User)
	}

	buf := bytes.NewBuffer([]byte{})
	usage.WriteJsonTo(buf)
	if !isValidJson(buf.Bytes()) {
		t.Errorf("Invalid JSON: %s", buf.String())
	}

	assertHasKey := func(key_path string) {
		if !jsonHasKey(buf.Bytes(), key_path) {
			t.Errorf("%v is not present in JSON:\n%v", key_path, buf.String())
		}
	}
	assertHasKey("num_core")
	assertHasKey("cores")
	assertHasKey("cores.[0]")
	assertHasKey("cores.[0].usr")
	assertHasKey("cores.[0].sys")
	assertHasKey("cores.[0].nice")
	assertHasKey("cores.[0].idle")
	assertHasKey("cores.[0].iowait")
	assertHasKey("cores.[0].hardirq")
	assertHasKey("cores.[0].softirq")
	assertHasKey("cores.[0].steal")
	assertHasKey("cores.[0].guest")
	assertHasKey("cores.[0].guestnice")
}

func TestDiskUsage(t *testing.T) {
	d1 := NewDiskStat()
	d2 := NewDiskStat()
	t1, perr := time.Parse(time.RFC3339, "2012-01-23T01:23:45+09:00")
	t2 := t1
	if perr != nil {
		t.Error("Timestamp parse error")
	}

	_, err := GetDiskUsage(t1, d1, t2, d2)
	if err == nil {
		t.Error("Error should be returned because timestamps are the same")
	}

	interval_duration := time.Second * 2
	interval := interval_duration.Seconds()
	t2 = t1.Add(interval_duration)

	_, err = GetDiskUsage(t1, d1, t2, d2)
	if err == nil {
		t.Error("Error should be returned because no entries in DiskStat")
	}

	d1.Entries = append(d1.Entries, NewDiskStatEntry())
	d1.Entries[0].Name = "sda"
	d1.Entries[0].RdIos = 100
	d1.Entries[0].RdSectors = 200
	d1.Entries[0].RdTicks = 500

	d2.Entries = append(d2.Entries, NewDiskStatEntry())
	d2.Entries[0].Name = "sda"
	d2.Entries[0].RdIos = d1.Entries[0].RdIos + 200
	d2.Entries[0].RdSectors = d1.Entries[0].RdSectors + 150
	d2.Entries[0].RdTicks = d1.Entries[0].RdTicks + 1000

	var usage *DiskUsage
	usage, err = GetDiskUsage(t1, d1, t2, d2)
	if err != nil {
		t.Error("Error should be returned.")
	}
	_, sda_ok := (*usage)["sda"]
	if len(*usage) != 1 || !sda_ok {
		t.Errorf("DiskUsage = %v, want 1 entry 'sda'", *usage)
	}
	if (*usage)["sda"].Interval != interval_duration {
		t.Errorf("sda.Interval = %v, want %v", (*usage)["sda"].Interval, interval_duration)
	}
	if !floatEqWithin((*usage)["sda"].RdIops, 200.0/interval, 0.001) {
		t.Errorf("sda.RdIops = %v, want %v", (*usage)["sda"].RdIops, 200.0/interval)
	}
	if !floatEqWithin((*usage)["sda"].RdLatency, 1000.0/200.0, 0.001) {
		t.Errorf("sda.RdLatency = %v, want %v", (*usage)["sda"].RdLatency, 1000.0/200.0)
	}
	if (*usage)["sda"].RdSectors != 150 {
		t.Errorf("sda.RdSectors = %v, want %v", (*usage)["sda"].RdSectors, 350)
	}

	buf := bytes.NewBuffer([]byte{})
	usage.WriteJsonTo(buf)
	if !isValidJson(buf.Bytes()) {
		t.Errorf("invalid json: %s", buf.String())
	}

	d1.Entries = append(d1.Entries, NewDiskStatEntry())
	d1.Entries[1].Name = "sdb"
	d1.Entries[1].RdIos = 200
	d1.Entries[1].RdSectors = 400
	d1.Entries[1].RdTicks = 10000

	d2.Entries = append(d2.Entries, NewDiskStatEntry())
	d2.Entries[1].Name = "sdb"
	d2.Entries[1].RdIos = d1.Entries[1].RdIos + 300
	d2.Entries[1].RdSectors = d1.Entries[1].RdSectors + 200
	d2.Entries[1].RdTicks = d1.Entries[1].RdTicks + 1000

	usage, err = GetDiskUsage(t1, d1, t2, d2)
	if err != nil {
		t.Error("Error should not be returned.")
	}
	_, sda_ok = (*usage)["sda"]
	_, sdb_ok := (*usage)["sda"]
	_, total_ok := (*usage)["total"]
	if len(*usage) != 3 || !sda_ok || !sdb_ok || !total_ok {
		t.Errorf("DiskUsage = %v, want 3 entries 'sda', 'sdb' and 'total'.",
			*usage)
	}
	if !floatEqWithin((*usage)["sdb"].RdIops, 300.0/interval, 0.001) {
		t.Errorf("sdb.RdIops = %v, want %v", (*usage)["sdb"].RdIops, 300.0/interval)
	}
	if !floatEqWithin((*usage)["sdb"].RdLatency, 1000.0/300.0, 0.001) {
		t.Errorf("sdb.RdLatency = %v, want %v", (*usage)["sdb"].RdLatency, 1000.0/300.0)
	}
	if (*usage)["sdb"].RdSectors != 200.0 {
		t.Errorf("sdb.RdSectors = %v, want %v", (*usage)["sdb"].RdSectors, 200)
	}

	if !floatEqWithin(
		(*usage)["sda"].RdIops+(*usage)["sdb"].RdIops,
		(*usage)["total"].RdIops, 0.001) {
		t.Errorf("sda.RdIops+sdb.RdIops = %v, total.RdIops = %v, want %v",
			(*usage)["sda"].RdIops+(*usage)["sdb"].RdIops, (*usage)["total"].RdIops,
			(200.0+300.0)/interval)
	}
	weighted_latency := (*usage)["sda"].RdLatency*(200.0/(200.0+300.0)) + (*usage)["sdb"].RdLatency*(300.0/(200.0+300.0))
	if !floatEqWithin(weighted_latency, (*usage)["total"].RdLatency, 0.001) {
		t.Errorf("weighted avg latency(sda+sdb) = %v, total.RdLatency = %v, want %v",
			weighted_latency, (*usage)["total"].RdLatency,
			(2.0/5.0)*1000.0/200.0+(3.0/5.0)*1000.0/300.0)
	}
	if (*usage)["total"].RdSectors != 350 {
		t.Errorf("total.RdSectors = %v, want %v", (*usage)["total"].RdSectors, 350)
	}

	buf = bytes.NewBuffer([]byte{})
	usage.WriteJsonTo(buf)
	if !isValidJson(buf.Bytes()) {
		t.Errorf("invalid json: %s", buf.String())
	}

	assertHasKey := func(key_path string) {
		if !jsonHasKey(buf.Bytes(), key_path) {
			t.Errorf("%v is not present in JSON:\n%v", key_path, buf.String())
		}
	}
	assertHasKey("devices")
	assertHasKey("sda")
	assertHasKey("sda.riops")
	assertHasKey("sda.wiops")
	assertHasKey("sda.rkbyteps")
	assertHasKey("sda.wkbyteps")
	assertHasKey("sda.rlatency")
	assertHasKey("sda.wlatency")
	assertHasKey("sda.rsize")
	assertHasKey("sda.wsize")
	assertHasKey("sda.qlen")
	assertHasKey("total")
}

func TestGetNetUsage(t *testing.T) {
	n1 := NewNetStat()
	n2 := NewNetStat()

	t1, perr := time.Parse(time.RFC3339, "2012-01-23T01:23:45+09:00")
	t2 := t1
	if perr != nil {
		t.Error("Timestamp parse error")
	}

	_, err := GetNetUsage(t1, n1, t2, n2)
	if err == nil {
		t.Error("Error should be returned because timestamps are the same")
	}

	interval_duration := time.Second * 2
	interval := interval_duration.Seconds()
	t2 = t1.Add(interval_duration)

	_, err = GetNetUsage(t1, n1, t2, n2)
	if err == nil {
		t.Error("Error should be returned because no entries in NetStat")
	}

	n1.Entries = append(n1.Entries, NewNetStatEntry())
	n1.Entries[0].Name = "lo"
	n1.Entries[0].RxBytes = 12345
	n1.Entries[0].RxPackets = 12345

	n2.Entries = append(n2.Entries, NewNetStatEntry())
	n2.Entries[0].Name = "lo"
	n2.Entries[0].RxBytes = n1.Entries[0].RxBytes + 8000
	n2.Entries[0].RxPackets = n1.Entries[0].RxPackets + 100

	var usage *NetUsage
	usage, err = GetNetUsage(t1, n1, t2, n2)
	if err != nil {
		t.Errorf("Error should not be returned with:\n  n1 = %v\n  n2 = %v",
			n1, n2)
	}
	_, lo_ok := (*usage)["lo"]
	if len(*usage) != 1 || !lo_ok {
		t.Errorf("NetUsage = %v, want 1 entry \"lo\"", usage)
	}

	if (*usage)["lo"].Interval != interval_duration {
		t.Errorf("lo.Interval = %v, want %v", (*usage)["lo"].Interval, interval_duration)
	}
	if !floatEqWithin((*usage)["lo"].RxBytesPerSec, 8000.0/interval, 0.001) {
		t.Errorf("lo.RxBytesPerSec = %v, want %v", (*usage)["lo"].RxBytesPerSec, 8000.0/interval)
	}
	if !floatEqWithin((*usage)["lo"].RxPacketsPerSec, 100.0/interval, 0.001) {
		t.Errorf("lo.RxPacketsPerSec = %v, want %v", (*usage)["lo"].RxPacketsPerSec, 100.0/interval)
	}

	buf := bytes.NewBuffer([]byte{})
	usage.WriteJsonTo(buf)
	if !isValidJson(buf.Bytes()) {
		t.Errorf("invalid json: %s", buf.String())
	}

	n1.Entries = append(n1.Entries, NewNetStatEntry())
	n1.Entries[1].Name = "eth0"
	n1.Entries[1].RxBytes = 45678
	n1.Entries[1].RxPackets = 123

	usage, err = GetNetUsage(t1, n1, t2, n2)
	if err != nil {
		t.Errorf("Error should not be returned with:\n  n1 = %v\n  n2 = %v",
			n1, n2)
	}
	_, lo_ok = (*usage)["lo"]
	if len(*usage) != 1 || !lo_ok {
		t.Errorf(`NetUsage = %v, want 1 entry "lo".`, usage)
	}

	n2.Entries = append(n2.Entries, NewNetStatEntry())
	n2.Entries[1].Name = "eth0"
	n2.Entries[1].RxBytes = n1.Entries[1].RxBytes + 7000
	n2.Entries[1].RxPackets = n1.Entries[1].RxPackets + 150

	usage, err = GetNetUsage(t1, n1, t2, n2)
	if err != nil {
		t.Errorf("Error should not be returned with:\n  n1 = %v\n  n2 = %v",
			n1, n2)
	}
	_, lo_ok = (*usage)["lo"]
	_, eth0_ok := (*usage)["eth0"]
	_, total_ok := (*usage)["total"]
	if len(*usage) != 3 || !lo_ok || !eth0_ok || !total_ok {
		t.Errorf(`NetUsage = %v, want 3 entry "lo", "eth0" and "total".`, usage)
	}

	if !floatEqWithin((*usage)["eth0"].RxBytesPerSec, 7000.0/interval, 0.001) {
		t.Errorf("eth0.RxBytesPerSec = %v, want %v", (*usage)["eth0"].RxBytesPerSec, 7000.0/interval)
	}
	if !floatEqWithin((*usage)["eth0"].RxPacketsPerSec, 150.0/interval, 0.001) {
		t.Errorf("eth0.RxPacketsPerSec = %v, want %v", (*usage)["eth0"].RxPacketsPerSec, 150.0/interval)
	}

	if !floatEqWithin((*usage)["total"].RxBytesPerSec, (8000.0+7000.0)/interval, 0.001) {
		t.Errorf("total.RxBytesPerSec = %v, want %v", (*usage)["total"].RxBytesPerSec,
			(8000.0+7000.0)/interval)
	}
	if !floatEqWithin((*usage)["total"].RxPacketsPerSec, (100.0+150.0)/interval, 0.001) {
		t.Errorf("total.RxPacketsPerSec = %v, want %v", (*usage)["total"].RxPacketsPerSec,
			(100.0+150.0)/interval)
	}

	buf = bytes.NewBuffer([]byte{})
	usage.WriteJsonTo(buf)
	if !isValidJson(buf.Bytes()) {
		t.Errorf("invalid json: %s", buf.String())
	}

	assertHasKey := func(key_path string) {
		if !jsonHasKey(buf.Bytes(), key_path) {
			t.Errorf("%v is not present in JSON:\n%v", key_path, buf.String())
		}
	}
	assertHasKey("devices")
	assertHasKey("devices.[0]")
	assertHasKey("eth0")
	assertHasKey("eth0.rxkbyteps")
	assertHasKey("eth0.txkbyteps")
	assertHasKey("eth0.rxpktps")
	assertHasKey("eth0.txpktps")
	assertHasKey("eth0.rxerrps")
	assertHasKey("eth0.txerrps")
	assertHasKey("eth0.rxdropps")
	assertHasKey("eth0.txdropps")
}
