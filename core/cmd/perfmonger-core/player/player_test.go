package player

import (
	"bufio"
	"errors"
	"testing"
)

// failingWriter is an io.Writer that always returns an error on Write.
type failingWriter struct{}

var errWriteFailed = errors.New("write failed")

func (w *failingWriter) Write(p []byte) (int, error) {
	return 0, errWriteFailed
}

// TestWriteRecordSurfacesWriteError verifies that a failure from the underlying
// WriteString is surfaced (returned as a non-nil error) rather than being
// silently discarded due to inner-scope variable shadowing.
func TestWriteRecordSurfacesWriteError(t *testing.T) {
	// Use a tiny buffer so WriteString immediately flushes to the underlying
	// failing writer and the error is observed at WriteString time.
	out := bufio.NewWriterSize(&failingWriter{}, 1)

	err := writeRecord(out, "some json record")
	if err == nil {
		t.Fatalf("expected a non-nil error when the underlying write fails, got nil")
	}
}
