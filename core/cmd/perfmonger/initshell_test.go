package main

import (
	"strings"
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

// TestErrorForShellUnsupported verifies the error message for a detected but
// unsupported shell. errorForShell is the surviving error path used by the live
// run() default branch (after the dead runInitShell was removed; see ticket
// 0036), so this guards that contract.
func TestErrorForShellUnsupported(t *testing.T) {
	err := errorForShell("fish")
	if err == nil {
		t.Fatalf("errorForShell(\"fish\") = nil, want an error")
	}
	want := "unsupported shell: fish. Only bash and zsh are supported"
	if got := err.Error(); got != want {
		t.Errorf("errorForShell(\"fish\") = %q, want %q", got, want)
	}
}

// TestCompletionScriptsListAllSubcommands verifies that the bash and zsh
// completion scripts offer every subcommand registered on the root command,
// so that adding a subcommand without updating the completions is caught.
func TestCompletionScriptsListAllSubcommands(t *testing.T) {
	bashSubcommands := map[string]bool{}
	for _, line := range strings.Split(bashCompletion, "\n") {
		line = strings.TrimSpace(line)
		if list, ok := strings.CutPrefix(line, `subcmds="`); ok {
			for _, name := range strings.Fields(strings.TrimSuffix(list, `"`)) {
				bashSubcommands[name] = true
			}
		}
	}
	if len(bashSubcommands) == 0 {
		t.Fatal("bash completion has no subcmds list")
	}

	for _, sub := range newRootCommand().Commands() {
		name := sub.Name()
		if name == "help" || name == "completion" {
			continue
		}
		if !bashSubcommands[name] {
			t.Errorf("bash completion does not list subcommand %q", name)
		}
		if !strings.Contains(zshCompletion, "'"+name+":") {
			t.Errorf("zsh completion does not list subcommand %q", name)
		}
	}
}
