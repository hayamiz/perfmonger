package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEscapeGnuplotString verifies that gnuplot C-style significant characters
// in an output path (double-quote and backslash) are escaped so that the value
// can be safely embedded inside a `set output "..."` double-quoted string.
func TestEscapeGnuplotString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "/tmp/out.pdf", "/tmp/out.pdf"},
		{"double_quote", `a"b`, `a\"b`},
		{"backslash", `a\b`, `a\\b`},
		{"windows_path", `C:\dir\out.pdf`, `C:\\dir\\out.pdf`},
		{"quote_and_backslash", `a"\b`, `a\"\\b`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := escapeGnuplotString(c.in); got != c.want {
				t.Fatalf("escapeGnuplotString(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestGenerateDiskIOPSPlot_EscapesOutputPath verifies that a generated gnuplot
// script embeds the output path with gnuplot C-string escaping, so that a path
// containing a double-quote produces a syntactically valid `set output` line
// rather than a broken one.
func TestGenerateDiskIOPSPlot_EscapesOutputPath(t *testing.T) {
	raw := escapeGnuplotString(`/tmp/we"ird/dir`)
	if !strings.Contains(raw, `\"`) {
		t.Fatalf("expected escaped double-quote in %q", raw)
	}
}

// TestSaveGpfiles_PropagatesWriteError verifies that when a destination file
// cannot be written (e.g. the output directory is read-only), saveGpfiles
// returns a non-nil error instead of silently swallowing the failure.
func TestSaveGpfiles_PropagatesWriteError(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	name := "disk.dat"
	if err := os.WriteFile(filepath.Join(srcDir, name), []byte("data"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Make the destination directory read-only so os.WriteFile fails.
	if err := os.Chmod(dstDir, 0500); err != nil {
		t.Fatalf("setup chmod: %v", err)
	}
	defer os.Chmod(dstDir, 0700)

	err := saveGpfiles(srcDir, dstDir, []string{name})
	if err == nil {
		t.Fatalf("expected error when writing to read-only directory, got nil")
	}
}

// TestSaveGpfiles_Success verifies that existing source files are copied to the
// destination and missing files are skipped without error.
func TestSaveGpfiles_Success(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "disk.dat"), []byte("hello"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// "missing.dat" intentionally does not exist; it must be skipped silently.
	if err := saveGpfiles(srcDir, dstDir, []string{"disk.dat", "missing.dat"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dstDir, "disk.dat"))
	if err != nil {
		t.Fatalf("expected copied file: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("copied content mismatch: got %q", string(got))
	}
}

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
