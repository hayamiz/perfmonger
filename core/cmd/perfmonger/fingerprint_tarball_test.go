package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var errTarballTest = errors.New("simulated write failure")

// failingWriter returns an error after a fixed number of bytes have been
// written. It is used to make the gzip/tar trailer flush (performed by
// Close()) fail, so we can verify that writeTarball propagates Close() errors
// instead of silently dropping them (ticket 0032).
type failingWriter struct {
	remaining int
	err       error
}

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, w.err
	}
	if len(p) > w.remaining {
		n := w.remaining
		w.remaining = 0
		return n, w.err
	}
	w.remaining -= len(p)
	return len(p), nil
}

// TestWriteTarballPropagatesCloseError verifies that a failure while flushing
// the gzip/tar trailers (which happens during Close()) is returned by
// writeTarball instead of being discarded by deferred Close() calls.
func TestWriteTarballPropagatesCloseError(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "perfmonger-fingerprint-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	baseName := "fp"
	baseDir := filepath.Join(tmpdir, baseName)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base dir: %v", err)
	}
	// Write a regular file so the tar contains content to flush.
	if err := ioutil.WriteFile(filepath.Join(baseDir, "info.log"), []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write sample file: %v", err)
	}

	f := &fingerprintCommand{}

	// Allow some bytes through (header + content), then fail. This makes the
	// final trailer flush during Close() error out.
	out := &failingWriter{remaining: 16, err: errTarballTest}

	if err := f.writeTarball(out, tmpdir, baseName); err == nil {
		t.Fatalf("expected writeTarball to return a Close()/write error, got nil")
	}
}
