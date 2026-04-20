package main

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewSummaryCommandStruct(t *testing.T) {
	cmd := newSummaryCommandStruct()

	if cmd.Pager != "" {
		t.Errorf("Pager = %q, want empty", cmd.Pager)
	}
	if cmd.SummaryOpt.JSON {
		t.Error("JSON should be false by default")
	}
	if cmd.SummaryOpt.Logfile != "" {
		t.Errorf("Logfile = %q, want empty", cmd.SummaryOpt.Logfile)
	}
}

func TestSummaryCommand_ValidateAndSetLogfile(t *testing.T) {
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
		wantTitle   string
	}{
		{
			name:    "no args",
			args:    []string{},
			wantErr: "PerfMonger log file is required",
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
			wantTitle:   tmpfile.Name(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newSummaryCommandStruct()
			err := cmd.validateAndSetLogfile(tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validateAndSetLogfile() unexpected error: %v", err)
				}
				if cmd.SummaryOpt.Logfile != tt.wantLogfile {
					t.Errorf("Logfile = %q, want %q", cmd.SummaryOpt.Logfile, tt.wantLogfile)
				}
				if cmd.SummaryOpt.Title != tt.wantTitle {
					t.Errorf("Title = %q, want %q", cmd.SummaryOpt.Title, tt.wantTitle)
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

func TestSummaryCommand_ValidatePagerOption(t *testing.T) {
	tests := []struct {
		name         string
		pagerEnv     string
		setPagerFlag bool
		wantErr      string
		wantPager    string
	}{
		{
			name:         "pager flag set with PAGER env",
			pagerEnv:     "less",
			setPagerFlag: true,
			wantErr:      "",
			wantPager:    "less",
		},
		{
			name:         "pager flag set without PAGER env",
			pagerEnv:     "",
			setPagerFlag: true,
			wantErr:      "no pager is available. Please set PAGER or give pager name to --pager option",
		},
		{
			name:         "pager flag not set",
			pagerEnv:     "",
			setPagerFlag: false,
			wantErr:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldPager := os.Getenv("PAGER")
			defer os.Setenv("PAGER", oldPager)

			if tt.pagerEnv != "" {
				os.Setenv("PAGER", tt.pagerEnv)
			} else {
				os.Unsetenv("PAGER")
			}

			summaryCmd := newSummaryCommandStruct()
			cobraCmd := &cobra.Command{}
			cobraCmd.Flags().StringVarP(&summaryCmd.Pager, "pager", "p", "", "")

			if tt.setPagerFlag {
				cobraCmd.Flags().Set("pager", "")
			}

			err := summaryCmd.validatePagerOption(cobraCmd)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validatePagerOption() unexpected error: %v", err)
				}
				if tt.wantPager != "" && summaryCmd.Pager != tt.wantPager {
					t.Errorf("Pager = %q, want %q", summaryCmd.Pager, tt.wantPager)
				}
			} else {
				if err == nil {
					t.Errorf("validatePagerOption() expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("validatePagerOption() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestNewSummaryCommand(t *testing.T) {
	cmd := newSummaryCommand()

	if cmd.Use != "summary [options] LOG_FILE" {
		t.Errorf("Use = %q, want %q", cmd.Use, "summary [options] LOG_FILE")
	}

	expectedFlags := []string{"json", "pager", "disk-only"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q to be defined", name)
		}
	}

	if cmd.Flags().ShorthandLookup("p") == nil {
		t.Error("expected short flag 'p' to be defined")
	}
}
