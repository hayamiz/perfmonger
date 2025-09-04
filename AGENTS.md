# AGENTS: Working Effectively in This Repo

This document guides coding agents (and humans) to explore, build, test, and modify this repository safely and productively.

## What This Repo Is

Perfmonger is a Ruby CLI that shells out to Go binaries for high‑resolution, holistic system performance monitoring and reporting (JSON output, summary, plotting, and a simple live server).

Core pieces:

- Ruby CLI and commands orchestrate behavior and UX.
- Go “core” binaries record, play, summarize, and format plot data.
- Assets for a minimal web dashboard and plotting via gnuplot/ImageMagick.

## Repository Layout

- CLI entry: `exe/perfmonger:1`
- Ruby entry and requires: `lib/perfmonger.rb:1`
- CLI runner: `lib/perfmonger/cli.rb:1`
- Command base and core bin resolver:
  - `lib/perfmonger/command/base_command.rb:1`
  - `lib/perfmonger/command/core.rb:1`
- Commands (patterns to follow):
  - `lib/perfmonger/command/record.rb:1`, `record_option.rb:1`
  - `lib/perfmonger/command/live.rb:1`
  - `lib/perfmonger/command/play.rb:1`
  - `lib/perfmonger/command/summary.rb:1`
  - `lib/perfmonger/command/plot.rb:1`
  - `lib/perfmonger/command/stat.rb:1`, `stat_option.rb:1`
  - `lib/perfmonger/command/server.rb:1`
- Go core (module `github.com/hayamiz/perfmonger/core`):
  - Binaries: `core/cmd/perfmonger-{recorder,player,viewer,summarizer,plot-formatter}/*.go`
  - Single-binary entry (Phase 1): `core/cmd/perfmonger-core/perfmonger-core.go`
  - Subcommand implementations (Phase 1): `core/internal/app/{recorder,player,summarizer,plotformatter,viewer}/*.go`
  - Subsystem models + readers: `core/subsystem/*.go`
  - Helpers: `core/utils.go:1`, `core/go.mod:1`
  - Build helpers: `core/build.sh:1`, generated `core/Makefile:1`
- Data/assets (server dashboard, licenses): `data/**`
- Specs (RSpec + sample data): `spec/**`
- Top level: `README.md`, `HOWTO.md`, `NEWS`, `Rakefile:1`, `perfmonger.gemspec:1`, `Gemfile`

## Prerequisites

- Ruby: 2.2+ recommended (gemspec says 1.9.3+, docs say 2.2+)
- Bundler: to install dev/test deps
- Go: 1.15+ (per `core/go.mod`), modern Go also works
- gnuplot: with pdfcairo terminal (for `perfmonger plot`)
- ImageMagick: for non‑PDF plot outputs (e.g., png)

## Setup

- Install Ruby deps: `bundle install`
- Build Go core (cross builds linux/darwin amd64 and drops binaries into `lib/exec`):
  - Phase 1: builds `perfmonger-core_<os>_<arch>`; compatibility wrappers (`perfmonger-{recorder,player,summarizer,plot-formatter}_<os>_<arch>`) invoke the core binary.
  - `bundle exec rake build` or `bundle exec rake cross_build_core`
  - Self‑build for current platform only: `bash core/build.sh -`

## Go Dev Env

- Load env: `source 00_LOAD_GO_DEVENV.sh`
  - Sets `GOROOT` (from system `go` if available), `GOPATH` to `./godevenv/gopath`, `GOBIN`, `GOCACHE`, `GOMODCACHE`, and prepends to `PATH`.
  - Creates local dirs under `./godevenv/` so builds/tests don’t pollute global caches.
  - Helper commands: `go_devenv_info` (show current settings), `go_devenv_unload` (restore previous env).
  - Note: Network may be restricted; the script defaults `GOPROXY=direct`. Adjust or pre‑populate module cache as needed.

Notes:

- Rake’s default task runs: Ruby specs, Go tests, Go vet. It also triggers a cross build first.
- Cross building for other OS/arch relies on Go’s cross‑compile; if CGO is introduced in the future you may need extra setup.

## Running

- Basic help: `exe/perfmonger --help`
- Live JSON to stdout: `exe/perfmonger live -i 0.5 --pretty`
- Record to file (gz by default): `exe/perfmonger record --timeout 3`
- Play a log (uses Go player): `exe/perfmonger play spec/data/busy100.pgr`
- Summary (text/JSON):
  - `exe/perfmonger summary spec/data/busy100.pgr`
  - `exe/perfmonger summary --json spec/data/busy100.pgr`
- Plot (requires gnuplot pdfcairo):
  - `exe/perfmonger plot -T pdf spec/data/busy100.pgr`
- Simple dashboard server:
  - `exe/perfmonger server -- -i 0.5`

## Testing & Analysis

- All at once (build + specs + Go tests + vet): `bundle exec rake`
- Ruby specs only: `bundle exec rspec`
- Go tests only: `go test ./core/subsystem`
- Static analysis (vet): `bundle exec rake analyze_core`

Notes:

- Some specs rely on `/proc` (Linux); they’re skipped when unavailable (`spec/spec_helper.rb:38`).
- Rake tasks will first build the Go binaries (cross build) so ensure Go toolchain is present.

## Coding Guidelines

Ruby (CLI and orchestration):

- Use `OptionParser` like existing commands do.
- Follow the `BaseCommand` pattern and register via `register_command`.
- Keep logic in commands thin; delegate heavy lifting to Go binaries.
- Avoid introducing new runtime Ruby deps; dev deps live in `Gemfile`/gemspec.

Go (core recording/playing/summarizing/plot formatting):

- Module is `github.com/hayamiz/perfmonger/core` (`core/go.mod:1`).
- Keep gob record format compatibility in mind when changing `core/subsystem/*.go` types.
- Prefer stdlib; existing deps are pinned in `core/go.mod`.
- Keep binaries no‑CGO if possible to retain simple cross builds.

Plotting/Web:

- `plot` shells out to gnuplot and optionally ImageMagick; detect tools and error clearly (`lib/perfmonger/command/plot.rb:1`).
- Dashboard assets are under `data/assets`; server wires routes in `lib/perfmonger/command/server.rb:1`.

## Adding a New CLI Command

1) Create `lib/perfmonger/command/<name>.rb` that subclasses `BaseCommand`, sets banner/options, and calls into a core binary (use `CoreFinder`).

2) Register it:

- Inside the class: `register_command '<name>', 'Short description'`

3) Ensure it’s required from `lib/perfmonger.rb`.

4) Add specs under `spec/` following existing examples (see `spec/play_spec.rb:1`, `spec/summary_spec.rb:1`).

5) If adding/altering a Go binary, wire it into `core/build.sh` (the script already enumerates known subcommands) and ensure `CoreFinder` can locate it by name.

## How Core Binaries Are Resolved

- `CoreFinder` looks for executables under `lib/exec/` with suffix `_<os>_<arch>` (e.g., `perfmonger-recorder_linux_amd64`). See `lib/perfmonger/command/core.rb:1`.
- Phase 1: these names are shell wrappers that dispatch to `perfmonger-core_<os>_<arch>`.
- Build scripts drop the core binary with the required name into that directory (see `core/build.sh:1` and generated `core/Makefile:1`).

## Common Gotchas

- Missing core binaries: build them first (`bundle exec rake build`), otherwise commands like `record`/`play`/`summary` fail.
- `/proc` access: some features/tests require Linux with `/proc` mounted.
- gnuplot availability: plotting requires gnuplot with pdfcairo; non‑PDF outputs require ImageMagick.
- Don’t forget to `require` new command files in `lib/perfmonger.rb`.
- Background recording uses per‑user PID files in `/tmp` (see `lib/perfmonger/command/record.rb:1` and `core/cmd/perfmonger-recorder/perfmonger-recorder.go:1`).

## Release Hints

- Gem metadata: `perfmonger.gemspec:1`.
- Typical flow (maintainers): `bundle exec rake build` then Bundler release tasks.
- CI configs exist historically (Travis/Wercker), may be outdated.

## Acceptance Checklist for Changes

- Command works via `exe/perfmonger <cmd> --help` and error handling is clear.
- Go and Ruby tests pass locally (`bundle exec rake`).
- New/changed options are reflected in help banners and README/HOWTO when applicable.
- No regressions in gob format or JSON schema unless explicitly versioned/communicated.

## Dev Docs

- `devdoc/TODO.md:1`: Backlog of upcoming development items. Use it to capture planned work, ideas, and prioritized tasks. Keep entries concise; link to files/lines and issues where relevant.
- `devdoc/DONE.md:1`: Chronicle of completed items moved from TODO for record‑keeping. When you finish a task from TODO, copy its entry here with a completion date and any noteworthy context or follow‑ups.
