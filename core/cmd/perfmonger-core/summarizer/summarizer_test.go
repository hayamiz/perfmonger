package summarizer

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ss "github.com/hayamiz/perfmonger/core/internal/perfmonger"
)

// writeOneRecordLog writes a plain (non-gzipped) gob log containing a common
// header, a platform header, and exactly one StatRecord. It returns the path
// to the written log file.
func writeOneRecordLog(t *testing.T, rec ss.StatRecord) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "one_record.pgr")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}
	defer f.Close()

	enc := gob.NewEncoder(f)

	cheader := ss.CommonHeader{
		Platform:  ss.Linux,
		Hostname:  "testhost",
		StartTime: rec.Time,
	}
	if err := enc.Encode(&cheader); err != nil {
		t.Fatalf("failed to encode common header: %v", err)
	}

	var pheader ss.PlatformHeader
	if err := enc.Encode(&pheader); err != nil {
		t.Fatalf("failed to encode platform header: %v", err)
	}

	if err := enc.Encode(&rec); err != nil {
		t.Fatalf("failed to encode record: %v", err)
	}

	return path
}

// TestRunDirectOneRecordInterval verifies that a log with exactly one data
// record uses that single record as both the first and last record, yielding a
// zero (non-negative) interval rather than a huge negative duration produced by
// a zero-initialized last record.
func TestRunDirectOneRecordInterval(t *testing.T) {
	rec := ss.StatRecord{
		Time: time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC),
	}

	path := writeOneRecordLog(t, rec)

	option := NewSummaryOption()
	option.Logfile = path
	option.JSON = true

	var buf bytes.Buffer
	if err := RunDirect(option, &buf); err != nil {
		t.Fatalf("RunDirect returned error: %v", err)
	}

	out := buf.String()
	// With the bug, the last record is a zero-initialized StatRecord whose
	// Time is the zero value, producing a huge negative exectime.
	if strings.Contains(out, `"exectime":-`) {
		t.Fatalf("one-record log produced negative exectime (zero last record bug): %s", out)
	}
	if !strings.Contains(out, `"exectime":0.000`) {
		t.Fatalf("expected zero exectime for one-record log, got: %s", out)
	}
}
