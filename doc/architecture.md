# PerfMonger Architecture & Behavior

This document describes the current implementation of PerfMonger — the Go
re-implementation that replaces the original Ruby CLI. It is intended for
contributors who need a single reference covering the CLI surface, the data
pipeline, the on-disk formats, and the runtime behavior of each subcommand.

Everything below reflects the state of `master` at the time of writing. Where
the code has TODOs, edge cases, or legacy quirks, this document calls them out
explicitly instead of papering over them.

---

## 1. Repository Layout

```
perfmonger/
├── core/
│   ├── cmd/
│   │   ├── perfmonger/              # Unified user-facing CLI (cobra)
│   │   │   ├── main.go              # Root cobra command, VERSION
│   │   │   ├── record.go / live.go / play.go / stat.go / plot.go
│   │   │   ├── summary.go / fingerprint.go / initshell.go
│   │   │   └── godevenv/            # Isolated Go toolchain (optional)
│   │   └── perfmonger-core/         # Reusable component packages
│   │       ├── recorder/            # RecorderOption + RunDirect
│   │       ├── player/              # PlayerOption + RunDirect
│   │       ├── summarizer/          # SummaryOption + RunDirect
│   │       ├── plotformatter/       # PlotFormatOption + RunDirect
│   │       └── viewer/              # gocui-based TUI (placeholder)
│   └── internal/
│       └── perfmonger/              # Linux /proc readers + stat types
├── lib/exec/                        # Build output (perfmonger_<os>_<arch>, …)
├── tests/                           # pytest integration tests
├── spec/data/                       # Golden .pgr fixtures
├── doc/
│   ├── architecture.md              # (this file)
│   └── tickets/                     # File-based work items
└── Makefile                         # Top-level build entry point
```

The `lib/` and `spec/` Ruby scaffolding are retained for historical reference;
`spec/data/` is the only part still used (by pytest fixtures). The Ruby gem
code itself is not exercised by any current test or build target. Top-level
`data/`, `dbg/`, `misc/`, `apm_modules/`, and the repo-root `godevenv/` (the
isolated Go toolchain that `00_LOAD_GO_DEVENV.sh` populates) are outside the
Go build graph; `core-old/` contains an earlier Go attempt that is not built.

Note that `core/cmd/perfmonger/godevenv/` shown in the tree above is an
unrelated empty scaffold — the active isolated-toolchain directory lives at
the repo root.

---

## 2. Binaries

The Makefile produces a single binary per target platform.

### 2.1 `perfmonger_<GOOS>_<GOARCH>` — the unified CLI

Single cobra-powered binary that end users invoke. `VERSION` is declared in
[core/cmd/perfmonger/main.go](../core/cmd/perfmonger/main.go) and currently
sits at `0.14.4`. Subcommands:

| Subcommand                       | File            | Purpose                                                  |
|----------------------------------|-----------------|----------------------------------------------------------|
| `live`                           | `live.go`       | Record + play JSON in real time                          |
| `record`                         | `record.go`     | Persist a recording to `.pgr` / `.pgr.gz`                |
| `play`                           | `play.go`       | Stream a recording as JSON lines                         |
| `stat`                           | `stat.go`       | Record while `-- <cmd>` runs, then print a summary       |
| `plot`                           | `plot.go`       | Invoke plotformatter + gnuplot to produce PDF/PNG charts |
| `summary` (alias `summarize`)    | `summary.go`    | Human/JSON aggregate over a whole recording              |
| `fingerprint` (alias `bukko`,`fp`)| `fingerprint.go`| Collect system config into a `.tar.gz`                   |
| `init-shell`                     | `initshell.go`  | Emit bash/zsh completion scriptlet                       |

`cobra` is configured with Ruby-compatible usage templates so `--help` output
mirrors the old Ruby CLI layout.

### 2.2 Build targets

`Makefile` drives everything:

| Target        | What it does                                                                 |
|---------------|------------------------------------------------------------------------------|
| `build` (default) | Build the `perfmonger` binary for the host platform                      |
| `test`        | `go test -v -cover` for `core/internal/perfmonger`, then `uv run pytest -v`  |
| `vet`         | `go vet` on Linux-specific sources in `core/internal/perfmonger`             |
| `cross-build` | Runs `build` twice: `GOOS=linux GOARCH=amd64` and `GOOS=linux GOARCH=arm64`  |
| `clean`       | Remove all `lib/exec/perfmonger*` artifacts                                  |

Only Linux is supported today. There is no Darwin implementation in the
current tree — a previous attempt lives under `core-old/` (not built, not
tested) and should not be treated as production code.

---

## 3. Internal Monitoring Layer — `core/internal/perfmonger`

This package is the data-collection foundation used by every other component.
It has no CLI of its own.

### 3.1 File layout

- [perfmonger.go](../core/internal/perfmonger/perfmonger.go) — `CommonHeader`,
  `PlatformType` constants (`Linux = 1`), `LinuxHeader`, `LinuxDevice`
- [perfmonger_linux.go](../core/internal/perfmonger/perfmonger_linux.go) —
  Linux-specific readers for `/proc/{stat,diskstats,net/dev,meminfo,interrupts}`
- [stat.go](../core/internal/perfmonger/stat.go) — per-sample record types
- [usage.go](../core/internal/perfmonger/usage.go) — delta/usage computations

### 3.2 Record types

`StatRecord` is the per-sample unit that the recorder encodes and the player
decodes. Its fields for the major metric groups are all pointers
(`*CpuStat`, `*InterruptStat`, `*SoftIrqStat`, `*DiskStat`, `*NetStat`,
`*MemStat`, `*ProcStat`) so a recording can omit any of them based on
`NoCPU` / `NoIntr` / `NoDisk` / `NoNet` / `NoMem` flags. Note that
`CpuStat.All` is embedded by **value** as a `CpuCoreStat`, not a pointer.

| Type              | Content                                                                 |
|-------------------|-------------------------------------------------------------------------|
| `CpuCoreStat`     | Raw jiffy counters (User, Nice, Sys, Idle, Iowait, Hardirq, Softirq, Steal, Guest, GuestNice) |
| `CpuStat`         | `All` (value) + `NumCore` + `CoreStats[]`                               |
| `InterruptStat`   | `NumEntries` + `Entries[]` (per-core counts + IRQ metadata)             |
| `SoftIrqStat`     | Per-category softirq counters; declared on `StatRecord` but **not populated** by any current Linux reader |
| `DiskStat`        | `Entries[]` with per-device read/write IOs, merges, sectors, ticks, queue depth |
| `NetStat`         | `Entries[]` per interface: rx/tx bytes/packets/errors/drops/fifo/frame/compressed/multicast |
| `MemStat`         | Every field exposed by `/proc/meminfo` in KB                            |
| `ProcStat`        | Context switches and fork count; declared but **not populated** by the Linux readers |
| `StatRecord`      | `Time time.Time` + pointers to each of the above                        |

### 3.3 Collection functions

Each reader takes a `*StatRecord` and fills in one pointer field. They are all
invoked once per sampling tick in the recorder loop:

- `ReadCpuStat` — parses `/proc/stat` "cpu" + "cpuN" lines, tolerates kernel
  variants that omit newer columns (Guest, GuestNice, etc.).
- `ReadInterruptStat` — parses `/proc/interrupts`; distinguishes device IRQs
  from system IRQs (NMI, LOC, TLB, …).
- `ReadDiskStats` — parses `/proc/diskstats`, supports both the classic
  14-field format and the legacy 7-field (partition) format, filters by the
  optional `TargetDisks` map.
- `ReadNetStat` — parses `/proc/net/dev`, skipping the two header rows.
- `ReadMemStat` — parses `/proc/meminfo` into `MemStat` fields by name.

`NewPlatformHeader()` populates the `LinuxHeader` by walking `/proc/diskstats`
+ `/sys/block/*` to classify physical devices vs. partitions.

### 3.4 Usage computation (`usage.go`)

The raw counters are cumulative. The player and summarizer call delta helpers
to produce per-interval rates before emitting JSON:

- `GetCpuCoreUsage(prev, curr)` → `CpuCoreUsage` with `{User, Nice, Sys, Idle,
  Iowait, Hardirq, Softirq, Steal, Guest, GuestNice}` as percentages.
- `GetCpuUsage(prev, curr)` → aggregate `CpuUsage` (all cores + per-core).
- `GetInterruptUsage(t1, i1, t2, i2)` → per-core interrupt rates split into
  Device vs. System categories.
- `GetDiskUsage1(t1, d1, t2, d2, regex)` → per-device IOPS, throughput
  (sectors/sec internally; the JSON layer reports KiB/s as `sectors/2.0`),
  average latency (`ticks/ops` in ms), average request size in sectors,
  queue length. The optional `regex` filter matches device names. A
  synthetic `"total"` entry is appended to the returned map.
- `GetNetUsage(...)` → per-interface rx/tx bytes/packets/errors/drops/fifo/
  frame/compressed/multicast per second; only the first four (bytes, pkts,
  errs, drops) are surfaced in the JSON output. Also appends a `"total"`
  aggregate entry.
- `GetMemUsage(mem)` → wrapper around the latest `MemStat` snapshot.

### 3.5 On-disk binary format — `.pgr`

Every recording is a `encoding/gob` stream with the following structure:

```
1. CommonHeader            (Platform tag, Hostname, StartTime)
2. PlatformHeader          (LinuxHeader: device list + partition map)
3. StatRecord, StatRecord, …   // repeated until EOF
```

`NewPerfmongerLogReader()` sniffs the first two bytes (`0x1f 0x8b`) to
auto-detect gzip compression, so callers treat `.pgr` and `.pgr.gz`
identically. There is no explicit EOF marker; readers stop on `io.EOF` from
the gob decoder.

No schema version is written into the header today. Backward compatibility
across gob layout changes would have to be added explicitly (e.g., new field
in `CommonHeader`) if the type definitions ever change incompatibly.

---

## 4. Core Reusable Packages — `core/cmd/perfmonger-core/*`

Each of these packages exposes a `RunDirect(option)` entry point that the
top-level cobra commands call into. Some also retain a `Run(args []string)`
that parses its own argument list; keeping `RunDirect` separate avoids
re-parsing arguments when invoked from Go code.

### 4.1 `recorder`

[core/cmd/perfmonger-core/recorder/recorder.go](../core/cmd/perfmonger-core/recorder/recorder.go)

`RecorderOption` fields:

| Field                | Purpose                                                        |
|----------------------|----------------------------------------------------------------|
| `Interval`           | Sampling period (base). Default `1s`.                          |
| `NoIntervalBackoff`  | Disable the 2×-every-1000-samples slow-down.                   |
| `Timeout`            | Total recording duration. `0` means infinite.                  |
| `StartDelay`         | Sleep before the first sample.                                 |
| `DevsParts`          | `-d` disk name list (resolved to `TargetDisks`).               |
| `Output`             | Output path, or `-` for stdout.                                |
| `NoCPU`/`NoIntr`/`NoDisk`/`NoNet`/`NoMem` | Feature toggles.                          |
| `Debug`              | Dumps the option struct to stderr.                             |
| `ListDevices`        | Prints device list to stderr and returns.                      |
| `PlayerBin` + `PlayerArgs` | When set, the recorder pipes its gob stream into a child player process (used by `live`). |
| `Gzip`               | Wrap output in `gzip.Writer`. Only applied when writing to a file; ignored when piping into a player. |
| `Color` / `Pretty`   | Forwarded to the child player.                                 |
| `Background`         | Tells `RunDirect` to write the session PID file.               |
| `StopCh`             | External stop channel (used by `stat`).                        |

`RunDirect` flow (single loop in [recorder.go:257-488](../core/cmd/perfmonger-core/recorder/recorder.go#L257-L488)):

1. If `Background`, call `WriteSessionFile()` and defer `RemoveSessionFile()`.
2. Build `CommonHeader` + `PlatformHeader`, handle `ListDevices` early-exit.
3. If `PlayerBin` is set, `exec.Command` the player, pipe its stdout back to
   `os.Stdout` via a goroutine, and attach its stdin as an additional writer.
4. Open output: stdout → `bufio.Writer`; `-` + player → player stdin only;
   file + player → `io.MultiWriter(file, player_stdin)`; file w/o player →
   optional `gzip.Writer` wrapped in `bufio.Writer`.
5. Gob-encode headers, sleep `StartDelay`, then enter the sample loop:
   - Fill `record.Time`, call each enabled reader.
   - `enc.Encode(record)` and `out.Flush()`.
   - Apply interval backoff: every `BACKOFF_THRESH=1000` samples, multiply
     `Interval` by `BACKOFF_RATIO=2.0`, capped at one hour.
   - `select` on `sigint_ch`, `timeout_ch`, `stopCh` (nil-safe), and the tick.
   - If the next scheduled time is within 10ms of the timeout deadline, treat
     the current sample as the last one to avoid a degenerate final interval.
6. On exit: flush, close player stdin, wait for player.

One observable consequence: the recorder writes at least two records before
the player can emit anything, because the player is delta-based.

### 4.2 `player`

[core/cmd/perfmonger-core/player/player.go](../core/cmd/perfmonger-core/player/player.go)

`PlayerOption`: `Logfile` (`-` for stdin), `Color`, `Pretty`, `DiskOnly` +
compiled `DiskOnlyRegex`.

`RunDirect` reads the gob stream via `NewPerfmongerLogReader`, decodes
`CommonHeader`, `PlatformHeader`, and then maintains a rolling
`records[2]` buffer. For each pair `(prev, curr)` it emits one JSON object
per line. The very first decoded record is stashed as `init_rec` to compute
`elapsed_time`. Output uses `go-projson` so `--pretty` and `--color` are just
projson style toggles.

JSON shape per line (exact key names from `WriteJsonTo` methods in
[usage.go](../core/internal/perfmonger/usage.go)):

```json
{
  "time": 1712345678.901,
  "elapsed_time": 12.345,
  "cpu":  {
    "num_core": 8,
    "all":   { "usr": 12.3, "nice": 0.0, "sys": 4.5, "idle": 78.9,
               "iowait": 0.1, "hardirq": 0.0, "softirq": 0.2,
               "steal": 0.0, "guest": 0.0, "guestnice": 0.0 },
    "cores": [ { "usr": ..., ... }, ... ]
  },
  "intr": { "core_dev_intr": [12.0, 3.0, ...], "core_sys_intr": [0.0, ...] },
  "disk": {
    "devices": ["sda", "sdb"],
    "sda":   { "riops": 100.0, "wiops": 50.0, "rkbyteps": 512.0,
               "wkbyteps": 256.0, "rlatency": 0.123, "wlatency": 0.456,
               "rsize": 8.0, "wsize": 4.0, "qlen": 1.23 },
    "sdb":   { ... },
    "total": { ... }
  },
  "net": {
    "devices": ["eth0"],
    "eth0":  { "rxkbyteps": 1000.0, "rxpktps": 10.0, "rxerrps": 0.0,
               "rxdropps": 0.0, "txkbyteps": 500.0, "txpktps": 5.0,
               "txerrps": 0.0, "txdropps": 0.0 },
    "total": { ... }
  },
  "mem":  { "mem_total": 16777216, "mem_used": ..., "mem_free": 8388608,
            "buffers": ..., "cached": ..., "swap_cached": ..., "active": ...,
            "inactive": ..., "swap_total": ..., "swap_free": ...,
            "dirty": ..., "writeback": ..., "anon_pages": ..., "mapped": ...,
            "shmem": ..., "slab": ..., /* and more */ }
}
```

Notable shape details the casual reader will miss:

- **Disk and net blocks are not plain device maps.** Each contains a
  `"devices"` array listing the device names alphabetically, plus one key per
  device, plus an auto-added `"total"` aggregate entry. Consumers iterating
  over keys should filter `"devices"` and `"total"` out (or use the
  `"devices"` array as the source of truth).
- **Disk throughput is in KiB/s**, not bytes/s — `rkbyteps`/`wkbyteps` are
  computed as `RdSecps / 2.0` assuming 512-byte sectors.
- **Net throughput is also in KiB/s** (`rxkbyteps`, `txkbyteps`). Only bytes,
  packets, errors, and drops are emitted; fifo/frame/compressed/multicast are
  collected into the record but not surfaced in JSON.
- **`intr` only exposes per-core totals** split into device vs. system
  interrupts; no per-IRQ breakdown reaches the JSON output.
- **CPU keys use abbreviated names** (`usr`, `sys`, `iowait`, `guestnice`,
  …), not the camel-case Go field names.

Optional keys (`cpu`, `intr`, `disk`, `net`, `mem`) are present only if both
records have non-nil pointers for that category. Errors from individual
sub-stat formatters cause that JSON object to be skipped (printed `skip by
err` to stderr) rather than aborting the whole stream.

Concurrency caveat: the player stashes the first decoded record in a
package-global `init_rec` to compute `elapsed_time`. Two concurrent
`RunDirect` calls in the same Go process would race on that global — safe
today because the CLI only ever runs one player per process, but something
to keep in mind if you embed the package.

### 4.3 `summarizer`

[core/cmd/perfmonger-core/summarizer/summarizer.go](../core/cmd/perfmonger-core/summarizer/summarizer.go)

`SummaryOption`: `Logfile`, `Title`, `JSON`, `DiskOnly` + compiled regex.

`RunDirect(option, out io.Writer) error` reads the full gob stream keeping
only the first record and a rolling last-two buffer. The summary is one
aggregate delta between first and last record, not a per-interval average.
`Duration` is `lst_record.Time - fst_record.Time`.

Text output (default) includes CPU usage block, per-device disk stats,
per-interface network stats, and memory. JSON output emits a single object
keyed by `exectime`, `cpu`, `intr`, `disk`, `net` — `mem` is not included in
the JSON form (see §9 for notes on this and similar asymmetries).

### 4.4 `plotformatter`

[core/cmd/perfmonger-core/plotformatter/plotformatter.go](../core/cmd/perfmonger-core/plotformatter/plotformatter.go)

`PlotFormatOption`: paths to output `.dat` files (`DiskFile`, `CpuFile`,
`MemFile`), input `PerfmongerFile`, and optional `DiskOnly` regex.

`RunDirect` writes tab-separated `.dat` files ready for gnuplot:
- `disk.dat` — one row per sample, columns indexed by device order.
- `cpu.dat` — aggregate CPU plus per-core columns.
- `mem.dat` — memory metrics per sample.

It returns a `PlotMeta` describing device indices, core count, and the time
range. `plot.go` in the CLI uses this metadata to generate the gnuplot script
it then feeds to `gnuplot`.

### 4.5 `viewer`

[core/cmd/perfmonger-core/viewer/viewer.go](../core/cmd/perfmonger-core/viewer/viewer.go)

A `gocui` + `termbox-go` scaffold with keybindings (`q`/`Ctrl-C`, `c` to
toggle color) and Unicode bar-chart primitives. It is **not wired into any
top-level subcommand** in the current CLI; the `live` subcommand streams JSON
via the `player` child process instead of a TUI. Treat `viewer` as
experimental / unfinished.

---

## 5. CLI Subcommand Behavior

This section documents the observable behavior of each subcommand, including
the quirks callers need to know about.

### 5.1 `record`

Defaults (set in `newRecordCommandStruct()`):

- `--logfile` = `perfmonger.pgr.gz`
- `--interval` = `1s`
- `--no-net` = true (matches the Ruby default; net must be opted in via
  `--no-net=false`)
- Gzip on, interrupts recording off (`--record-intr` opts in)

Flags (all accept Ruby-style float seconds thanks to `secondsDurationValue`):

| Flag                    | Effect                                                     |
|-------------------------|------------------------------------------------------------|
| `-d`, `--disk`          | Repeatable; device names to monitor.                       |
| `-l`, `--logfile`       | Output path. If `.gz` suffix is present and `--no-gzip` is set, the suffix is stripped. |
| `-i`, `--interval`      | Base sampling interval.                                    |
| `-s`, `--start-delay`   | Delay before first sample.                                 |
| `-t`, `--timeout`       | Total duration. `0` = forever.                             |
| `--background`          | Detach via re-exec (see §6).                               |
| `--record-intr`         | Enable `/proc/interrupts` sampling (experimental).         |
| `--no-cpu`/`--no-net`/`--no-mem` | Feature toggles.                                  |
| `--no-gzip`             | Write raw `.pgr` instead of gzip-wrapped.                  |
| `--no-interval-backoff` | Disable the automatic interval doubling.                   |
| `--kill`                | SIGINT any running background session, with exponential backoff (50ms×2 up to five tries). |
| `--status`              | Print PID, cmdline, start time of the running background session. |
| `-v`, `--verbose`       | Sets `RecorderOpt.Debug` via env `PERFMONGER_DEBUG`; otherwise verbose is currently advisory only. |

Validation:
- `--kill` and `--status` are mutually exclusive.
- `--timeout` / `--start-delay` must be non-negative; `--interval` must be > 0.
- Before launching a background session, the CLI checks for an existing
  session PID and refuses to start if one is alive.

Output path is resolved to absolute before daemonizing because the child
process is re-exec'd with `cwd=/`.

### 5.2 `live`

Wraps `record` but forces `Output="-"` and sets `PlayerBin=<self>
PlayerArgs=["play"]`. Exposed flags: `-d`, `-i`, `-s`, `-t`,
`--record-intr`, `--no-cpu`, `--no-net`, `--no-mem`, `--no-gzip`,
`-c`/`--color`, `--pretty`, `-v`/`--verbose`. Missing (by design): no
`-l`/`--logfile`, no `--background`, no `--kill`/`--status`, no
`--no-interval-backoff`. `--color` and `--pretty` are propagated to the
child player. Network recording is still off by default.

### 5.3 `play`

Args: optional `LOG_FILE` (defaults to stdin).

Flags: `-c`/`--color`, `-p`/`--pretty`, `--disk-only <regex>`.

Panics on I/O errors from the input file (see §9). Gzip is auto-detected.

### 5.4 `stat`

Usage: `perfmonger stat [options] -- <command> [args...]`. Flags mirror
`record` (`-d`, `-l`/`--logfile`, `-i`, `-s`, `-t`, `--record-intr`,
`--no-cpu`/`--no-net`/`--no-mem`, `--no-gzip`, `--no-interval-backoff`,
`-v`/`--verbose`) plus `--json` for the summary output.

Defaults worth noting:

- `-l` defaults to `./perfmonger.pgr` — i.e., a file in the current working
  directory, **not** a temp file. Running `stat` will leave that file behind
  after the command completes.
- `NoNet`, `NoIntr`, and `Gzip` are all true by default.
- Interval is the recorder's default (`1s`); `stat` does not auto-tune for
  short-lived workloads. Pass `-i` explicitly for sub-second resolution.

Orchestration (see [stat.go:130-175](../core/cmd/perfmonger/stat.go#L130-L175)):

1. Build a `RecorderOption` with `StopCh` set; attach the stop channel.
2. Launch `recorder.RunWithOption` in a goroutine.
3. `time.Sleep(100 * time.Millisecond)` to give the recorder a first sample.
4. `exec.Command(cmd[0], cmd[1:]...)` with inherited stdio; `Run()` it.
5. `close(stopCh)`, then `select` on `recorderDone` vs. `time.After(5s)` —
   the 5s is a worst-case bound; normally the recorder exits immediately.
6. Print `\n== Performance Summary ==\n\n` **to stderr**, then call
   `summarizer.RunWithOption(cmd.SummaryOpt)` (which writes the summary
   body to stdout). A non-zero exit from the user command is reported on
   stderr but does not prevent the summary from being printed.

### 5.5 `plot`

Args: required `LOG_FILE`.

Flags: `-o`/`--output-dir` (default `.`), `-T`/`--output-type` (`pdf`|`png`,
default `pdf`), `-p`/`--prefix`, `-s`/`--save` (keep `.gp` + `.dat`),
`--disk-only`, `--plot-read-only`/`--plot-write-only`/`--plot-read-write`,
`--plot-numkey-threshold` (hide legend if device count exceeds; default 10),
`--plot-iops-max` (0 = auto), `--with-gnuplot` (path, default `gnuplot`),
`--offset-time` (shift x-axis).

Produces: `disk-iops.{pdf|png}`, `disk-transfer.{pdf|png}`, `cpu.{pdf|png}`,
`allcpu.{pdf|png}`.

Requires `gnuplot` on `$PATH` with `pdfcairo` (for PDF) or ImageMagick
`convert` (for PNG). The command checks these at startup and errors out with
a helpful message if missing.

### 5.6 `summary`

Args: required `LOG_FILE`.

Flags: `-p`/`--pager <cmd>`, `--no-pager`, `--disk-only <regex>`, `--json`.

Pager selection logic:
- `--pager <cmd>` overrides everything when non-empty.
- Otherwise fall back to `$PAGER`.
- Pipe to the pager **only** when `--no-pager` is not set, a non-empty pager
  was resolved, and `stdout` is a TTY.
- If starting the pager fails (`exec.Command(...).Start()` errors), the
  command falls back to writing to `os.Stdout` rather than failing outright.

### 5.7 `fingerprint`

Args: optional `OUTPUT_TARBALL` (default `./fingerprint.<hostname>.tar.gz`).

Collects (see [fingerprint.go](../core/cmd/perfmonger/fingerprint.go)):

- `/proc/{cpuinfo,meminfo,mdstat,mounts,interrupts,diskstats,partitions,ioports,version,cmdline,filesystems,swaps}`
  and the contents of `/proc/sys/fs/`
- `lscpu`, CPU frequency scaling info, `free -h`, `numactl --hardware`
- `fdisk -l`, `lsblk -t`, `df -h`, `mount`
- `/sys/block/*` attributes (size, scheduler, rotational, hw_sector_size)
- `lspci -vvv`, `lsmod`
- `uname -a`, `lsb_release -a`, `/etc/{debian_version,redhat-release,os-release}`
- `sysctl -a`, `ip addr`, `ip route`, `netstat -i`

Each artifact is a file under `fingerprint.<hostname>/` inside the resulting
tarball. Commands whose binaries are missing are skipped rather than treated
as fatal.

### 5.8 `init-shell`

Emits a bash/zsh completion scriptlet. Called as `perfmonger init-shell -` to
write the completion script directly, or without `-` to print setup
instructions (e.g., how to `eval` it from `~/.bashrc`).

---

## 6. Background Recording

Go is not fork-safe — spawning a background process via `fork` would leave
the runtime (goroutines, GC) in an undefined state. PerfMonger uses a re-exec
pattern instead:

1. The parent detects `--background`, resolves the output path to absolute,
   and calls `launchDaemonChild()`
   ([record.go:323](../core/cmd/perfmonger/record.go#L323)).
2. The child is spawned via `exec.Command(selfBin, args...)` with:
   - `Env` extended with the sentinel `PERFMONGER_DAEMON_CHILD=1`.
   - `Dir = "/"`, `Stdin/Stdout/Stderr = nil` (Go closes them to `/dev/null`).
   - `SysProcAttr.Setsid = true` so the child becomes a session leader and
     survives the parent exiting.
3. The argument list is reconstructed field-by-field from `RecorderOpt` — it
   is *not* simply `os.Args` passed through. This means any flag the CLI
   normalizes (e.g., `-d` repetition, `--record-intr=false`) is also what the
   child sees.
4. The parent returns; the child detects `PERFMONGER_DAEMON_CHILD=1`, skips
   the re-exec branch, and calls `recorder.RunWithOption()`.
5. The recorder writes `<os.TempDir()>/perfmonger-<username>-session.pid`
   atomically under an `flock` held on `<os.TempDir()>/.perfmonger.lock`.
   `os.TempDir()` resolves to `/tmp` on Linux by default but honors
   `$TMPDIR` if set.

Signal handling: the child installs a `signal.Notify(sigint_ch, os.Interrupt)`
inside the recorder loop. `SIGINT` ends the loop and triggers cleanup.
`--kill` sends SIGINT from outside with exponential backoff (50ms, 100ms,
200ms, 400ms, 800ms) and then does a `syscall.Kill(pid, 0)` liveness probe
before returning success.

`--status` reads `/proc/<pid>/cmdline` and `/proc/<pid>` mtime to show what
the running process was invoked with and when it started.

Stale session cleanup: `getRunningSessionPID()` uses `syscall.Kill(pid, 0)`
to detect dead PIDs and removes the session file under the lock.
`RemoveSessionFile()` (on clean shutdown) also takes the lock, falling back
to an unlocked `os.Remove` if the lock file can't be opened.

TOCTOU caveat: the duplicate-session check in the CLI
(`getRunningSessionPID`) and the subsequent `WriteSessionFile` in the
recorder do not share a single locked critical section, so two parents
racing `--background` simultaneously can both pass the check and then both
write the PID file in sequence. In practice only one wins the final write,
but the loser still ends up running a recorder without a visible session
file.

---

## 7. Live Monitoring

`live` is a thin pipeline:

```
recorder goroutine (gob)
    │  (RecorderOpt.Output = "-", piped via player_stdin)
    ▼
child process: perfmonger play [--color] [--pretty]
    │  (line-delimited JSON on its stdout)
    ▼
goroutine in recorder copies player stdout → os.Stdout
```

The recorder's producer loop and the player-stdout-copier goroutine both run
in the parent process; the player itself is a real child process. On
shutdown, the recorder closes `player_stdin` and waits for the player to
exit.

The `viewer` package is **not** used by `live` today; live monitoring is
JSON-only. A TUI would have to be wired in separately.

---

## 8. Testing

Two test suites run from `make test`:

### 8.1 Go unit tests — `core/internal/perfmonger/*_test.go`

Covers stat-struct parsing, delta/usage math, and Linux `/proc` parsers.
Executed under `go test -v -cover`. Note that there are also `*_test.go`
files under `core/cmd/perfmonger/` (play, record, stat, summary) that are
**not** invoked by `make test`; to run them, `cd core/cmd/perfmonger && go
test`.

### 8.2 Integration tests — `tests/` (pytest + uv)

Dependencies are declared in `pyproject.toml` (`pytest`, `pytest-timeout`).
`uv sync` creates the virtualenv.

Shared helpers live in [tests/conftest.py](../tests/conftest.py):

- `perfmonger_bin` (pytest fixture) — resolves `lib/exec/perfmonger_linux_amd64`.
- `data_file` (pytest fixture) — returns a function that maps a fixture
  name to a full path into `spec/data/` (golden `.pgr` / `.pgr.gz`
  fixtures and expected outputs).
- `run_perfmonger(*args, timeout=120, cwd=None)` (module-level function,
  not a fixture) — wraps `subprocess.run`, returning a `CompletedProcess`;
  does not raise on non-zero exit.

Skip markers (also in `conftest.py`):

- `@requires_proc_diskstats` — skip on platforms without `/proc/diskstats`.
- `@requires_gnuplot` — skip if `gnuplot` is not on `$PATH`.

Test modules are organised per subcommand (`test_record.py`, `test_play.py`,
`test_summary.py`, `test_stat.py`, `test_live.py`, `test_background.py`,
`test_plot.py`, `test_fingerprint.py`, `test_cli.py`). Known-failing
behaviors are marked `@pytest.mark.xfail(reason=…)` so the suite stays green
while bugs are tracked in `doc/tickets/`.

Golden fixtures in `spec/data/`:

- `busy100.pgr`, `busy100.pgr.gz` — sample recordings.
- `busy100.pgr.summary`, `busy100.pgr.summary.json` — expected summary
  outputs (text + JSON).
- `busy100.pgr.played` — expected line-delimited JSON from `play`.
- `busy100.pgr.plot-formatted.disk.dat`,
  `busy100.pgr.plot-formatted.cpu.dat` — expected plot-formatter outputs.

---

## 9. Known Rough Edges

Things the code does today that are worth flagging for contributors. These
are *not* recommended behaviors — they are observations.

- **Summary JSON omits `mem`.** The text summary includes memory but the
  JSON form ([summarizer.go:180-207](../core/cmd/perfmonger-core/summarizer/summarizer.go#L180-L207))
  emits only `exectime` / `cpu` / `intr` / `disk` / `net`.
- **Player panics on I/O errors.** Gob decode errors other than `EOF` are
  thrown via `panic` rather than returning an error; same for `os.Open`.
- **`viewer` package is a placeholder.** Layout is hardcoded to "Hello
  world"; the package is retained in the tree but is not wired up to any
  `perfmonger` subcommand and has no real visualization.
- **Interval backoff is silent.** The recorder doubles the interval every
  1000 samples up to one hour unless `--no-interval-backoff` is passed.
  Long-running recordings therefore have non-uniform time granularity. There
  is no metadata in the `.pgr` stream indicating *when* a backoff happened.
- **`ProcStat` and `SoftIrqStat` are declared but not populated.** Both are
  pointer fields on `StatRecord` but no current Linux reader fills them in,
  so they are always nil in practice.
- **`--record-intr=false` is emitted to the daemon child in the default
  path.** `launchDaemonChild` appends `--record-intr=false` whenever
  `NoIntr` is true *or* `RecordIntr` is false. Under defaults both are
  true, so the flag is present. If the user actually passes `--record-intr`
  (enabling interrupt recording), neither condition holds and the flag is
  not emitted — so the child correctly does record interrupts.
- **No `.pgr` schema versioning.** `CommonHeader` has no version field. Any
  incompatible change to the gob-encoded types would silently break older
  recordings.
- **Pager fallback on start failure only.** The pager handshake in
  `summary.go` falls back to stdout if `cmd.Start()` fails, but a pager that
  crashes partway through the stream results in a broken-pipe panic.
- **`-v`/`--verbose` on `record`.** The flag is parsed into
  `recordCommand.Verbose` but the only thing that currently enables
  recorder-level verbose output is the `PERFMONGER_DEBUG` env var being
  non-empty. The flag is effectively a no-op unless you also set the env.

---

## 10. Data Flow Summary

```
                                ┌────────────────────────┐
                                │ perfmonger (cobra CLI) │
                                └─────────┬──────────────┘
                                          │
             ┌────────────────────────────┼─────────────────────────────┐
             ▼                            ▼                             ▼
     recorder.RunDirect           player.RunDirect           summarizer.RunDirect
     (reads /proc → gob)         (gob → JSON lines)         (gob → text/JSON summary)
             │                            ▲                             ▲
             ▼                            │                             │
        .pgr / .pgr.gz  ──────────────────┴─────────────────────────────┘
             │
             ▼
     plotformatter.RunDirect → .dat files → gnuplot → PDF/PNG
```

The gob-encoded `.pgr` stream is the pivot point: every downstream tool
consumes it. Anything that can produce a valid gob stream of
`CommonHeader`, `PlatformHeader`, `StatRecord…` works with the rest of the
toolchain — which is how `live` reuses the player over a pipe.
