---
title: Integrate external GPU-usage tools (e.g. nvidia-smi) for GPU metric collection
type: feature
priority: medium
status: open
created: 2026-06-27
updated: 2026-06-27
---


## Description

PerfMonger currently collects CPU, disk I/O, and network metrics directly
from Linux virtual filesystems (`/proc`, `/sys`) inside
[perfmonger_linux.go](core/internal/perfmonger/perfmonger_linux.go). It has
no notion of GPU utilization. With GPUs now central to ML/HPC workloads,
users increasingly want sub-second GPU usage alongside the existing CPU/IO
metrics in the same recording, so that GPU stalls can be correlated with CPU
and I/O behavior on one timeline.

Unlike CPU/disk/net, there is no stable, vendor-neutral kernel interface for
GPU utilization. The de-facto source on NVIDIA systems is the `nvidia-smi`
tool (or the underlying NVML library); AMD has `rocm-smi`, Intel has
`intel_gpu_top`. This ticket is about **integrating with such external GPU
tools** as a metric source, rather than reimplementing vendor drivers.

The goal of this ticket is a **design recommendation**, not an
implementation. The hard parts to decide:

1. **Integration mechanism:** shell out to `nvidia-smi` per sample vs. run
   `nvidia-smi --loop`/`dmon` as a long-lived child and stream its output
   vs. bind to NVML directly via cgo. Each has different latency, overhead,
   and dependency trade-offs.
2. **Sampling-loop impact:** the recorder samples at intervals as low as
   `-i 0.1` (100 ms). Forking `nvidia-smi` per sample is far too slow and
   would dominate the interval. GPU collection must not block or jitter the
   existing CPU/IO sampling loop.
3. **Data model + output:** how a new `Gpu` section fits into the recorded
   `.pgr` stream and the JSON emitted by `play`/`summary`/`stat`, and how to
   stay backward-compatible with existing readers.

### Existing collection surfaces (for reference)

- The per-sample collection happens in the recorder's sampling loop; metric
  structs are gob-encoded into the `.pgr` stream — see
  [recorder.go](core/cmd/perfmonger-core/recorder/recorder.go).
- Platform metric readers (`ReadCpuStat`, `ReadDiskStats`, `ReadNetStat`,
  `GetInterruptUsage`, etc.) live in
  [perfmonger_linux.go](core/internal/perfmonger/perfmonger_linux.go) and
  are pure `/proc`+`/sys` file reads — no subprocess forks today.
- `StatRecord` / the header structs in
  [perfmonger.go](core/internal/perfmonger/perfmonger.go) define what each
  sample carries; `play`/`summary`/`plot`/`viewer` all decode these.

### Design axes to evaluate

#### A. Integration mechanism

1. **Fork `nvidia-smi` per sample.** Simplest to implement, but a fork+exec
   plus driver query costs tens to hundreds of ms — incompatible with
   `-i 0.1`. Reject for the hot path.
2. **Long-lived streaming child.** Spawn `nvidia-smi dmon` or
   `nvidia-smi --query-gpu=... --format=csv,noheader,nounits -l <interval>`
   once at recording start, read its stdout line-by-line in a dedicated
   goroutine, and pair the most-recent GPU reading with each PerfMonger
   sample. One fork for the whole run; low per-sample cost. Recommended
   starting point.
3. **NVML via cgo.** Lowest latency and richest data, but adds a cgo build
   dependency on the NVIDIA driver/headers, complicates cross-compilation
   and the single-static-binary story, and is vendor-locked. Defer unless
   (2) proves insufficient.

Recommendation: design around (2), with the GPU tool path/args configurable
so AMD (`rocm-smi`) and others can be slotted in later behind the same
abstraction.

#### B. Metrics to collect (NVIDIA first)

Per GPU, the high-value fields from
`nvidia-smi --query-gpu=...`:

- GPU index, name/UUID (header-time, not per-sample).
- Utilization: `utilization.gpu`, `utilization.memory` (%).
- Memory: `memory.used`, `memory.total` (MiB).
- Power: `power.draw`, `power.limit` (W).
- Temperature: `temperature.gpu` (°C).
- Clocks: `clocks.sm`, `clocks.mem` (optional).

Per-process GPU usage (`nvidia-smi pmon`) is richer but heavier and
higher-cardinality — defer to a follow-up / opt-in.

#### C. Decoupling GPU sampling from the main loop

- The streaming child emits readings on its own cadence; the recorder
  should keep the latest reading in a goroutine-safe holder and snapshot it
  when it writes each PerfMonger sample. Stale-reading handling (mark a
  reading as stale if older than N intervals) needs a policy.
- All GPU collection must be **non-fatal and optional**: if `nvidia-smi` is
  absent, not on `PATH`, or exits, recording continues with no GPU section
  rather than aborting. This mirrors how missing `/proc` sources should
  degrade gracefully.
- The child process must be reaped and its pipe fds closed on shutdown,
  including on SIGINT/SIGTERM — coordinate with the existing recorder
  signal/cleanup work (see related robustness tickets).

#### D. Output / format compatibility

- Add an optional `Gpu` section to the per-sample record (or a new optional
  record), analogous to the metadata-embedding discussion in
  [#0010](0010-record-embed-system-metadata.md). Older `play`/`summary`
  binaries must tolerate its presence/absence.
- `play` JSON gains a `gpu` key; `summary` reports averages/peaks per GPU;
  `live`/`viewer` and `plot` GPU support can be follow-up tickets.

### Questions to answer during triage

- Which mechanism (A.2 vs A.3) — confirm streaming-child as the v1 approach?
- Exact `nvidia-smi` invocation and field list; how to parse CSV robustly
  across driver versions (column order, `[N/A]` values, units).
- Enablement UX: auto-detect `nvidia-smi` and enable if present, vs. an
  explicit `--gpu` flag, vs. `--gpu-tool <path>`? Default behavior matters
  for users with no GPU.
- Stream-format/versioning impact and the backward-compat plan for readers.
- Multi-GPU and MIG handling — list all GPUs, fixed order by index?
- Testing strategy: the test host has no GPU, so integration tests need a
  mock `nvidia-smi` (a stub script on `PATH` emitting canned CSV) to
  validate parsing, pairing, and graceful-absence behavior.

### Deliverable

A written recommendation (added as `## Implementation Notes` during triage)
covering:

1. Chosen integration mechanism + dependency/build implications.
2. Final per-GPU field list and the Go struct sketch.
3. How GPU readings are decoupled from and paired with the main sampling
   loop, plus child-process lifecycle/cleanup.
4. `.pgr` format change + backward-compat plan and output changes for
   `play`/`summary`.
5. A pytest using a mock `nvidia-smi` stub that validates end-to-end
   collection, JSON output, and graceful behavior when the tool is absent.

No code changes in this ticket — design only. Implementation follows in a
separate ticket once the shape is agreed.

## Triage

- Complexity: high
- Mechanical fix: no
- Requires user decision: yes
- Notes: Design-phase ticket: choose integration mechanism (streaming child vs NVML cgo vs per-sample fork), the metric set, decoupling from the main loop, and .pgr format/back-compat before implementing.

## Implementation Notes

Mechanism (Axis A): A.1 per-sample fork — rejected (too slow for -i 0.1). A.2 long-lived streaming child (recommended) — spawn `nvidia-smi --query-gpu=... --format=csv,noheader,nounits -l <interval>` once; a reader goroutine buffers the latest reading under an RWMutex; the main loop snapshots it per StatRecord. One fork, negligible per-sample cost. A.3 NVML via cgo — deferred (build/cross-compile burden, vendor lock).

Metrics: header-time static per GPU (index, name, uuid); per-sample (utilization.gpu, utilization.memory, memory.used, memory.total, power.draw, power.limit, temperature.gpu); Tier 2 later (clocks, per-process via pmon).

Architecture: new recorder/gpu.go with a GpuCollector (Start/readLoop/GetLatest/Stop); main loop sets record.Gpu = collector.GetLatest() before encode; stale-reading policy (mark is_stale if older than 2× interval); clean child shutdown on SIGINT/SIGTERM. Continue recording if GPU data is missing (non-fatal).

Format: add Gpu []*GpuReading to StatRecord (gob zero-value both directions for back-compat); optional RecordingHasGPU header flag so old play can warn. play emits a gpu JSON array; summary GPU section deferred to a follow-up.

Enablement: auto-detect nvidia-smi on PATH + --no-gpu opt-out (recommended); optional --gpu-tool for AMD/Intel deferred to v2.

Questions for the user: confirm A.2; confirm metric list; auto-detect vs explicit --gpu; hardcode nvidia-smi vs --gpu-tool now; stale handling (is_stale flag vs interpolate vs exclude); GPU support for live/plot in scope; pytest mock-subprocess vs C integration test.
