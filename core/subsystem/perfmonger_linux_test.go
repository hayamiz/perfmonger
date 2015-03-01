package subsystem

import (
	"os"
	"testing"
)

func TestReadNetStat(t *testing.T) {
	var err error
	var stat_record *StatRecord = nil

	err = ReadNetStat(stat_record)
	if err == nil {
		t.Errorf("Error should be returned with nil *StatRecord.")
	}

	_, err = os.Stat("/proc/net/dev")
	if err != nil {
		t.Skip("/proc/net/dev is not present.")
	}

	stat_record = NewStatRecord()
	err = ReadNetStat(stat_record)
	if err != nil {
		t.Error("Error should not be returned with valid *StatRecord.")
	}
	if stat_record.Net == nil {
		t.Error("stat_record.Net should not be nil")
		return
	}
	if len(stat_record.Net.Entries) == 0 {
		t.Error("No net device found, though every linux host should have 'lo'.")
	}
	// search 'lo'
	lo_found := false
	for _, entry := range stat_record.Net.Entries {
		if entry.Name == "lo" {
			lo_found = true
			break
		}
	}
	if !lo_found {
		t.Error("Device 'lo' not found.")
	}
}
