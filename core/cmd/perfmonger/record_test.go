package main

import (
	"testing"
	"time"
)

func TestNewRecordCommandStruct(t *testing.T) {
	cmd := newRecordCommandStruct()

	// Ruby-compatible defaults
	if cmd.RecorderOpt.Output != "perfmonger.pgr.gz" {
		t.Errorf("Output = %q, want %q", cmd.RecorderOpt.Output, "perfmonger.pgr.gz")
	}
	if !cmd.RecorderOpt.NoNet {
		t.Error("NoNet should be true by default")
	}
	if !cmd.RecorderOpt.Gzip {
		t.Error("Gzip should be true by default")
	}
	if cmd.Kill {
		t.Error("Kill should be false by default")
	}
	if cmd.Status {
		t.Error("Status should be false by default")
	}
	if cmd.NoGzip {
		t.Error("NoGzip should be false by default")
	}
	if cmd.RecordIntr {
		t.Error("RecordIntr should be false by default")
	}
	if cmd.Verbose {
		t.Error("Verbose should be false by default")
	}
}

func TestRecordCommand_ValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*recordCommand)
		wantErr string
	}{
		{
			name:    "valid defaults",
			setup:   func(cmd *recordCommand) {},
			wantErr: "",
		},
		{
			name: "kill and status mutually exclusive",
			setup: func(cmd *recordCommand) {
				cmd.Kill = true
				cmd.Status = true
			},
			wantErr: "--kill and --status cannot be used together",
		},
		{
			name: "negative interval",
			setup: func(cmd *recordCommand) {
				cmd.RecorderOpt.Interval = -1 * time.Second
			},
			wantErr: "interval must be positive",
		},
		{
			name: "negative timeout",
			setup: func(cmd *recordCommand) {
				cmd.RecorderOpt.Timeout = -5 * time.Second
			},
			wantErr: "timeout cannot be negative",
		},
		{
			name: "negative start delay",
			setup: func(cmd *recordCommand) {
				cmd.RecorderOpt.StartDelay = -2 * time.Second
			},
			wantErr: "start-delay cannot be negative",
		},
		{
			name: "kill alone skips validation",
			setup: func(cmd *recordCommand) {
				cmd.Kill = true
			},
			wantErr: "",
		},
		{
			name: "status alone skips validation",
			setup: func(cmd *recordCommand) {
				cmd.Status = true
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRecordCommandStruct()
			tt.setup(cmd)
			err := cmd.validateOptions()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validateOptions() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("validateOptions() expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("validateOptions() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestNewRecordCommand(t *testing.T) {
	cmd := newRecordCommand()

	if cmd.Use != "record [options]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "record [options]")
	}

	// Verify expected flags exist
	expectedFlags := []string{
		"disk", "logfile", "interval", "start-delay", "timeout",
		"kill", "status", "background", "record-intr",
		"no-cpu", "no-net", "no-mem", "no-gzip", "no-interval-backoff",
		"verbose",
	}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q to be defined", name)
		}
	}

	// Verify short flags
	shortFlags := []string{"d", "l", "i", "s", "t", "v"}
	for _, s := range shortFlags {
		if cmd.Flags().ShorthandLookup(s) == nil {
			t.Errorf("expected short flag %q to be defined", s)
		}
	}
}
