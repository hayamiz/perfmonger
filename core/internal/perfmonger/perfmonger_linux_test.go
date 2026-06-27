package perfmonger

import (
	"log"
	"os"
	"strings"
	"testing"
)

func TestParseNetStatShortLine(t *testing.T) {
	// /proc/net/dev-like input containing an empty line and a line
	// shorter than 7 bytes, which must not crash the parser.
	input := "Inter-|   Receive                                                |  Transmit\n" +
		" face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n" +
		"\n" +
		"abc\n" +
		"    lo:  1000      10    0    0    0     0          0         0     1000      10    0    0    0     0       0          0\n"

	record := NewStatRecord()
	err := parseNetStat(record, strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseNetStat should not return an error on short lines: %v", err)
	}
	if record.Net == nil {
		t.Fatal("record.Net should not be nil")
	}

	lo_found := false
	for _, entry := range record.Net.Entries {
		if entry.Name == "lo" {
			lo_found = true
			break
		}
	}
	if !lo_found {
		t.Error("Valid device 'lo' should still be parsed despite short lines.")
	}
}

func TestParseDiskStatsShortFormat(t *testing.T) {
	// A 7-field (old/short) /proc/diskstats line. fmt.Sscanf requesting 14
	// fields returns (7, io.EOF) for such a line; the parser must treat the
	// io.EOF as a valid partial-read outcome and parse the line instead of
	// failing the whole read.
	input := "   1    0 ram0 100 200 300 400\n"

	record := NewStatRecord()
	targets := map[string]bool{"ram0": true}
	err := parseDiskStats(record, strings.NewReader(input), &targets)
	if err != nil {
		t.Fatalf("parseDiskStats should not return an error on short-format lines: %v", err)
	}
	if record.Disk == nil {
		t.Fatal("record.Disk should not be nil")
	}

	ram0_found := false
	for _, entry := range record.Disk.Entries {
		if entry.Name == "ram0" {
			ram0_found = true
			break
		}
	}
	if !ram0_found {
		t.Error("Short-format device 'ram0' should be parsed.")
	}
}

func TestReadDiskStat(t *testing.T) {
	var err error
	var stat_record *StatRecord = nil

	err = ReadDiskStats(stat_record, nil)
	if err == nil {
		t.Errorf("Error should be returned with nil *StatRecord.")
	}

	_, err = os.Stat("/proc/diskstats")
	if err != nil {
		t.Skip("/proc/diskstats is not present.")
	}

	stat_record = NewStatRecord()
	err = ReadDiskStats(stat_record, nil)
	if err != nil {
		t.Error("Error should not be returned with valid *StatRecord")
	}
	if stat_record.Disk == nil {
		t.Error("stat_record.Disk should not be nil")
		return
	}
	if len(stat_record.Disk.Entries) == 0 {
		t.Error("No device found.")
	}
}

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

func TestReadMemStat(t *testing.T) {
	var err error
	var stat_record *StatRecord = nil

	err = ReadMemStat(stat_record)
	if err == nil {
		t.Errorf("Error should not be returned with non-nil *StatRecord.")
	}

	_, err = os.Stat("/proc/meminfo")
	if err != nil {
		t.Skip("/proc/meminfo is not present.")
	}

	stat_record = NewStatRecord()
	err = ReadMemStat(stat_record)
	if err != nil {
		log.Print(err)
		t.Error("Error should not be returned with valid *StatRecord.")
		return
	}
	if stat_record.Mem == nil {
		t.Error("stat_record.Mem should not be nil")
		return
	}

	if stat_record.Mem.MemTotal == 0 {
		t.Error("Cannot read MemTotal correctly")
		return
	}

	mem := stat_record.Mem

	if (mem.MemFree + mem.Cached + mem.Buffers) > mem.MemTotal {
		t.Error("Inconsistent meminfo values")
		return
	}
}
