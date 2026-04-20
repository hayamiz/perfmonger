package main

import (
	"testing"
	"time"
)

func TestNewStatCommandStruct(t *testing.T) {
	cmd := newStatCommandStruct()

	if cmd.RecorderOpt.Output != "./perfmonger.pgr" {
		t.Errorf("Output = %q, want %q", cmd.RecorderOpt.Output, "./perfmonger.pgr")
	}
	if !cmd.RecorderOpt.NoNet {
		t.Error("NoNet should be true by default")
	}
	if !cmd.RecorderOpt.NoIntr {
		t.Error("NoIntr should be true by default")
	}
	if !cmd.RecorderOpt.Gzip {
		t.Error("Gzip should be true by default")
	}
	if cmd.RecordIntr {
		t.Error("RecordIntr should be false by default")
	}
	if cmd.NoGzip {
		t.Error("NoGzip should be false by default")
	}
}

func TestStatCommand_ValidateAndSetCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     string
		wantCommand []string
	}{
		{
			name:        "basic command",
			args:        []string{"echo", "hello"},
			wantErr:     "",
			wantCommand: []string{"echo", "hello"},
		},
		{
			name:        "command with dash separator",
			args:        []string{"--", "echo", "hello"},
			wantErr:     "",
			wantCommand: []string{"echo", "hello"},
		},
		{
			name:    "empty args",
			args:    []string{},
			wantErr: "no command given",
		},
		{
			name:    "dash separator only",
			args:    []string{"--"},
			wantErr: "no command given",
		},
		{
			name:        "complex command",
			args:        []string{"ls", "-la", "/tmp"},
			wantErr:     "",
			wantCommand: []string{"ls", "-la", "/tmp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newStatCommandStruct()
			err := cmd.validateAndSetCommand(tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validateAndSetCommand() unexpected error: %v", err)
				}
				if len(tt.wantCommand) > 0 {
					if len(cmd.Command) != len(tt.wantCommand) {
						t.Errorf("Command = %v, want %v", cmd.Command, tt.wantCommand)
					} else {
						for i := range tt.wantCommand {
							if cmd.Command[i] != tt.wantCommand[i] {
								t.Errorf("Command[%d] = %q, want %q", i, cmd.Command[i], tt.wantCommand[i])
							}
						}
					}
				}
			} else {
				if err == nil {
					t.Errorf("validateAndSetCommand() expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("validateAndSetCommand() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestStatCommand_ValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*statCommand)
		wantErr string
	}{
		{
			name:    "valid defaults",
			setup:   func(cmd *statCommand) {},
			wantErr: "",
		},
		{
			name: "negative timeout",
			setup: func(cmd *statCommand) {
				cmd.RecorderOpt.Timeout = -1 * time.Second
			},
			wantErr: "timeout cannot be negative",
		},
		{
			name: "negative start delay",
			setup: func(cmd *statCommand) {
				cmd.RecorderOpt.StartDelay = -1 * time.Second
			},
			wantErr: "start-delay cannot be negative",
		},
		{
			name: "zero interval",
			setup: func(cmd *statCommand) {
				cmd.RecorderOpt.Interval = 0
			},
			wantErr: "interval must be positive",
		},
		{
			name: "negative interval",
			setup: func(cmd *statCommand) {
				cmd.RecorderOpt.Interval = -1 * time.Second
			},
			wantErr: "interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newStatCommandStruct()
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

func TestNewStatCommand(t *testing.T) {
	cmd := newStatCommand()

	if cmd.Use != "stat [options] -- <command>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "stat [options] -- <command>")
	}

	// Verify expected flags exist
	expectedFlags := []string{
		"disk", "logfile", "interval", "start-delay", "timeout",
		"record-intr", "no-cpu", "no-net", "no-mem", "no-gzip",
		"no-interval-backoff", "json", "verbose",
	}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q to be defined", name)
		}
	}

	// Verify short flags
	shortFlags := []string{"d", "l", "i", "s", "t"}
	for _, s := range shortFlags {
		if cmd.Flags().ShorthandLookup(s) == nil {
			t.Errorf("expected short flag %q to be defined", s)
		}
	}

	// Verify --kill and --status are NOT present
	removedFlags := []string{"kill", "status"}
	for _, name := range removedFlags {
		if cmd.Flags().Lookup(name) != nil {
			t.Errorf("flag %q should not be defined on stat command", name)
		}
	}
}
