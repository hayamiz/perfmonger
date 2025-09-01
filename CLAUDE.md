# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PerfMonger is a high-resolution system performance monitor written in Ruby (CLI) and Go (core monitoring components). It enables sub-second level monitoring of CPU, disk I/O, and network performance with JSON output for easy processing.

## Architecture

The project has a hybrid architecture:

1. **Ruby Layer** (`lib/`, `exe/`): CLI interface and command handling
   - Main entry point: `exe/perfmonger` → `lib/perfmonger/cli.rb`
   - Commands in `lib/perfmonger/command/`: record, play, plot, summary, live, etc.

2. **Go Core** (`core/`): Performance-critical monitoring binaries
   - `core/subsystem/`: Core monitoring logic (platform-specific implementations)
   - `core/cmd/`: Individual tools (recorder, player, summarizer, plot-formatter, viewer)
   - Binaries are built to `lib/exec/perfmonger-<tool>_<os>_<arch>`

3. **Platform Support**: Linux (primary) and macOS (experimental)
   - Platform-specific code: `perfmonger_linux.go`, `perfmonger_darwin.go`

## Development Commands

### Building
```bash
# Full build with cross-compilation for Linux/Darwin AMD64
bundle install
rake build

# Build only Go core components
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
cd core/subsystem && go test -v -cover

# Run Go static analysis
rake analyze_core
```

### Linting
```bash
# Go static analysis (included in default rake task)
cd core/subsystem && go vet perfmonger_<platform>.go $(ls *.go | grep -v perfmonger_)
cd core/cmd/<tool> && go vet *.go
```

## Key Implementation Details

- Performance data is collected using platform-specific system calls
- Output format is JSON for easy processing with tools like jq
- The recorder samples system metrics at configurable intervals (default 1s)
- Live monitoring uses termbox-go for terminal UI
- Plot formatting prepares data for gnuplot visualization

## Testing Individual Components

```bash
# Test recording
./exe/perfmonger record -i 0.1 -d sda

# Test playback
./exe/perfmonger play <recorded_file.pgr>

# Test live monitoring
./exe/perfmonger live
```