package perfmonger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// Per-core CPU frequency collection.
//
// Frequencies are read from /sys/devices/system/cpu/cpuN/cpufreq/scaling_cur_freq
// (kHz). The files are opened once and kept open, and each sample re-reads
// them with a single pread(fd, buf, 0) per core, which re-invokes the sysfs
// show() callback without the cost of open/close.
//
// Overhead considerations (x86):
//   - Since Linux 5.19, scaling_cur_freq is computed from APERF/MPERF deltas
//     cached by the scheduler tick (arch_freq_get_on_cpu); a read sends no IPI
//     and never sleeps. For a core that has been idle for more than 20ms the
//     kernel falls back to the last requested cpufreq frequency.
//   - Up to Linux 5.18, every read of a sample older than 10ms performs a
//     synchronous IPI to the target core (idle cores included), and a sample
//     older than 1s additionally triggers msleep(10). That would perturb the
//     very C-state/DVFS behavior being measured, so frequency recording is
//     refused on x86 kernels older than 5.19.
//
// On other architectures the generic arch_freq_get_on_cpu() returns 0 and the
// value comes from the cpufreq policy (no IPI).

const sysCpuDir = "/sys/devices/system/cpu"

// ErrCpuFreqUnavailable is returned (possibly wrapped) by OpenCpuFreqReader when
// no CPU core exposes a readable cpufreq scaling_cur_freq file, e.g. inside
// most VMs and containers.
var ErrCpuFreqUnavailable = errors.New("cpufreq scaling_cur_freq is not available")

// CpuFreqKernelError is returned by OpenCpuFreqReader when cpufreq is present
// but reading it is deliberately avoided because the running kernel would send
// an IPI to each core on every read.
type CpuFreqKernelError struct {
	Release string
}

func (e *CpuFreqKernelError) Error() string {
	return fmt.Sprintf("x86 kernel %s (< 5.19) sends an IPI to each core on every scaling_cur_freq read", e.Release)
}

// CpuFreqReader samples the operating frequency of every CPU core.
type CpuFreqReader struct {
	fds []int // per-core fd of scaling_cur_freq, -1 if unavailable
	buf [32]byte
}

// OpenCpuFreqReader probes cpufreq sysfs for numCore cores and returns a reader
// when at least one core's frequency can be read at low cost. Otherwise it
// returns a nil reader and an error: *CpuFreqKernelError when the kernel is
// known to make the reads expensive, or an error wrapping
// ErrCpuFreqUnavailable when no core exposes a readable frequency.
func OpenCpuFreqReader(numCore int) (*CpuFreqReader, error) {
	release := ""
	if b, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		release = strings.TrimSpace(string(b))
	}
	return openCpuFreqReader(sysCpuDir, numCore, runtime.GOARCH, release)
}

// openCpuFreqReader is the testable core of OpenCpuFreqReader: the sysfs CPU
// directory, architecture and kernel release are injectable.
func openCpuFreqReader(cpuDir string, numCore int, goarch string, release string) (*CpuFreqReader, error) {
	if numCore < 1 {
		return nil, fmt.Errorf("%w: invalid number of cores %d", ErrCpuFreqUnavailable, numCore)
	}

	r := &CpuFreqReader{fds: make([]int, numCore)}
	opened := 0
	for i := range r.fds {
		path := filepath.Join(cpuDir, "cpu"+strconv.Itoa(i), "cpufreq", "scaling_cur_freq")
		fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
		if err != nil {
			r.fds[i] = -1
			continue
		}
		r.fds[i] = fd
		opened++
	}
	if opened == 0 {
		return nil, fmt.Errorf("%w: no %s/cpuN/cpufreq/scaling_cur_freq could be opened", ErrCpuFreqUnavailable, cpuDir)
	}

	// Check the kernel before the first read: on affected kernels even the
	// probe read would IPI every core.
	if !cpuFreqReadIsCheap(goarch, release) {
		r.Close()
		return nil, &CpuFreqKernelError{Release: release}
	}

	// Probe once: at least one core must yield a valid frequency.
	probe := NewCpuFreqStat(numCore)
	r.read(probe)
	for _, khz := range probe.CoreFreqs {
		if khz > 0 {
			return r, nil
		}
	}
	r.Close()
	return nil, fmt.Errorf("%w: no core returned a valid frequency", ErrCpuFreqUnavailable)
}

// cpuFreqReadIsCheap reports whether reading scaling_cur_freq avoids IPIs on
// the given architecture and kernel release. Only x86 kernels older than 5.19
// are known to be expensive. An unparsable release is assumed to be recent.
func cpuFreqReadIsCheap(goarch string, release string) bool {
	if goarch != "amd64" && goarch != "386" {
		return true
	}
	major, minor, ok := parseKernelVersion(release)
	if !ok {
		return true
	}
	return major > 5 || (major == 5 && minor >= 19)
}

// parseKernelVersion extracts the leading "major.minor" of a kernel release
// string such as "6.8.0-142-generic" or "4.18.0-553.el8_10.x86_64".
func parseKernelVersion(release string) (major int, minor int, ok bool) {
	parts := strings.SplitN(release, ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minorStr := parts[1]
	end := 0
	for end < len(minorStr) && minorStr[end] >= '0' && minorStr[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, 0, false
	}
	minor, err = strconv.Atoi(minorStr[:end])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

// NumCore returns the number of cores the reader covers.
func (r *CpuFreqReader) NumCore() int {
	return len(r.fds)
}

// Read samples the current frequency of every core into record.CpuFreq,
// allocating it on first use. Cores whose frequency cannot be read in this
// sample (no cpufreq policy, offline CPU, read error) are recorded as 0.
func (r *CpuFreqReader) Read(record *StatRecord) {
	if record.CpuFreq == nil || len(record.CpuFreq.CoreFreqs) != len(r.fds) {
		record.CpuFreq = NewCpuFreqStat(len(r.fds))
	}
	r.read(record.CpuFreq)
}

func (r *CpuFreqReader) read(stat *CpuFreqStat) {
	for i, fd := range r.fds {
		stat.CoreFreqs[i] = 0
		if fd < 0 {
			continue
		}
		n, err := syscall.Pread(fd, r.buf[:], 0)
		if err != nil || n <= 0 {
			continue
		}
		if khz, ok := parseFreqKHz(r.buf[:n]); ok {
			stat.CoreFreqs[i] = khz
		}
	}
}

// Close releases all file descriptors held by the reader. It is safe to call
// on a nil reader and more than once.
func (r *CpuFreqReader) Close() {
	if r == nil {
		return
	}
	for i, fd := range r.fds {
		if fd >= 0 {
			syscall.Close(fd)
			r.fds[i] = -1
		}
	}
}

// parseFreqKHz parses the decimal kHz value of a scaling_cur_freq read (e.g.
// "1200000\n") without allocating. It rejects empty and non-numeric content.
func parseFreqKHz(b []byte) (int64, bool) {
	var v int64
	digits := 0
	for _, c := range b {
		switch {
		case c >= '0' && c <= '9':
			if digits >= 18 {
				return 0, false
			}
			v = v*10 + int64(c-'0')
			digits++
		case c == '\n' || c == ' ' || c == '\t' || c == '\r' || c == 0:
			if digits > 0 {
				return v, true
			}
		default:
			return 0, false
		}
	}
	return v, digits > 0
}
