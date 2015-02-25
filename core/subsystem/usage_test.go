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
