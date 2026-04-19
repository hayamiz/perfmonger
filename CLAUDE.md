# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PerfMonger is a high-resolution system performance monitor written in Go. It enables sub-second level monitoring of CPU, disk I/O, and network performance with JSON output for easy processing.

## Architecture

The project is a single-binary Go implementation:

1. **Go CLI** (`core/cmd/perfmonger/`): Top-level `perfmonger` binary with cobra subcommands
   - Subcommands: record, play, summary, plot, live, stat, fingerprint, init-shell
   - Built to `lib/exec/perfmonger_linux_amd64`

2. **Go Core Packages** (`core/cmd/perfmonger-core/`): Reusable component packages
   - `recorder/`, `player/`, `summarizer/`, `plotformatter/`, `viewer/`
   - Each exposes `RunDirect(option)` / `RunWithOption(option)` APIs

3. **Core Monitoring Logic** (`core/internal/perfmonger/`): Platform-specific system metric collection
   - Platform-specific code: `perfmonger_linux.go`
   - Linux only (Darwin support removed)

4. **Legacy Ruby Code** (`lib/`, `exe/`, `spec/`): Not actively used; retained for reference during migration cleanup

## Development Commands

### Building
```bash
# Build the unified Go binary (perfmonger_linux_amd64)
cd core/cmd/perfmonger && go build -o ../../../lib/exec/perfmonger_linux_amd64 .

# Build core components (perfmonger-core binary, compatibility wrappers)
cd core && ./build.sh

# Self-build (current platform only)
cd core && ./build.sh -

# Optional: Use isolated Go environment
source 00_LOAD_GO_DEVENV.sh
```

### Testing
```bash
# Run integration tests (pytest, recommended)
uv sync && uv run pytest -v

# Run a specific test file
uv run pytest tests/test_record.py -v

# Run Go unit tests
cd core/internal/perfmonger && go test -v -cover

# Run Go static analysis
cd core/internal/perfmonger && go vet perfmonger_linux.go $(ls *.go | grep -v perfmonger_)
```

### Integration Test Framework (pytest)

Integration tests live in `tests/` and use **pytest** with **uv** for dependency management.

- **`pyproject.toml`**: Project config with pytest + pytest-timeout dependencies
- **`tests/conftest.py`**: Shared fixtures (`perfmonger_bin`, `data_file`, `run_perfmonger()`)
- **Test files**: `tests/test_*.py` — one per subcommand
- **Test data**: `spec/data/` contains sample `.pgr` and `.pgr.gz` files used as golden fixtures

Key conventions:
- Tests that exercise known bugs use `@pytest.mark.xfail(reason="...")`
- Tests requiring `/proc/diskstats` use `@requires_proc_diskstats` skip marker
- Tests requiring `gnuplot` use `@requires_gnuplot` skip marker
- `run_perfmonger(*args, cwd=...)` helper runs the Go binary and returns `CompletedProcess`
- Background mode tests (`test_background.py`) manage session files and cleanup via fixtures

## Key Implementation Details

- Performance data is collected using platform-specific system calls
- Output format is JSON for easy processing with tools like jq
- The recorder samples system metrics at configurable intervals (default 1s)
- Live monitoring uses termbox-go for terminal UI
- Plot formatting prepares data for gnuplot visualization

## Testing Individual Components

```bash
# Record system metrics
./lib/exec/perfmonger_linux_amd64 record -i 0.1 -d sda --timeout 5

# Playback recorded data as JSON
./lib/exec/perfmonger_linux_amd64 play <recorded_file.pgr>

# Live monitoring
./lib/exec/perfmonger_linux_amd64 live

# Background recording with session management
./lib/exec/perfmonger_linux_amd64 record --background --timeout 60
./lib/exec/perfmonger_linux_amd64 record --status
./lib/exec/perfmonger_linux_amd64 record --kill
```

## Development Documentation

- `devdoc/TODO.md`: Contains implementation tasks to be completed. Updated this file as tasks progress.
- `devdoc/DONE.md`: Stores completed tasks moved from TODO.md for record keeping.

## Code Style Guidelines

- **Comments**: ALL comments in code must be written in English only. No Japanese or other languages are allowed in code comments.
- **Variable/Function Names**: Use descriptive English names for variables, functions, and types.
- **Documentation**: Code documentation and docstrings should be in English to maintain consistency and readability for international contributors.
## Devcontainer

This project uses a devcontainer for a consistent development environment.

### Files

- `.devcontainer/Dockerfile` — Container image definition. Edit to change the base image, install additional packages, or add language runtimes.
- `.devcontainer/devcontainer.json` — Devcontainer configuration. Controls the container name, VS Code extensions, environment variables, bind mounts, and port forwarding.
- `.devcontainer/inside-container.settings.local.json` — Claude Code local permission overrides inside the container. Mounted as `.claude/settings.local.json` to override project-level settings.
- `.devcontainer/build.sh` — Builds the container image. Run this after modifying the Dockerfile.
- `.devcontainer/shell.sh` — Starts the container and opens an interactive bash shell inside it.

### Usage

```bash
# Build the container image
.devcontainer/build.sh

# Start the container and open a shell
.devcontainer/shell.sh
```
