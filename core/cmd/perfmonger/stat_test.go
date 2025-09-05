package main

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestStatOptions_parseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setupOpts func(*statOptions)
		wantErr   string
		wantOpts  *statOptions
	}{
		{
			name: "basic command",
			args: []string{"echo", "hello"},
			setupOpts: func(opts *statOptions) {
				// Use defaults
			},
			wantOpts: &statOptions{
				Disks:             []string{},
				Logfile:           "./perfmonger.pgr",
				Interval:          1.0,
				StartDelay:        0.0,
				Timeout:           0.0,
				NoCPU:             false,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              false,
				Command:           []string{"echo", "hello"},
			},
		},
		{
			name: "command with dash separator",
			args: []string{"--", "echo", "hello"},
			setupOpts: func(opts *statOptions) {
				// Use defaults
			},
			wantOpts: &statOptions{
				Disks:             []string{},
				Logfile:           "./perfmonger.pgr",
				Interval:          1.0,
				StartDelay:        0.0,
				Timeout:           0.0,
				NoCPU:             false,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              false,
				Command:           []string{"echo", "hello"},
			},
		},
		{
			name: "command with complex args",
			args: []string{"ls", "-la", "/tmp"},
			setupOpts: func(opts *statOptions) {
				// Use defaults
			},
			wantOpts: &statOptions{
				Disks:             []string{},
				Logfile:           "./perfmonger.pgr",
				Interval:          1.0,
				StartDelay:        0.0,
				Timeout:           0.0,
				NoCPU:             false,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              false,
				Command:           []string{"ls", "-la", "/tmp"},
			},
		},
		{
			name: "empty command should fail",
			args: []string{},
			setupOpts: func(opts *statOptions) {
				// Use defaults
			},
			wantErr: "no command given",
		},
		{
			name: "command with empty dash separator should fail",
			args: []string{"--"},
			setupOpts: func(opts *statOptions) {
				// Use defaults
			},
			wantErr: "no command given",
		},
		{
			name: "negative timeout should fail",
			args: []string{"echo", "hello"},
			setupOpts: func(opts *statOptions) {
				opts.Timeout = -1.0
			},
			wantErr: "timeout cannot be negative",
		},
		{
			name: "negative start delay should fail",
			args: []string{"echo", "hello"},
			setupOpts: func(opts *statOptions) {
				opts.StartDelay = -1.0
			},
			wantErr: "start-delay cannot be negative",
		},
		{
			name: "zero interval should fail",
			args: []string{"echo", "hello"},
			setupOpts: func(opts *statOptions) {
				opts.Interval = 0.0
			},
			wantErr: "interval must be positive",
		},
		{
			name: "negative interval should fail",
			args: []string{"echo", "hello"},
			setupOpts: func(opts *statOptions) {
				opts.Interval = -1.0
			},
			wantErr: "interval must be positive",
		},
		{
			name: "valid options with command",
			args: []string{"sleep", "1"},
			setupOpts: func(opts *statOptions) {
				opts.Interval = 0.5
				opts.StartDelay = 0.1
				opts.Timeout = 5.0
				opts.Disks = []string{"sda", "sdb"}
				opts.Logfile = "test.pgr"
				opts.JSON = true
				opts.NoCPU = true
			},
			wantOpts: &statOptions{
				Disks:             []string{"sda", "sdb"},
				Logfile:           "test.pgr",
				Interval:          0.5,
				StartDelay:        0.1,
				Timeout:           5.0,
				NoCPU:             true,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              true,
				Command:           []string{"sleep", "1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := newStatOptions()
			if tt.setupOpts != nil {
				tt.setupOpts(opts)
			}

			// Create a dummy command for testing
			cmd := &cobra.Command{}

			err := opts.parseArgs(tt.args, cmd)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("parseArgs() expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("parseArgs() expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("parseArgs() unexpected error: %v", err)
				return
			}

			if tt.wantOpts != nil {
				if !reflect.DeepEqual(opts, tt.wantOpts) {
					t.Errorf("parseArgs() options mismatch:\ngot:  %+v\nwant: %+v", opts, tt.wantOpts)
				}
			}
		})
	}
}

func TestStatOptions_buildRecorderArgs(t *testing.T) {
	tests := []struct {
		name     string
		opts     *statOptions
		wantArgs []string
	}{
		{
			name: "default options",
			opts: newStatOptions(),
			wantArgs: []string{
				"-interval=1000.0ms",
				"-no-net",
				"-no-intr",
				"-gzip",
				"-output", "./perfmonger.pgr",
			},
		},
		{
			name: "custom interval",
			opts: &statOptions{
				Disks:             []string{},
				Logfile:           "./perfmonger.pgr",
				Interval:          0.5,
				StartDelay:        0.0,
				Timeout:           0.0,
				NoCPU:             false,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              false,
				Command:           []string{"echo"},
			},
			wantArgs: []string{
				"-interval=500.0ms",
				"-no-net",
				"-no-intr",
				"-gzip",
				"-output", "./perfmonger.pgr",
			},
		},
		{
			name: "with disks",
			opts: &statOptions{
				Disks:             []string{"sda", "sdb"},
				Logfile:           "./perfmonger.pgr",
				Interval:          1.0,
				StartDelay:        0.0,
				Timeout:           0.0,
				NoCPU:             false,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              false,
				Command:           []string{"echo"},
			},
			wantArgs: []string{
				"-interval=1000.0ms",
				"-no-net",
				"-no-intr",
				"-disks", "[sda sdb]",
				"-gzip",
				"-output", "./perfmonger.pgr",
			},
		},
		{
			name: "all features disabled",
			opts: &statOptions{
				Disks:             []string{},
				Logfile:           "test.pgr",
				Interval:          2.0,
				StartDelay:        1.0,
				Timeout:           10.0,
				NoCPU:             true,
				NoDisk:            true,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             true,
				NoGzip:            true,
				NoIntervalBackoff: true,
				Debug:             false,
				JSON:              false,
				Command:           []string{"echo"},
			},
			wantArgs: []string{
				"-interval=2000.0ms",
				"-no-interval-backoff",
				"-start-delay", "1000ms",
				"-timeout", "10000ms",
				"-no-cpu",
				"-no-disk",
				"-no-net",
				"-no-intr",
				"-no-mem",
				"-output", "test.pgr",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs := tt.opts.buildRecorderArgs()
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("buildRecorderArgs() mismatch:\ngot:  %v\nwant: %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestStatOptions_buildSummaryArgs(t *testing.T) {
	tests := []struct {
		name     string
		opts     *statOptions
		wantArgs []string
	}{
		{
			name: "default options",
			opts: &statOptions{
				Disks:             []string{},
				Logfile:           "./perfmonger.pgr",
				Interval:          1.0,
				StartDelay:        0.0,
				Timeout:           0.0,
				NoCPU:             false,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              false,
				Command:           []string{"echo", "hello"},
			},
			wantArgs: []string{
				"-title", "echo hello",
				"./perfmonger.pgr",
			},
		},
		{
			name: "with JSON output",
			opts: &statOptions{
				Disks:             []string{},
				Logfile:           "test.pgr",
				Interval:          1.0,
				StartDelay:        0.0,
				Timeout:           0.0,
				NoCPU:             false,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              true,
				Command:           []string{"ls", "-la"},
			},
			wantArgs: []string{
				"-json",
				"-title", "ls -la",
				"test.pgr",
			},
		},
		{
			name: "complex command",
			opts: &statOptions{
				Disks:             []string{},
				Logfile:           "./perfmonger.pgr",
				Interval:          1.0,
				StartDelay:        0.0,
				Timeout:           0.0,
				NoCPU:             false,
				NoDisk:            false,
				NoNet:             true,
				NoIntr:            true,
				NoMem:             false,
				NoGzip:            false,
				NoIntervalBackoff: false,
				Debug:             false,
				JSON:              false,
				Command:           []string{"find", "/tmp", "-name", "*.txt", "-exec", "cat", "{}", ";"},
			},
			wantArgs: []string{
				"-title", "find /tmp -name *.txt -exec cat {} ;",
				"./perfmonger.pgr",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs := tt.opts.buildSummaryArgs()
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("buildSummaryArgs() mismatch:\ngot:  %v\nwant: %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestNewStatCommand(t *testing.T) {
	cmd := newStatCommand()

	// Test basic command properties
	if cmd.Use != "stat [options] -- <command>" {
		t.Errorf("expected Use to be 'stat [options] -- <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Run a command and show performance summary" {
		t.Errorf("expected Short to be 'Run a command and show performance summary', got %q", cmd.Short)
	}

	// Test that required flags are present (long names)
	requiredLongFlags := []string{
		"disk", "logfile", "interval", "start-delay", "timeout", "json",
	}

	for _, flagName := range requiredLongFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q to be defined", flagName)
		}
	}

	// Test that short flags are present
	requiredShortFlags := []string{"d", "l", "i", "s", "t"}
	for _, shortFlag := range requiredShortFlags {
		flag := cmd.Flags().ShorthandLookup(shortFlag)
		if flag == nil {
			t.Errorf("expected short flag %q to be defined", shortFlag)
		}
	}

	// Test that boolean flags are present
	boolFlags := []string{
		"no-cpu", "no-net", "no-mem", "no-gzip", "no-interval-backoff", "json",
	}

	for _, flagName := range boolFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected boolean flag %q to be defined", flagName)
		}
		if flag.Value.Type() != "bool" {
			t.Errorf("expected flag %q to be boolean, got %s", flagName, flag.Value.Type())
		}
	}
}