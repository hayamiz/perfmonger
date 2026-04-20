PerfMonger
==========

PerfMonger is a high-resolution system performance monitor for Linux. It
samples CPU, disk I/O, and memory metrics at sub-second intervals and emits
results in JSON, which makes later analysis with tools such as
[jq](https://github.com/stedolan/jq) straightforward.

Highlights:

* **High resolution** — sub-second sampling (default 1s, tunable down to
  fractions of a second).
* **Holistic** — CPU (including per-core), disk I/O, and memory in a single
  record.
* **Programmer friendly** — JSON output, one record per line.

PerfMonger is distributed as a single self-contained Go binary.

**Status:** PerfMonger is still maturing. Interfaces may change between
releases; avoid relying on it for production-critical workflows.

## Target platform

* Linux (x86_64). Darwin and other platforms are not supported.

## Installation

### Pre-built binary (coming soon)

Pre-built releases on GitHub will be provided once the release pipeline is in
place. In the meantime, please build from source as described below.

### Build from source

You need Go 1.24 or later (matches `core/go.mod`).

```sh
git clone https://github.com/hayamiz/perfmonger.git
cd perfmonger/core/cmd/perfmonger
go build -o ../../../lib/exec/perfmonger_linux_amd64 .
```

The resulting binary is `lib/exec/perfmonger_linux_amd64`. Copy it somewhere
on your `PATH` (e.g. `~/bin/perfmonger`) if you want to invoke it as
`perfmonger`.

### Optional dependencies

* `gnuplot` is required only for the `perfmonger plot` subcommand (graph
  rendering). All other subcommands work without it.

## Quick tour

All examples below assume the binary is on your `PATH` as `perfmonger`.

### See the available subcommands

```
$ perfmonger --help
PerfMonger is a system performance monitoring tool that enables sub-second
level monitoring of CPU, disk I/O, and network performance with JSON output.

Usage:
  perfmonger [flags]

Commands:
  completion  Generate the autocompletion script for the specified shell
  fingerprint Gather all possible system config information
  help        Help about any command
  init-shell  Initialize shell integration
  live        Monitor live system performance
  play        Play a recorded perfmonger session
  plot        Plot system performance graphs
  record      Record system performance information
  stat        Run a command and show performance summary
  summary     Summarize system performance data
```

Run `perfmonger <subcommand> --help` for the flags of any individual
subcommand. The short tour below introduces the five you are most likely to
reach for day to day.

### 1. `record` — capture a session to a log file

```
$ perfmonger record -i 0.5 --timeout 3 -d nvme0n1 -l /tmp/sample.pgr.gz
[recording to /tmp/sample.pgr.gz]
```

* `-i 0.5` samples every 0.5 seconds.
* `--timeout 3` stops recording after 3 seconds (omit to record until you hit
  Ctrl-C).
* `-d nvme0n1` narrows disk metrics to the named device; repeat the flag for
  multiple disks. Without it, PerfMonger records every block device it sees.
* `-l <path>` sets the output path; output is gzipped by default
  (`--no-gzip` disables that).

`record` also supports background sessions via `--background`, `--status`,
and `--kill` (these flags are specific to `record`):

```sh
perfmonger record --background --timeout 60
perfmonger record --status
perfmonger record --kill
```

### 2. `play` — replay a log file as JSON

```
$ perfmonger play /tmp/sample.pgr.gz | head -n 1
{"time":1776652123.249,"elapsed_time":0.500,"cpu":{"num_core":24,"all":{"usr":102.60,"nice":0.00,"sys":40.23,"idle":2257.17,...},"cores":[...]},"disk":{"devices":["nvme0n1",...],"nvme0n1":{"riops":18.00,"wiops":44.00,"rkbyteps":800.06,"wkbyteps":240.02,...},"total":{...}},"mem":{...}}
```

One JSON object is printed per sample, one per line, so it plays well with
`jq` and streaming tools. Pretty-printed (`--pretty`) and color (`--color`)
modes are available for interactive inspection, and `--disk-only <regex>`
filters the disk section to matching device names.

```sh
perfmonger play --pretty --disk-only '^nvme0n1$' /tmp/sample.pgr.gz
```

### 3. `stat` — run a command and print a summary

`stat` is the quickest way to profile a one-off command. It records for the
duration of the child process and prints a human-readable summary at the end.

```
$ perfmonger stat -d nvme0n1 -- sh -c 'find /workspaces/perfmonger -type f > /dev/null'
== Performance Summary ==

== performance summary of 'Command: sh -c find /workspaces/perfmonger -type f > /dev/null' ==

Duration: 0.289 sec

* Average CPU usage (MAX: 2400 %)
  * Non-idle usage: 90.30 %
       %usr: 45.15 %
       %sys: 45.15 %
       %irq: 0.00 %
      %soft: 0.00 %
     %other: 0.00 %
  * Idle usage: 2309.70 %
    %iowait: 0.00 %
      %idle: 2309.70 %

* Average DEVICE usage: nvme0n1
        read IOPS: 0.00
       write IOPS: 0.00
  read throughput: 0.00 MB/s
 ...
```

Use `--json` to emit the same summary as JSON for scripting.

### 4. `summary` — summarize an existing log file

If you already recorded a session with `record`, `summary` aggregates it into
the same report format as `stat`.

```
$ perfmonger summary --disk-only '^nvme0n1$' /tmp/sample.pgr.gz
== performance summary of '/tmp/sample.pgr.gz' ==

Duration: 3.000 sec

* Average CPU usage (MAX: 2400 %)
  * Non-idle usage: 67.16 %
       %usr: 38.76 %
       %sys: 28.40 %
       %irq: 0.00 %
      %soft: 0.00 %
     %other: 0.00 %
  * Idle usage: 2332.84 %
    %iowait: 0.33 %
      %idle: 2332.51 %

* Average DEVICE usage: nvme0n1
        read IOPS: 3.00
       write IOPS: 10.00
  read throughput: 0.13 MB/s
 write throughput: 0.54 MB/s
     read latency: 777.8 usec
    write latency: 1533.3 usec
      read amount: 0.39 MB
     write amount: 1.61 MB
```

`--json` is available here too, and `--pager <cmd>` pipes the report through
a pager of your choice.

### 5. `plot` — render graphs from a log file

`plot` turns a recorded log file into CPU and disk I/O graphs via
[gnuplot](http://www.gnuplot.info/). Make sure `gnuplot` is installed and on
your `PATH` first.

```sh
perfmonger plot -o /tmp/report -p session -T pdf /tmp/sample.pgr.gz
```

This produces PDF plots under `/tmp/report/session_*.pdf`. Use `-T png` for
PNG output, and `--save` to also keep the intermediate gnuplot scripts and
data files.

## Record JSON schema (short version)

Each record written by `record` / emitted by `play` has this shape (fields
abbreviated):

```
{
  "time": 1776652123.249,         // Unix epoch seconds
  "elapsed_time": 0.500,          // seconds since the session started
  "cpu": {
    "num_core": 24,
    "all": { "usr": ..., "sys": ..., "idle": ..., ... },
    "cores": [ { "usr": ..., "sys": ..., "idle": ..., ... }, ... ]
  },
  "disk": {
    "devices": ["nvme0n1", ...],
    "nvme0n1": { "riops": ..., "wiops": ..., "rkbyteps": ..., "wkbyteps": ..., ... },
    "total":   { "riops": ..., "wiops": ..., ... }
  },
  "mem": { "mem_total": ..., "mem_used": ..., "mem_free": ..., ... }
}
```

CPU percentages under `all` are scaled to `num_core * 100` (so a fully loaded
24-core box can reach `%usr` of up to 2400). Per-core values are on the
usual 0–100 scale.
