package main

import (
	"testing"
)

// TestNormalizeShellName verifies that raw `ps` / SHELL output is mapped to a
// clean shell name, and that the empty-output case does NOT collapse to "."
// (which filepath.Base("") would otherwise produce). See ticket 0034.
func TestNormalizeShellName(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"bash", "bash", "bash"},
		{"zsh full path", "/bin/zsh", "zsh"},
		{"bash with whitespace", "  bash\n", "bash"},
		{"empty output stays empty", "", ""},
		{"whitespace only stays empty", "   \n", ""},
		{"versioned bash binary", "/usr/bin/bash-5.1", "bash"},
		{"versioned zsh binary", "/usr/local/bin/zsh-5.9", "zsh"},
		{"versioned bash name only", "bash-5.1", "bash"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeShellName(tc.raw)
			if got != tc.want {
				t.Errorf("normalizeShellName(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestRunUnsupportedShellMessage verifies that when no shell can be detected
// (empty result), run() reports a clear "could not detect parent shell" error
// rather than the misleading "unsupported shell: ." message. See ticket 0034.
func TestRunDetectionFailureMessage(t *testing.T) {
	err := errorForShell("")
	if err == nil {
		t.Fatalf("errorForShell(\"\") = nil, want an error")
	}
	if got := err.Error(); got != "could not detect parent shell" {
		t.Errorf("errorForShell(\"\") = %q, want %q", got, "could not detect parent shell")
	}
}
