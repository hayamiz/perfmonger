package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/player"
)

// playCommand represents the play command with direct PlayerOption setting
type playCommand struct {
	// Direct field (no embedding) for maximum efficiency
	PlayerOpt *player.PlayerOption
	
	// No Ruby-specific options for play command
}

// newPlayCommandStruct creates playCommand with Ruby-compatible defaults
func newPlayCommandStruct() *playCommand {
	return &playCommand{
		PlayerOpt: player.NewPlayerOption(),
	}
}

// validateAndSetLogfile validates the logfile argument using cobra's PreRunE approach
func (cmd *playCommand) validateAndSetLogfile(args []string) error {
	if len(args) == 0 {
		// No file argument: read from stdin (default Logfile is "-")
		return nil
	}

	// Take the first argument as log file
	cmd.PlayerOpt.Logfile = args[0]

	// Check if file exists
	if _, err := os.Stat(cmd.PlayerOpt.Logfile); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", cmd.PlayerOpt.Logfile)
	}

	return nil
}

// run executes the play command with direct API calls
func (cmd *playCommand) run() error {
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running player with options: %+v\n", cmd.PlayerOpt)
	}
	
	// Direct API call - no conversion needed
	player.RunWithOption(cmd.PlayerOpt)
	return nil
}


// newPlayCommand creates the play subcommand with direct cobra setting
func newPlayCommand() *cobra.Command {
	playCmd := newPlayCommandStruct()
	
	cmd := &cobra.Command{
		Use:   "play [options] LOG_FILE",
		Short: "Play a recorded perfmonger session",
		Long:  `Play a perfmonger log file in JSON`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validation moved to PreRunE for cobra integration
			return playCmd.validateAndSetLogfile(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Direct execution - no additional validation needed
			return playCmd.run()
		},
	}
	
	// Direct cobra flag setting to PlayerOption fields (no conversion needed)
	cmd.Flags().BoolVarP(&playCmd.PlayerOpt.Color, "color", "c", playCmd.PlayerOpt.Color,
		"Use colored JSON output")
	cmd.Flags().BoolVarP(&playCmd.PlayerOpt.Pretty, "pretty", "p", playCmd.PlayerOpt.Pretty,
		"Use human readable JSON output")
	cmd.Flags().StringVar(&playCmd.PlayerOpt.DiskOnly, "disk-only", playCmd.PlayerOpt.DiskOnly,
		"Select disk devices that matches REGEX (Ex. 'sd[b-d]')")
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}