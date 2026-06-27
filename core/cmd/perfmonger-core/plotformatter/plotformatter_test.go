package plotformatter

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ss "github.com/hayamiz/perfmonger/core/internal/perfmonger"
)

// findRepoRoot walks up from the current directory to locate the repository
// root (identified by the presence of a go.mod / spec directory).
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "spec", "data", "busy100.pgr")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repository root from working directory")
		}
		dir = parent
	}
}

// TestRunPlotFormatClosesTmpFilesOnce verifies that each disk_dat / cpu_dat
// temp file is closed exactly once during runPlotFormat. A double-close is an
// FD-reuse hazard (the second Close could close an unrelated, recycled fd).
func TestRunPlotFormatClosesTmpFilesOnce(t *testing.T) {
	root := findRepoRoot(t)
	pgr := filepath.Join(root, "spec", "data", "busy100.pgr")

	// Count how many times each *os.File is closed via the temp-file close
	// seam used by runPlotFormat.
	closeCounts := map[*os.File]int{}
	orig := closeTmpFile
	closeTmpFile = func(f *os.File) error {
		closeCounts[f]++
		return orig(f)
	}
	defer func() { closeTmpFile = orig }()

	tmpDir := t.TempDir()
	opt := &CmdOption{
		DiskFile:       filepath.Join(tmpDir, "disk.dat"),
		CpuFile:        filepath.Join(tmpDir, "cpu.dat"),
		MemFile:        filepath.Join(tmpDir, "mem.dat"),
		PerfmongerFile: pgr,
	}

	_, err := runPlotFormat(opt)
	if err != nil {
		t.Fatalf("runPlotFormat failed: %v", err)
	}

	if len(closeCounts) == 0 {
		t.Fatalf("expected at least one temp file to be closed via the close seam")
	}

	for f, n := range closeCounts {
		if n != 1 {
			t.Errorf("temp file %q closed %d times, want exactly 1", f.Name(), n)
		}
	}
}

// TestRunPlotFormatNilCpuReturnsError verifies that a malformed log whose first
// record has a nil Cpu field yields a clean error instead of a nil-pointer
// dereference panic. StatRecord.Cpu is a *CpuStat, so a partially-written log
// can decode into a record with Cpu == nil.
func TestRunPlotFormatNilCpuReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nil-cpu.pgr")

	// Write a valid header pair followed by a first record with a nil Cpu.
	f, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create log: %v", err)
	}
	enc := gob.NewEncoder(f)
	if err := enc.Encode(&ss.CommonHeader{StartTime: time.Now()}); err != nil {
		t.Fatalf("encode common header: %v", err)
	}
	if err := enc.Encode(&ss.PlatformHeader{}); err != nil {
		t.Fatalf("encode platform header: %v", err)
	}
	// First record: Cpu is left nil.
	if err := enc.Encode(&ss.StatRecord{Time: time.Now()}); err != nil {
		t.Fatalf("encode record: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close log: %v", err)
	}

	opt := &CmdOption{
		DiskFile:       filepath.Join(tmpDir, "disk.dat"),
		CpuFile:        filepath.Join(tmpDir, "cpu.dat"),
		MemFile:        filepath.Join(tmpDir, "mem.dat"),
		PerfmongerFile: logPath,
	}

	_, err = runPlotFormat(opt)
	if err == nil {
		t.Fatalf("runPlotFormat returned nil error for a record with nil Cpu")
	}
}

// TestRunPlotFormatPropagatesFlushError verifies that when a bufio.Writer.Flush
// fails (e.g. underlying I/O error / full disk), runPlotFormat surfaces the
// error instead of silently dropping the buffered data and returning nil.
func TestRunPlotFormatPropagatesFlushError(t *testing.T) {
	root := findRepoRoot(t)
	pgr := filepath.Join(root, "spec", "data", "busy100.pgr")

	wantErr := fmt.Errorf("injected flush failure")
	orig := flushWriter
	flushWriter = func(w *bufio.Writer) error {
		return wantErr
	}
	defer func() { flushWriter = orig }()

	tmpDir := t.TempDir()
	opt := &CmdOption{
		DiskFile:       filepath.Join(tmpDir, "disk.dat"),
		CpuFile:        filepath.Join(tmpDir, "cpu.dat"),
		MemFile:        filepath.Join(tmpDir, "mem.dat"),
		PerfmongerFile: pgr,
	}

	_, err := runPlotFormat(opt)
	if err == nil {
		t.Fatalf("runPlotFormat returned nil error despite a failing Flush")
	}
	if !strings.Contains(err.Error(), wantErr.Error()) {
		t.Errorf("error %q does not wrap the underlying flush error %q", err, wantErr)
	}
}
