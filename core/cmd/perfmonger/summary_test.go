package main

import (
	"os"
	"testing"
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

func TestSummaryCommand_ResolvePager(t *testing.T) {
	tests := []struct {
		name      string
		pagerFlag string // value set on cmd.Pager (empty means --pager not given)
		setEnv    bool   // whether to set PAGER env var
		pagerEnv  string // PAGER env value when setEnv is true
		want      string
	}{
		{
			name:      "pager flag overrides PAGER env",
			pagerFlag: "more",
			setEnv:    true,
			pagerEnv:  "less",
			want:      "more",
		},
		{
			name:      "PAGER env used when flag empty",
			pagerFlag: "",
			setEnv:    true,
			pagerEnv:  "less",
			want:      "less",
		},
		{
			name:      "no pager when flag empty and PAGER unset",
			pagerFlag: "",
			setEnv:    false,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldPager, hadPager := os.LookupEnv("PAGER")
			defer func() {
				if hadPager {
					os.Setenv("PAGER", oldPager)
				} else {
					os.Unsetenv("PAGER")
				}
			}()

			if tt.setEnv {
				os.Setenv("PAGER", tt.pagerEnv)
			} else {
				os.Unsetenv("PAGER")
			}

			summaryCmd := newSummaryCommandStruct()
			summaryCmd.Pager = tt.pagerFlag

			got := summaryCmd.resolvePager()
			if got != tt.want {
				t.Errorf("resolvePager() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewSummaryCommand(t *testing.T) {
	cmd := newSummaryCommand()

	if cmd.Use != "summary [options] LOG_FILE" {
		t.Errorf("Use = %q, want %q", cmd.Use, "summary [options] LOG_FILE")
	}

	expectedFlags := []string{"json", "pager", "no-pager", "disk-only"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q to be defined", name)
		}
	}

	if cmd.Flags().ShorthandLookup("p") == nil {
		t.Error("expected short flag 'p' to be defined")
	}
}
