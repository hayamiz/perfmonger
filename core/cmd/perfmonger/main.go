package main

import (
	"os"

	"github.com/spf13/cobra"
)

const VERSION = "0.14.4"

// Ruby-compatible usage template
const usageTemplate = `Usage:
  {{.UseLine}}{{if .HasAvailableSubCommands}}

Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`

// Subcommand usage template (Ruby-like format)
const subCommandUsageTemplate = `Usage:
  {{.UseLine}}{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "perfmonger",
		Short:   "PerfMonger - System performance monitor",
		Version: VERSION,
		Long: `PerfMonger is a system performance monitoring tool that enables sub-second 
level monitoring of CPU, disk I/O, and network performance with JSON output.`,
	}

	// Custom version template
	cmd.SetVersionTemplate("PerfMonger version {{.Version}}\n")
	cmd.SetUsageTemplate(usageTemplate)

	// Add subcommands
	cmd.AddCommand(newLiveCommand())
	cmd.AddCommand(newRecordCommand()) 
	cmd.AddCommand(newPlayCommand())
	cmd.AddCommand(newStatCommand())
	cmd.AddCommand(newPlotCommand())
	cmd.AddCommand(newSummaryCommand())
	cmd.AddCommand(newFingerprintCommand())
	cmd.AddCommand(newInitShellCommand())
	cmd.AddCommand(newPlotFormatCommand())

	return cmd
}

// newLiveCommand is now defined in live.go

// newRecordCommand is now defined in record.go

// newPlayCommand is now defined in play.go

// newStatCommand is now defined in stat.go

// newPlotCommand is now defined in plot.go

// newSummaryCommand is now defined in summary.go

// newFingerprintCommand is now defined in fingerprint.go

// newInitShellCommand is now defined in initshell.go

func main() {
	rootCmd := newRootCommand()
	
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}