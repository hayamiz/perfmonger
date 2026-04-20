package main

import (
	"os"
	"testing"
)

func TestNewPlayCommandStruct(t *testing.T) {
	cmd := newPlayCommandStruct()

	if cmd.PlayerOpt.Color {
		t.Error("Color should be false by default")
	}
	if cmd.PlayerOpt.Pretty {
		t.Error("Pretty should be false by default")
	}
	if cmd.PlayerOpt.Logfile != "-" {
		t.Errorf("Logfile = %q, want %q", cmd.PlayerOpt.Logfile, "-")
	}
}

func TestPlayCommand_ValidateAndSetLogfile(t *testing.T) {
	// Create a temporary file for testing
	tmpfile, err := os.CreateTemp("", "test*.pgr")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	tests := []struct {
		name        string
		args        []string
		wantErr     string
		wantLogfile string
	}{
		{
			name:        "no args reads from stdin",
			args:        []string{},
			wantErr:     "",
			wantLogfile: "-",
		},
		{
			name:    "non-existent file",
			args:    []string{"nonexistent.pgr"},
			wantErr: "no such file: nonexistent.pgr",
		},
		{
			name:        "valid file",
			args:        []string{tmpfile.Name()},
			wantErr:     "",
			wantLogfile: tmpfile.Name(),
		},
		{
			name:        "extra args ignored",
			args:        []string{tmpfile.Name(), "extra", "args"},
			wantErr:     "",
			wantLogfile: tmpfile.Name(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPlayCommandStruct()
			err := cmd.validateAndSetLogfile(tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validateAndSetLogfile() unexpected error: %v", err)
				} else if cmd.PlayerOpt.Logfile != tt.wantLogfile {
					t.Errorf("Logfile = %q, want %q", cmd.PlayerOpt.Logfile, tt.wantLogfile)
				}
			} else {
				if err == nil {
					t.Errorf("validateAndSetLogfile() expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("validateAndSetLogfile() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestNewPlayCommand(t *testing.T) {
	cmd := newPlayCommand()

	if cmd.Use != "play [options] LOG_FILE" {
		t.Errorf("Use = %q, want %q", cmd.Use, "play [options] LOG_FILE")
	}

	expectedFlags := []string{"color", "pretty", "disk-only"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q to be defined", name)
		}
	}

	shortFlags := []string{"c", "p"}
	for _, s := range shortFlags {
		if cmd.Flags().ShorthandLookup(s) == nil {
			t.Errorf("expected short flag %q to be defined", s)
		}
	}
}
