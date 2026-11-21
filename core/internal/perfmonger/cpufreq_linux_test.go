package perfmonger

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// makeFakeCpuDir builds a fake /sys/devices/system/cpu layout. contents[i] is
// written to cpu<i>/cpufreq/scaling_cur_freq; a nil entry leaves cpu<i>
// without a cpufreq directory.
func makeFakeCpuDir(t *testing.T, contents []*string) string {
	t.Helper()
	dir := t.TempDir()
	for i, c := range contents {
		coreDir := filepath.Join(dir, "cpu"+strconv.Itoa(i))
		if err := os.MkdirAll(coreDir, 0755); err != nil {
			t.Fatal(err)
		}
		if c == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Join(coreDir, "cpufreq"), 0755); err != nil {
			t.Fatal(err)
		}
		writeFakeFreq(t, dir, i, *c)
	}
	return dir
}

func writeFakeFreq(t *testing.T, dir string, core int, content string) {
	t.Helper()
	path := filepath.Join(dir, "cpu"+strconv.Itoa(core), "cpufreq", "scaling_cur_freq")
	// os.WriteFile truncates the existing inode, so already-open fds observe
	// the new content, just like a sysfs attribute being re-read.
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func strp(s string) *string { return &s }

func TestParseFreqKHz(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"1200000\n", 1200000, true},
		{"2700000", 2700000, true},
		{"  800000 \n", 800000, true},
		{"0\n", 0, true},
		{"", 0, false},
		{"\n", 0, false},
		{"<unknown>\n", 0, false},
		{"12a00\n", 0, false},
		{"-1\n", 0, false},
		{"1234567890123456789\n", 0, false},
	}
	for _, c := range cases {
		got, ok := parseFreqKHz([]byte(c.in))
		if got != c.want || ok != c.ok {
			t.Errorf("parseFreqKHz(%q) = (%d, %t), want (%d, %t)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseKernelVersion(t *testing.T) {
	cases := []struct {
		in           string
		major, minor int
		ok           bool
	}{
		{"6.8.0-142-generic", 6, 8, true},
		{"5.15.0-91-generic", 5, 15, true},
		{"5.19.0", 5, 19, true},
		{"5.19-rc1", 5, 19, true},
		{"4.18.0-553.el8_10.x86_64", 4, 18, true},
		{"6.1.0-26-amd64", 6, 1, true},
		{"", 0, 0, false},
		{"6", 0, 0, false},
		{"x.y.z", 0, 0, false},
		{"6.rc", 0, 0, false},
	}
	for _, c := range cases {
		major, minor, ok := parseKernelVersion(c.in)
		if major != c.major || minor != c.minor || ok != c.ok {
			t.Errorf("parseKernelVersion(%q) = (%d, %d, %t), want (%d, %d, %t)",
				c.in, major, minor, ok, c.major, c.minor, c.ok)
		}
	}
}

func TestCpuFreqReadIsCheap(t *testing.T) {
	cases := []struct {
		goarch, release string
		want            bool
	}{
		{"amd64", "6.8.0-142-generic", true},
		{"amd64", "5.19.0-50-generic", true},
		{"amd64", "5.18.19", false},
		{"amd64", "5.15.0-91-generic", false},
		{"amd64", "4.18.0-553.el8_10.x86_64", false},
		{"386", "5.4.0", false},
		{"arm64", "5.15.0-1050-aws", true},
		{"amd64", "unparsable", true},
	}
	for _, c := range cases {
		if got := cpuFreqReadIsCheap(c.goarch, c.release); got != c.want {
			t.Errorf("cpuFreqReadIsCheap(%q, %q) = %t, want %t", c.goarch, c.release, got, c.want)
		}
	}
}

func TestCpuFreqReaderReadsFakeSysfs(t *testing.T) {
	dir := makeFakeCpuDir(t, []*string{
		strp("1200000\n"),
		strp("2400000\n"),
		nil, // core without cpufreq
		strp("<unknown>\n"),
	})

	r, err := openCpuFreqReader(dir, 4, "amd64", "6.8.0-142-generic")
	if err != nil {
		t.Fatalf("openCpuFreqReader returned error: %v", err)
	}
	defer r.Close()

	if r.NumCore() != 4 {
		t.Fatalf("NumCore() = %d, want 4", r.NumCore())
	}

	record := NewStatRecord()
	r.Read(record)
	if record.CpuFreq == nil {
		t.Fatal("Read did not populate record.CpuFreq")
	}
	if record.CpuFreq.NumCore != 4 {
		t.Errorf("CpuFreq.NumCore = %d, want 4", record.CpuFreq.NumCore)
	}
	want := []int64{1200000, 2400000, 0, 0}
	for i, w := range want {
		if record.CpuFreq.CoreFreqs[i] != w {
			t.Errorf("first read core %d = %d, want %d", i, record.CpuFreq.CoreFreqs[i], w)
		}
	}

	// A second read must re-read the kept-open files from offset 0 and reuse
	// the record's slice.
	writeFakeFreq(t, dir, 0, "3000000\n")
	writeFakeFreq(t, dir, 1, "garbage\n")
	writeFakeFreq(t, dir, 3, "800000\n")
	prevSlice := record.CpuFreq.CoreFreqs
	r.Read(record)
	want = []int64{3000000, 0, 0, 800000}
	for i, w := range want {
		if record.CpuFreq.CoreFreqs[i] != w {
			t.Errorf("second read core %d = %d, want %d", i, record.CpuFreq.CoreFreqs[i], w)
		}
	}
	if &prevSlice[0] != &record.CpuFreq.CoreFreqs[0] {
		t.Error("Read reallocated CoreFreqs instead of reusing it")
	}
}

func TestCpuFreqReaderUnavailableWithoutCpufreq(t *testing.T) {
	dir := makeFakeCpuDir(t, []*string{nil, nil})

	r, err := openCpuFreqReader(dir, 2, "amd64", "6.8.0")
	if r != nil {
		r.Close()
		t.Fatal("expected nil reader when no core has cpufreq")
	}
	if !errors.Is(err, ErrCpuFreqUnavailable) {
		t.Fatalf("expected ErrCpuFreqUnavailable, got %v", err)
	}
}

func TestCpuFreqReaderUnavailableWhenNoValidValue(t *testing.T) {
	dir := makeFakeCpuDir(t, []*string{strp("<unknown>\n"), strp("")})

	r, err := openCpuFreqReader(dir, 2, "amd64", "6.8.0")
	if r != nil {
		r.Close()
		t.Fatal("expected nil reader when no core yields a valid frequency")
	}
	if !errors.Is(err, ErrCpuFreqUnavailable) {
		t.Fatalf("expected ErrCpuFreqUnavailable, got %v", err)
	}
}

func TestCpuFreqReaderRefusesOldX86Kernel(t *testing.T) {
	dir := makeFakeCpuDir(t, []*string{strp("1200000\n")})

	r, err := openCpuFreqReader(dir, 1, "amd64", "5.15.0-91-generic")
	if r != nil {
		r.Close()
		t.Fatal("expected nil reader on x86 kernel < 5.19")
	}
	var kerr *CpuFreqKernelError
	if !errors.As(err, &kerr) {
		t.Fatalf("expected *CpuFreqKernelError, got %v", err)
	}
	if kerr.Release != "5.15.0-91-generic" {
		t.Errorf("Release = %q", kerr.Release)
	}

	// Without cpufreq the unavailability takes precedence over the kernel
	// check, so callers can stay silent on VMs running old kernels.
	empty := makeFakeCpuDir(t, []*string{nil})
	_, err = openCpuFreqReader(empty, 1, "amd64", "5.15.0-91-generic")
	if !errors.Is(err, ErrCpuFreqUnavailable) {
		t.Fatalf("expected ErrCpuFreqUnavailable without cpufreq, got %v", err)
	}

	// Non-x86 architectures are not affected by the IPI-based implementation.
	r, err = openCpuFreqReader(dir, 1, "arm64", "5.15.0")
	if err != nil {
		t.Fatalf("arm64 on 5.15 should be allowed, got %v", err)
	}
	r.Close()
}

func TestCpuFreqReaderInvalidCoreCount(t *testing.T) {
	_, err := openCpuFreqReader(t.TempDir(), 0, "amd64", "6.8.0")
	if !errors.Is(err, ErrCpuFreqUnavailable) {
		t.Fatalf("expected ErrCpuFreqUnavailable for 0 cores, got %v", err)
	}
}

func TestCpuFreqReaderCloseIsIdempotentAndNilSafe(t *testing.T) {
	var nilReader *CpuFreqReader
	nilReader.Close() // must not panic

	dir := makeFakeCpuDir(t, []*string{strp("1200000\n")})
	r, err := openCpuFreqReader(dir, 1, "amd64", "6.8.0")
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	r.Close()

	// Reading after Close yields "unavailable" rather than touching a stale fd.
	record := NewStatRecord()
	r.Read(record)
	if record.CpuFreq.CoreFreqs[0] != 0 {
		t.Errorf("read after Close = %d, want 0", record.CpuFreq.CoreFreqs[0])
	}
}

// TestCpuFreqReaderRealSysfs exercises the real cpufreq sysfs when the host
// exposes it at low cost; it is skipped elsewhere (VMs, containers, old x86
// kernels).
func TestCpuFreqReaderRealSysfs(t *testing.T) {
	numCore := NumCpuCores()
	r, err := OpenCpuFreqReader(numCore)
	if err != nil {
		t.Skipf("cpufreq not usable on this host: %v", err)
	}
	defer r.Close()

	record := NewStatRecord()
	r.Read(record)
	valid := 0
	for _, khz := range record.CpuFreq.CoreFreqs {
		if khz > 0 {
			valid++
		}
	}
	if valid == 0 {
		t.Fatalf("no core reported a frequency: %v", record.CpuFreq.CoreFreqs)
	}
}

// legacyStatRecord mirrors StatRecord as it was before the CpuFreq field was
// added, to check gob compatibility in both directions.
type legacyStatRecord struct {
	Time      time.Time
	Cpu       *CpuStat
	Interrupt *InterruptStat
	Proc      *ProcStat
	Disk      *DiskStat
	Softirq   *SoftIrqStat
	Net       *NetStat
	Mem       *MemStat
}

func TestStatRecordCpuFreqGobCompatibility(t *testing.T) {
	// New writer -> old reader: the extra field is ignored.
	rec := NewStatRecord()
	rec.Cpu = NewCpuStat(2)
	rec.CpuFreq = NewCpuFreqStat(2)
	rec.CpuFreq.CoreFreqs[0] = 1200000
	rec.CpuFreq.CoreFreqs[1] = 2400000

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(rec); err != nil {
		t.Fatal(err)
	}
	var legacy legacyStatRecord
	if err := gob.NewDecoder(&buf).Decode(&legacy); err != nil {
		t.Fatalf("legacy decoder failed on new record: %v", err)
	}
	if legacy.Cpu == nil || legacy.Cpu.NumCore != 2 {
		t.Errorf("legacy decode lost CPU stats: %+v", legacy.Cpu)
	}

	// New writer -> new reader: the frequencies round-trip.
	buf.Reset()
	if err := gob.NewEncoder(&buf).Encode(rec); err != nil {
		t.Fatal(err)
	}
	var decoded StatRecord
	if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.CpuFreq == nil || decoded.CpuFreq.NumCore != 2 ||
		decoded.CpuFreq.CoreFreqs[0] != 1200000 || decoded.CpuFreq.CoreFreqs[1] != 2400000 {
		t.Errorf("CpuFreq did not round-trip: %+v", decoded.CpuFreq)
	}

	// Old writer -> new reader: CpuFreq stays nil.
	buf.Reset()
	old := legacyStatRecord{Time: time.Now(), Cpu: NewCpuStat(2)}
	if err := gob.NewEncoder(&buf).Encode(&old); err != nil {
		t.Fatal(err)
	}
	var fromOld StatRecord
	if err := gob.NewDecoder(&buf).Decode(&fromOld); err != nil {
		t.Fatal(err)
	}
	if fromOld.CpuFreq != nil {
		t.Errorf("CpuFreq should be nil for a legacy record, got %+v", fromOld.CpuFreq)
	}
}

// BenchmarkCpuFreqReaderRead measures one sweep over all cores of the real
// cpufreq sysfs (the per-sample cost added to the recorder).
func BenchmarkCpuFreqReaderRead(b *testing.B) {
	r, err := OpenCpuFreqReader(NumCpuCores())
	if err != nil {
		b.Skipf("cpufreq not usable on this host: %v", err)
	}
	defer r.Close()
	record := NewStatRecord()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Read(record)
	}
}

// BenchmarkReadCpuStat is the existing /proc/stat sampling cost, for
// comparison with BenchmarkCpuFreqReaderRead.
func BenchmarkReadCpuStat(b *testing.B) {
	record := NewStatRecord()
	if err := ReadCpuStat(record); err != nil {
		b.Skipf("/proc/stat not readable: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadCpuStat(record)
	}
}
