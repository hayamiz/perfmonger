package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestNewPlayOptions(t *testing.T) {
	opts := newPlayOptions()
	
	expected := &playOptions{
		Color:    false,
		Pretty:   false,
		DiskOnly: "",
		LogFile:  "",
	}
	
	if !reflect.DeepEqual(opts, expected) {
		t.Errorf("newPlayOptions() defaults mismatch.\nGot:      %+v\nExpected: %+v", opts, expected)
	}
}

func TestPlayOptions_ParseArgs(t *testing.T) {
	// Create a temporary test file
	tmpfile, err := ioutil.TempFile("", "test.pgr")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()
	
	tests := []struct {
		name    string
		opts    *playOptions
		args    []string
		wantErr bool
		errMsg  string
		wantLogFile string
	}{
		{
			name:    "no log file provided",
			opts:    newPlayOptions(),
			args:    []string{},
			wantErr: true,
			errMsg:  "PerfMonger log file is required",
		},
		{
			name:    "non-existent log file",
			opts:    newPlayOptions(),
			args:    []string{"nonexistent.pgr"},
			wantErr: true,
			errMsg:  "no such file: nonexistent.pgr",
		},
		{
			name:        "valid log file",
			opts:        newPlayOptions(),
			args:        []string{tmpfile.Name()},
			wantErr:     false,
			wantLogFile: tmpfile.Name(),
		},
		{
			name:        "log file with extra args ignored",
			opts:        newPlayOptions(),
			args:        []string{tmpfile.Name(), "extra", "args"},
			wantErr:     false,
			wantLogFile: tmpfile.Name(),
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
			if !tt.wantErr && tt.opts.LogFile != tt.wantLogFile {
				t.Errorf("parseArgs() LogFile = %v, want %v", tt.opts.LogFile, tt.wantLogFile)
			}
		})
	}
}

func TestPlayOptions_BuildPlayerArgs(t *testing.T) {
	tests := []struct {
		name string
		opts *playOptions
		want []string
	}{
		{
			name: "default options",
			opts: &playOptions{
				Color:    false,
				Pretty:   false,
				DiskOnly: "",
				LogFile:  "test.pgr",
			},
			want: []string{"test.pgr"},
		},
		{
			name: "with color option",
			opts: &playOptions{
				Color:    true,
				Pretty:   false,
				DiskOnly: "",
				LogFile:  "test.pgr",
			},
			want: []string{"-color", "test.pgr"},
		},
		{
			name: "with pretty option",
			opts: &playOptions{
				Color:    false,
				Pretty:   true,
				DiskOnly: "",
				LogFile:  "test.pgr",
			},
			want: []string{"-pretty", "test.pgr"},
		},
		{
			name: "with disk-only option",
			opts: &playOptions{
				Color:    false,
				Pretty:   false,
				DiskOnly: "sd[a-d]",
				LogFile:  "test.pgr",
			},
			want: []string{"-disk-only", "sd[a-d]", "test.pgr"},
		},
		{
			name: "with all options",
			opts: &playOptions{
				Color:    true,
				Pretty:   true,
				DiskOnly: "nvme.*",
				LogFile:  "test.pgr",
			},
			want: []string{"-color", "-pretty", "-disk-only", "nvme.*", "test.pgr"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.buildPlayerArgs()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildPlayerArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}