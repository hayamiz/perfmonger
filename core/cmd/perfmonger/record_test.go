package main

import (
	"testing"
	"reflect"
)

func TestNewRecordOptions(t *testing.T) {
	opts := newRecordOptions()
	
	// Test Ruby-compatible defaults
	expected := &recordOptions{
		Disks:             []string{},
		Logfile:           "perfmonger.pgr",
		Interval:          1.0,
		Kill:              false,
		Status:            false,
		Background:        false,
		StartDelay:        0.0,
		Timeout:           0.0,
		RecordIntr:        false,
		NoCPU:             false,
		NoDisk:            false,
		NoNet:             true,  // Ruby default: don't record network
		NoMem:             false,
		NoGzip:            false, // Ruby default: use gzip
		NoIntervalBackoff: false,
		Debug:             false,
		Verbose:           false,
	}
	
	if !reflect.DeepEqual(opts, expected) {
		t.Errorf("newRecordOptions() defaults mismatch.\nGot:      %+v\nExpected: %+v", opts, expected)
	}
}

func TestRecordOptions_ParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		opts    *recordOptions
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default options",
			opts:    newRecordOptions(),
			args:    []string{},
			wantErr: false,
		},
		{
			name: "kill and status mutually exclusive",
			opts: &recordOptions{
				Kill:   true,
				Status: true,
			},
			args:    []string{},
			wantErr: true,
			errMsg:  "--kill and --status cannot be used together",
		},
		{
			name: "negative interval",
			opts: &recordOptions{
				Interval: -1.0,
			},
			args:    []string{},
			wantErr: true,
			errMsg:  "interval must be positive",
		},
		{
			name: "negative timeout",
			opts: &recordOptions{
				Timeout: -5.0,
			},
			args:    []string{},
			wantErr: true,
			errMsg:  "timeout cannot be negative",
		},
		{
			name: "negative start delay",
			opts: &recordOptions{
				StartDelay: -2.0,
			},
			args:    []string{},
			wantErr: true,
			errMsg:  "start-delay cannot be negative",
		},
		{
			name: "valid kill option",
			opts: &recordOptions{
				Kill:     true,
				Interval: 1.0,
			},
			args:    []string{},
			wantErr: false,
		},
		{
			name: "valid status option",
			opts: &recordOptions{
				Status:   true,
				Interval: 1.0,
			},
			args:    []string{},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.parseArgs(tt.args, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("parseArgs() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestRecordOptions_BuildRecorderArgs(t *testing.T) {
	tests := []struct {
		name string
		opts *recordOptions
		want []string
	}{
		{
			name: "default options",
			opts: newRecordOptions(),
			want: []string{
				"-interval=1000.0ms",
				"-no-intr", // RecordIntr is false by default
				"-no-net",  // NoNet is true by default (Ruby compatible)
				"-gzip",    // NoGzip is false, so add gzip
				"-output", "perfmonger.pgr.gz", // .gz added because NoGzip=false
			},
		},
		{
			name: "with disks and custom interval",
			opts: &recordOptions{
				Disks:      []string{"sda", "sdb"},
				Interval:   0.5,
				Logfile:    "custom.pgr",
				RecordIntr: true, // Enable interrupt recording
				NoNet:      false, // Enable network recording
				NoGzip:     true,  // Disable gzip
			},
			want: []string{
				"-interval=500.0ms",
				"-disks", "sda,sdb",
				"-output", "custom.pgr", // No .gz because NoGzip=true
			},
		},
		{
			name: "with timing options",
			opts: &recordOptions{
				Interval:   2.0,
				StartDelay: 1.5,
				Timeout:    10.0,
				Logfile:    "timed.pgr",
				RecordIntr: true,
				NoNet:      false,
				NoGzip:     true,
			},
			want: []string{
				"-interval=2000.0ms",
				"-start-delay", "1500ms",
				"-timeout", "10000ms", 
				"-output", "timed.pgr",
			},
		},
		{
			name: "with feature flags",
			opts: &recordOptions{
				Interval:          1.0,
				NoCPU:            true,
				NoDisk:           true,
				NoMem:            true,
				NoIntervalBackoff: true,
				Background:       true,
				Logfile:          "features.pgr",
				RecordIntr:       false,
				NoNet:           false,
				NoGzip:          true,
			},
			want: []string{
				"-interval=1000.0ms",
				"-no-interval-backoff",
				"-no-cpu",
				"-no-disk",
				"-no-intr",
				"-no-mem",
				"-background",
				"-output", "features.pgr",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.buildRecorderArgs()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildRecorderArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}