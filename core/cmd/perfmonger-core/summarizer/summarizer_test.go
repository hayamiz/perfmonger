package summarizer

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	projson "github.com/hayamiz/go-projson"
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

// TestWriteJSONPropagatesPrinterError verifies that when the JSON printer is in
// an unfinished state (printer.String() fails), writeJSON returns the actual
// error and does NOT write the debug literal "skip by err" to the output.
func TestWriteJSONPropagatesPrinterError(t *testing.T) {
	printer := projson.NewPrinter()
	// Begin an object but never finish it, so printer.String() returns an error.
	printer.BeginObject()

	var buf bytes.Buffer
	err := writeJSON(printer, &buf)
	if err == nil {
		t.Fatalf("expected writeJSON to return an error for an unfinished printer, got nil")
	}

	if strings.Contains(buf.String(), "skip by err") {
		t.Fatalf("writeJSON wrote the debug literal %q to output: %s", "skip by err", buf.String())
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output on error, got: %q", buf.String())
	}
}
