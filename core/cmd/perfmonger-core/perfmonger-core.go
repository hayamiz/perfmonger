//usr/bin/env go run $0 $@ ; exit

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/recorder"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/player"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/summarizer"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/plotformatter"
	"github.com/hayamiz/perfmonger/core/cmd/perfmonger-core/viewer"
)

func main() {
	// Get the program name (argv[0])
	progName := filepath.Base(os.Args[0])
	
	var subcommand string
	var args []string
	
	// Check if called via argv[0] compatibility (e.g., perfmonger-recorder)
	if strings.HasPrefix(progName, "perfmonger-") && !strings.HasPrefix(progName, "perfmonger-core") {
		// Extract subcommand from program name, handling suffixes like _linux_amd64
		subcommand = strings.TrimPrefix(progName, "perfmonger-")
		// Remove platform suffix if present
		if idx := strings.Index(subcommand, "_"); idx != -1 {
			subcommand = subcommand[:idx]
		}
		args = os.Args[1:] // All arguments except program name
	} else {
		// Called as perfmonger-core with subcommand
		if len(os.Args) < 2 {
			usage()
			os.Exit(1)
		}
		subcommand = os.Args[1]
		args = os.Args[2:] // All arguments except program name and subcommand
	}
	
	// Route to appropriate handler
	switch subcommand {
	case "record", "recorder":
		recorder.Run(args)
	case "play", "player":
		player.Run(args)
	case "summarize", "summarizer":
		summarizer.Run(args)
	case "plot-format", "plot-formatter":
		plotformatter.Run(args)
	case "view", "viewer":
		viewer.Run(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcommand)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: perfmonger-core <subcommand> [options]")
	fmt.Println("")
	fmt.Println("Available subcommands:")
	fmt.Println("  record, recorder       Record performance data")
	fmt.Println("  play, player          Play back recorded data")
	fmt.Println("  summarize, summarizer  Generate summary statistics")
	fmt.Println("  plot-format, plot-formatter  Format data for plotting")
	fmt.Println("  view, viewer          View performance data interactively")
	fmt.Println("")
	fmt.Println("Or call via argv[0] compatibility:")
	fmt.Println("  perfmonger-recorder, perfmonger-player, etc.")
}