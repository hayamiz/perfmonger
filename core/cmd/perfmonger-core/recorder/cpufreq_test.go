package recorder

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	ss "github.com/hayamiz/perfmonger/core/internal/perfmonger"
)

// stubOpenCpuFreqReader replaces the cpufreq probe for the duration of a test
// and reports how many times it was called.
func stubOpenCpuFreqReader(t *testing.T, err error) *int {
	t.Helper()
	calls := 0
	orig := openCpuFreqReader
	openCpuFreqReader = func() (*ss.CpuFreqReader, error) {
		calls++
		return nil, err
	}
	t.Cleanup(func() { openCpuFreqReader = orig })
	return &calls
}

func TestSetupCpuFreqReaderHonorsNoCpuFreq(t *testing.T) {
	calls := stubOpenCpuFreqReader(t, nil)

	option := NewRecorderOption()
	option.NoCpuFreq = true
	var logbuf bytes.Buffer
	if r := setupCpuFreqReader(option, &logbuf); r != nil {
		t.Fatal("expected nil reader with --no-cpufreq")
	}
	if *calls != 0 {
		t.Errorf("probe was called %d times despite --no-cpufreq", *calls)
	}
	if logbuf.Len() != 0 {
		t.Errorf("unexpected log output: %q", logbuf.String())
	}
}

func TestSetupCpuFreqReaderKernelErrorNotifies(t *testing.T) {
	stubOpenCpuFreqReader(t, &ss.CpuFreqKernelError{Release: "5.15.0-91-generic"})

	option := NewRecorderOption()
	var logbuf bytes.Buffer
	if r := setupCpuFreqReader(option, &logbuf); r != nil {
		t.Fatal("expected nil reader on kernel error")
	}
	if !strings.Contains(logbuf.String(), "per-core CPU frequency recording disabled") ||
		!strings.Contains(logbuf.String(), "5.15.0-91-generic") {
		t.Errorf("expected a notice naming the kernel, got %q", logbuf.String())
	}
}

func TestSetupCpuFreqReaderUnavailableIsSilentUnlessDebug(t *testing.T) {
	stubOpenCpuFreqReader(t, fmt.Errorf("%w: test", ss.ErrCpuFreqUnavailable))

	option := NewRecorderOption()
	var logbuf bytes.Buffer
	if r := setupCpuFreqReader(option, &logbuf); r != nil {
		t.Fatal("expected nil reader when cpufreq is unavailable")
	}
	if logbuf.Len() != 0 {
		t.Errorf("expected no output without debug, got %q", logbuf.String())
	}

	option.Debug = true
	setupCpuFreqReader(option, &logbuf)
	if !strings.Contains(logbuf.String(), "per-core CPU frequency recording disabled") {
		t.Errorf("expected the reason in debug mode, got %q", logbuf.String())
	}
}

// recordAndDecode runs a short recording into a temporary file and returns the
// decoded sample records.
func recordAndDecode(t *testing.T, option *RecorderOption) []ss.StatRecord {
	t.Helper()
	tmpfile := path.Join(t.TempDir(), "out.pgr")
	option.Output = tmpfile
	option.Timeout = 30 * time.Millisecond
	option.Interval = 10 * time.Millisecond
	option.NoIntervalBackoff = true
	option.NoIntr = true
	option.NoDisk = true

	if err := RunDirect(option); err != nil {
		t.Fatalf("RunDirect failed: %v", err)
	}

	f, err := os.Open(tmpfile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var cheader ss.CommonHeader
	var pheader ss.PlatformHeader
	if err := dec.Decode(&cheader); err != nil {
		t.Fatal(err)
	}
	if err := dec.Decode(&pheader); err != nil {
		t.Fatal(err)
	}
	var records []ss.StatRecord
	for {
		// Decode into a fresh value each time: gob leaves fields that are
		// absent from the stream untouched in a reused destination.
		var rec ss.StatRecord
		err := dec.Decode(&rec)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, rec)
	}
	if len(records) == 0 {
		t.Fatal("no records decoded")
	}
	return records
}

func TestRunDirectRecordsCpuFreqWhenAvailable(t *testing.T) {
	probe, err := ss.OpenCpuFreqReader(ss.NumCpuCores())
	if err != nil {
		t.Skipf("cpufreq not usable on this host: %v", err)
	}
	probe.Close()

	records := recordAndDecode(t, NewRecorderOption())
	for i, rec := range records {
		if rec.CpuFreq == nil {
			t.Fatalf("record %d has no CpuFreq", i)
		}
		if rec.CpuFreq.NumCore != ss.NumCpuCores() || len(rec.CpuFreq.CoreFreqs) != rec.CpuFreq.NumCore {
			t.Fatalf("record %d: NumCore=%d len=%d, want %d",
				i, rec.CpuFreq.NumCore, len(rec.CpuFreq.CoreFreqs), ss.NumCpuCores())
		}
	}
}

func TestRunDirectNoCpuFreqOmitsFrequencies(t *testing.T) {
	option := NewRecorderOption()
	option.NoCpuFreq = true
	for i, rec := range recordAndDecode(t, option) {
		if rec.CpuFreq != nil {
			t.Fatalf("record %d has CpuFreq despite NoCpuFreq", i)
		}
		if rec.Cpu == nil {
			t.Fatalf("record %d lost CPU usage stats", i)
		}
	}
}

func TestRunDirectWithoutCpuFreqSupportKeepsCpuStats(t *testing.T) {
	stubOpenCpuFreqReader(t, fmt.Errorf("%w: test", ss.ErrCpuFreqUnavailable))

	for i, rec := range recordAndDecode(t, NewRecorderOption()) {
		if rec.CpuFreq != nil {
			t.Fatalf("record %d has CpuFreq although the probe failed", i)
		}
		if rec.Cpu == nil {
			t.Fatalf("record %d lost CPU usage stats", i)
		}
	}
}
