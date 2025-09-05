# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PerfMonger is a high-resolution system performance monitor written in Ruby (CLI) and Go (core monitoring components). It enables sub-second level monitoring of CPU, disk I/O, and network performance with JSON output for easy processing.

## Architecture

The project has a hybrid architecture (currently transitioning to single-binary Go implementation):

1. **Ruby Layer** (`lib/`, `exe/`): CLI interface and command handling
   - Main entry point: `exe/perfmonger` → `lib/perfmonger/cli.rb`
   - Commands in `lib/perfmonger/command/`: record, play, plot, summary, live, etc.

2. **Go Core** (`core/`): Performance-critical monitoring components (unified into single binary)
   - `core/internal/perfmonger/`: Core monitoring logic (platform-specific implementations)
   - `core/cmd/perfmonger-core/`: Unified single binary with subcommands (record, play, summarize, plot-format, view)
   - Individual tool packages: recorder, player, summarizer, plotformatter, viewer
   - Binaries built to `lib/exec/perfmonger-core_<os>_<arch>` with compatibility wrappers

3. **Platform Support**: Linux only (Darwin support removed)
   - Platform-specific code: `perfmonger_linux.go`

## Development Commands

### Building
```bash
# Full build with Ruby CLI + unified Go core binary
bundle install
rake build

# Build only Go core components (single perfmonger-core binary)
cd core && ./build.sh

# Self-build (current platform only)
cd core && ./build.sh -

# Optional: Use isolated Go environment
source 00_LOAD_GO_DEVENV.sh
```

### Testing
```bash
# Run all tests (Ruby specs + Go tests + static analysis)
rake

# Run Ruby specs only
rake spec

# Run Go tests only
rake test_core
# or directly:
cd core/internal/perfmonger && go test -v -cover

# Run Go static analysis
rake analyze_core
```

### Linting
```bash
# Go static analysis (included in default rake task)
cd core/internal/perfmonger && go vet perfmonger_linux.go $(ls *.go | grep -v perfmonger_)
cd core/cmd/perfmonger-core && go vet *.go
cd core/cmd/perfmonger-core/<tool> && go vet *.go
```

## Key Implementation Details

- Performance data is collected using platform-specific system calls
- Output format is JSON for easy processing with tools like jq
- The recorder samples system metrics at configurable intervals (default 1s)
- Live monitoring uses termbox-go for terminal UI
- Plot formatting prepares data for gnuplot visualization

## Testing Individual Components

```bash
# Test recording (via Ruby CLI)
./exe/perfmonger record -i 0.1 -d sda

# Test playback (via Ruby CLI)
./exe/perfmonger play <recorded_file.pgr>

# Test live monitoring (via Ruby CLI)
./exe/perfmonger live

# Test core components directly (single binary with subcommands)
./lib/exec/perfmonger-core_linux_amd64 record -i 0.1 -d sda
./lib/exec/perfmonger-core_linux_amd64 play <recorded_file.pgr>
./lib/exec/perfmonger-core_linux_amd64 plot-format -perfmonger <file.pgr>
```

## Development Documentation

- `devdoc/TODO.md`: Contains implementation tasks to be completed. Updated this file as tasks progress.
- `devdoc/DONE.md`: Stores completed tasks moved from TODO.md for record keeping.