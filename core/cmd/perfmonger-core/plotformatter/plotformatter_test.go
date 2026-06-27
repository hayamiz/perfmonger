package plotformatter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
