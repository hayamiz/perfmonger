---
title: ReadCpuStat indexes CoreStats with unvalidated CPU number from /proc/stat (out-of-bounds panic)
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/internal/perfmonger/perfmonger_linux.go` around lines 168-173,
`ReadCpuStat` sizes `record.Cpu.CoreStats` from the core count obtained at startup
(`nproc --all`, or `runtime.NumCPU()` as a fallback). It then parses the CPU number
from each `cpu<N>` line of `/proc/stat` into `n_core` (line 168) and directly
indexes the slice at line 173 (`&record.Cpu.CoreStats[n_core]`) with no bounds
check.

If `/proc/stat` reports a CPU number `>= len(CoreStats)`, the access panics with
"index out of range". This can happen when:

- a CPU is hot-plugged online between when `nproc` ran and when `/proc/stat` is
  read, or
- in containerized environments where `nproc` sees only the cgroup quota while
  `/proc/stat` exposes all host CPUs.

A panic in the sampling loop bypasses `out.Flush()` and `gzwriter.Close()`, leaving
a corrupted gzip output file.

Why it is a bug: untrusted-length kernel input is used as a slice index, crashing
the recorder and corrupting output on realistic configurations.

Suggested direction: bounds-check `n_core` against `len(record.Cpu.CoreStats)` and
either grow the slice to accommodate new cores or skip/ignore out-of-range CPU
lines, rather than indexing unconditionally.

## Triage

- Complexity: high
- Mechanical fix: no
- Requires user decision: yes
- Notes: ReadCpuStat indexes CoreStats with an unvalidated CPU number from /proc/stat. Fix needs a behavior decision: grow the slice (capture hot-plugged CPUs), skip/warn, or error out — meaningfully different semantics.

## Implementation Notes

record.Cpu.CoreStats is sized num_core but n_core is parsed from /proc/stat without bounds checking; an out-of-range index panics.

Options:
- Option A: grow the slice dynamically (append/resize, update NumCore). Captures hot-plug; may change NumCore mid-stream and over-allocate.
- Option B: guard `if n_core < len(CoreStats)` and skip out-of-bounds cores. Safe, minimal; loses hot-plug data silently.
- Option C: guard and return an error on mismatch. Fails fast; breaks recording on first hot-plug.

Open questions: should the recorder tolerate dynamic CPU-count changes or treat them as fatal; if growing, how to compute deltas for new entries; backward-compat for consumers relying on NumCore.
