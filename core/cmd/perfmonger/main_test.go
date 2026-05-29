package main

import (
	"testing"
)

// TestRootCommandVersionWired verifies that the root command's Version is
// wired from the package-level VERSION variable. VERSION is a var (not a
// const) so that release builds can override it via
// -ldflags "-X main.VERSION=<tag>".
func TestRootCommandVersionWired(t *testing.T) {
	if VERSION == "" {
		t.Fatal("VERSION must not be empty")
	}

	cmd := newRootCommand()
	if cmd.Version == "" {
		t.Error("root command Version should not be empty")
	}
	if cmd.Version != VERSION {
		t.Errorf("root command Version = %q, want %q", cmd.Version, VERSION)
	}
}
