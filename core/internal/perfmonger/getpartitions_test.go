package perfmonger

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetPartitionsFromDirNoFdLeak verifies that getPartitionsFromDir does not
// leak a directory file descriptor on each call. The function opens the
// block-device directory; without a corresponding Close, every call permanently
// leaks one OS file descriptor.
func TestGetPartitionsFromDirNoFdLeak(t *testing.T) {
	// Build a fake sysfs layout:
	//   <blockDir>/<dev>/<part>/stat   (a partition)
	blockDir := t.TempDir()
	dev := "sdtest"
	devDir := filepath.Join(blockDir, dev)
	partDir := filepath.Join(devDir, dev+"1")
	if err := os.MkdirAll(partDir, 0o755); err != nil {
		t.Fatalf("failed to set up fake sysfs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(partDir, "stat"), []byte("0\n"), 0o644); err != nil {
		t.Fatalf("failed to write fake partition stat: %v", err)
	}

	countOpenFds := func() int {
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Skipf("/proc/self/fd not available: %v", err)
		}
		return len(entries)
	}

	// Warm up once so any lazy one-time allocations are not counted.
	_ = getPartitionsFromDir(blockDir, dev)

	before := countOpenFds()
	const iterations = 200
	for i := 0; i < iterations; i++ {
		parts := getPartitionsFromDir(blockDir, dev)
		if len(parts) != 1 || parts[0] != dev+"1" {
			t.Fatalf("unexpected partitions: %v", parts)
		}
	}
	after := countOpenFds()

	if after-before > 1 {
		t.Fatalf("getPartitionsFromDir leaked file descriptors: before=%d after=%d (delta=%d) over %d calls",
			before, after, after-before, iterations)
	}
}
