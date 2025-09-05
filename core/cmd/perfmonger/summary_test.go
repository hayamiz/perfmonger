package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewSummaryOptions(t *testing.T) {
	opts := newSummaryOptions()
	
	expected := &summaryOptions{
		JSON:     false,
		Pager:    "",
		DiskOnly: "",
		LogFile:  "",
	}
	
	if !reflect.DeepEqual(opts, expected) {
		t.Errorf("newSummaryOptions() defaults mismatch.\nGot:      %+v\nExpected: %+v", opts, expected)
	}
}

func TestSummaryOptions_ParseArgs(t *testing.T) {
	// Create a temporary test file
	tmpfile, err := ioutil.TempFile("", "test.pgr")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()
	
	tests := []struct {
		name        string
		opts        *summaryOptions
		args        []string
		pagerEnv    string
		setPagerFlag bool
		wantErr     bool
		errMsg      string
		wantLogFile string
		wantPager   string
	}{
		{
			name:    "no log file provided",
			opts:    newSummaryOptions(),
			args:    []string{},
			wantErr: true,
			errMsg:  "PerfMonger log file is required",
		},
		{
			name:    "non-existent log file",
			opts:    newSummaryOptions(),
			args:    []string{"nonexistent.pgr"},
			wantErr: true,
			errMsg:  "no such file: nonexistent.pgr",
		},
		{
			name:        "valid log file",
			opts:        newSummaryOptions(),
			args:        []string{tmpfile.Name()},
			wantErr:     false,
			wantLogFile: tmpfile.Name(),
		},
		{
			name:         "pager flag set without value, PAGER env available",
			opts:         &summaryOptions{Pager: ""},
			args:         []string{tmpfile.Name()},
			pagerEnv:     "less",
			setPagerFlag: true,
			wantErr:      false,
			wantLogFile:  tmpfile.Name(),
			wantPager:    "less",
		},
		{
			name:         "pager flag set without value, no PAGER env",
			opts:         &summaryOptions{Pager: ""},
			args:         []string{tmpfile.Name()},
			pagerEnv:     "",
			setPagerFlag: true,
			wantErr:      true,
			errMsg:       "no pager is available. Please set PAGER or give pager name to --pager option",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			oldPager := os.Getenv("PAGER")
			defer os.Setenv("PAGER", oldPager)
			
			if tt.pagerEnv != "" {
				os.Setenv("PAGER", tt.pagerEnv)
			} else {
				os.Unsetenv("PAGER")
			}
			
			// Create a mock cobra command to simulate flag behavior
			cmd := &cobra.Command{}
			cmd.Flags().StringVarP(&tt.opts.Pager, "pager", "p", tt.opts.Pager, "")
			
			if tt.setPagerFlag {
				cmd.Flags().Set("pager", "")
			}
			
			err := tt.opts.parseArgs(tt.args, cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("parseArgs() error message = %v, want %v", err.Error(), tt.errMsg)
			}
			if !tt.wantErr {
				if tt.opts.LogFile != tt.wantLogFile {
					t.Errorf("parseArgs() LogFile = %v, want %v", tt.opts.LogFile, tt.wantLogFile)
				}
				if tt.wantPager != "" && tt.opts.Pager != tt.wantPager {
					t.Errorf("parseArgs() Pager = %v, want %v", tt.opts.Pager, tt.wantPager)
				}
			}
		})
	}
}

func TestSummaryOptions_BuildSummarizerArgs(t *testing.T) {
	tests := []struct {
		name string
		opts *summaryOptions
		want []string
	}{
		{
			name: "default options",
			opts: &summaryOptions{
				JSON:     false,
				DiskOnly: "",
				LogFile:  "test.pgr",
			},
			want: []string{"-title", "test.pgr", "test.pgr"},
		},
		{
			name: "with JSON option",
			opts: &summaryOptions{
				JSON:     true,
				DiskOnly: "",
				LogFile:  "test.pgr",
			},
			want: []string{"-json", "-title", "test.pgr", "test.pgr"},
		},
		{
			name: "with disk-only option",
			opts: &summaryOptions{
				JSON:     false,
				DiskOnly: "sd[a-d]",
				LogFile:  "test.pgr",
			},
			want: []string{"-disk-only", "sd[a-d]", "-title", "test.pgr", "test.pgr"},
		},
		{
			name: "with all options",
			opts: &summaryOptions{
				JSON:     true,
				DiskOnly: "nvme.*",
				LogFile:  "test.pgr",
			},
			want: []string{"-json", "-disk-only", "nvme.*", "-title", "test.pgr", "test.pgr"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.buildSummarizerArgs()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildSummarizerArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}