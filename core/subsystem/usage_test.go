package subsystem

import (
	"math"
	"testing"
)

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
	if math.Abs(usage.User-75.0) > 0.01 {
		t.Errorf("usage.User = %v, want 75.0", usage.User)
	}
	if math.Abs(usage.Sys-25.0) > 0.01 {
		t.Errorf("usage.Sys = %v, want 25.0", usage.User)
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
	if math.Abs(usage.All.User-150.0) > 0.01 {
		t.Errorf("usage.User = %v, want 150.0", usage.All.User)
	}
	if math.Abs(usage.All.Sys-50.0) > 0.01 {
		t.Errorf("usage.Sys = %v, want 50.0", usage.All.User)
	}
	if math.Abs(usage.CoreUsages[0].User-75.0) > 0.01 {
		t.Errorf("usage.User = %v, want 75.0", usage.CoreUsages[0].User)
	}
	if math.Abs(usage.CoreUsages[0].Sys-25.0) > 0.01 {
		t.Errorf("usage.Sys = %v, want 25.0", usage.CoreUsages[0].User)
	}
	if math.Abs(usage.CoreUsages[1].User-75.0) > 0.01 {
		t.Errorf("usage.User = %v, want 75.0", usage.CoreUsages[1].User)
	}
	if math.Abs(usage.CoreUsages[1].Sys-25.0) > 0.01 {
		t.Errorf("usage.Sys = %v, want 25.0", usage.CoreUsages[1].User)
	}
}
