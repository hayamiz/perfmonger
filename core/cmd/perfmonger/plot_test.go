package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCheckPdfCairoSupported_NoShellInjection verifies that a malicious
// GnuplotBin value containing shell metacharacters is NOT executed via a
// shell. If the value were interpolated into a shell command string, the
// injected payload would create a sentinel file as a side effect. With a
// direct argv exec, the whole value is treated as a literal program name
// (which does not exist), so the sentinel must never appear.
func TestCheckPdfCairoSupported_NoShellInjection(t *testing.T) {
	tmpDir := t.TempDir()
	sentinel := filepath.Join(tmpDir, "pwned")

	// A command-injection payload: if this string reaches a shell, the
	// `touch` runs and creates the sentinel file.
	payload := "gnuplot; touch " + sentinel + " #"

	// We expect an error (the bogus binary cannot be executed / pdfcairo
	// probe fails), but the key assertion is the absence of the side effect.
	_ = checkPdfCairoSupported(payload)

	if _, err := os.Stat(sentinel); err == nil {
		t.Fatalf("command injection occurred: sentinel file %q was created; "+
			"GnuplotBin must not be passed through a shell", sentinel)
	}
}
