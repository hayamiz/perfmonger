---
title: Embed basic system component metadata in `record` log files
type: feature
priority: medium
status: open
created: 2026-04-20
updated: 2026-04-20
---

## Description

Today, detailed system-component information (CPU model, memory layout,
block-device topology, kernel/distro info, `lscpu`/`cpuinfo`/`lsblk`
dumps, etc.) is only captured by the separate `perfmonger fingerprint`
subcommand, which writes a directory of files *out-of-band* from any
recording. The recorded `.pgr` stream itself carries only a minimal
header — see
[perfmonger.go:37-55](core/internal/perfmonger/perfmonger.go#L37-L55):

```go
type CommonHeader struct {
    Platform  PlatformType
    Hostname  string
    StartTime time.Time
}

type LinuxHeader struct {
    Devices   map[string]LinuxDevice // name + partition list only
    DevsParts []string
}
```

This means a `.pgr` in isolation cannot answer questions like "what CPU
produced these numbers?" or "what model of SSD is `sda`?". Users who
share `.pgr` files, or who come back to old recordings months later,
have no way to recover that context unless they remembered to run
`fingerprint` alongside the recording and keep the two together.

We want **basic, high-value system identification info embedded directly
in the `.pgr` header**, so that `play`/`summary`/`stat` can display it,
and so that archived recordings are self-describing.

The goal of this ticket is a **design recommendation**, not an
implementation. Two things need to be decided:

1. **Storage:** extend the existing header structs vs. add a new
   dedicated metadata record vs. something else. Backward-compat with
   existing `.pgr` files matters.
2. **Content + collection cost:** exactly which pieces of
   system-component info are useful enough to include, *and* cheap
   enough to collect synchronously at recording start without
   perceptibly delaying the first sample.

### Existing metadata surfaces (for reference)

- Header written at recording start — see the `enc.Encode(cheader)` /
  `enc.Encode(platform_header)` calls in
  [recorder.go:267-381](core/cmd/perfmonger-core/recorder/recorder.go#L267-L381).
  Two gob-encoded structs before the first sample record.
- `fingerprint` subcommand — see
  [fingerprint.go](core/cmd/perfmonger/fingerprint.go). Runs
  `lscpu`, dumps `/proc/cpuinfo`, `/proc/meminfo`, `/proc/mdstat`,
  `/proc/mounts`, `/proc/interrupts`, plus block-device details.
  Writes to a directory on disk, **not** into the `.pgr`.
- Player / summary / stat already decode `CommonHeader` +
  `PlatformHeader`; any new fields must either be additive (gob is
  tolerant of missing fields in older readers only in specific ways)
  or carried as a new optional third record.

### Design axes to evaluate

#### A. Storage shape

1. **Extend `CommonHeader` / `LinuxHeader` in place.** Simplest, but
   gob encoding compatibility needs verification: new fields should
   decode as zero values in older players, but removed/renamed fields
   break things. Also mixes "structural" header info with rich
   metadata, which may bloat the hot path.
2. **Add a new optional `SystemMetadata` record** encoded right after
   the existing two headers (or gated behind a version bump).
   Decoders that don't know about it can skip it if framed with a
   length prefix or a type tag. Keeps the two concerns separate.
3. **Sidecar-in-stream as a keyed blob.** Opaque `map[string]string`
   (or JSON blob) inside the header; future-proof but loses typing.
4. **External sidecar file (`<name>.pgr.meta.json`).** Avoids touching
   the binary format, but breaks the "self-describing single file"
   goal and is easy to lose.

Recommend (1) or (2) with a stream-format version bump, and document
the bump policy.

#### B. Content: what to embed

Candidate fields, ranked by value-per-byte and collection cost. The
hard constraint is **zero user-visible startup delay** — collection
must be deterministic, non-blocking, and ideally sub-millisecond.

**Tier 1 — cheap, always include (file reads only, a few ms worst case):**

- Kernel version + build (`uname -r`, `/proc/version`).
- OS release (`/etc/os-release` → `PRETTY_NAME`, `ID`, `VERSION_ID`).
- CPU model string (first `model name` from `/proc/cpuinfo`).
- CPU logical/physical core count (derived from `/proc/cpuinfo`
  without shelling out to `lscpu`).
- Total memory (`MemTotal` from `/proc/meminfo`).
- Architecture (`runtime.GOARCH` or `uname -m`).
- perfmonger version (already known at compile time).
- Timezone / clocksource (`/sys/devices/system/clocksource/...`).
- Boot time (`/proc/stat` `btime`) — useful to compute uptime at
  recording start.

**Tier 2 — cheap if we already walk `/sys/block` for the recorder:**

- Per-recorded-block-device model + vendor + size + rotational flag
  (`/sys/block/<dev>/device/model`, `.../vendor`, `.../size`,
  `.../queue/rotational`). We already enumerate block devices to
  populate `LinuxHeader.Devices`; reading one extra sysfs file per
  device is negligible.
- Per-recorded-block-device scheduler (`/sys/block/<dev>/queue/scheduler`).
- NIC model / driver for recorded interfaces
  (`/sys/class/net/<iface>/device/{vendor,device}` +
  `/sys/class/net/<iface>/device/driver`). Cheap sysfs reads.

**Tier 3 — defer or make opt-in (requires subprocess / non-trivial parsing):**

- `lscpu` / `dmidecode` output — forks a subprocess, can be slow,
  `dmidecode` needs root. **Exclude from default path.**
- Full `/proc/cpuinfo` contents — verbose; superseded by Tier 1.
- Container / virtualization detection (`systemd-detect-virt`,
  `/proc/1/cgroup` parsing) — forks or heuristic; opt-in.
- NUMA topology — parse `/sys/devices/system/node/`; useful but
  non-trivial. Defer unless a concrete use case shows up.

**Tier 4 — explicitly out of scope for the inline header:**

- Anything `fingerprint` currently dumps as raw files (full
  `/proc/interrupts`, `/proc/mdstat`, `/proc/mounts`, etc.). Too
  verbose for the hot path; keep in `fingerprint` for deep dives.

Recommendation: embed **Tier 1 + Tier 2** by default. Add a
`--no-sysinfo` flag for users who want the old minimal header (e.g.
to keep `.pgr` files anonymized). Tier 3 stays behind
`fingerprint` / explicit flags.

#### C. Collection-time cost budget

Target: **< 5 ms total** on a typical Linux host for Tier 1 + Tier 2,
because `record -i 0.1` users expect the first sample ~100 ms after
invocation.

- All Tier 1/2 sources are file reads under `/proc` and `/sys`, which
  are backed by kernel virtual filesystems — no disk I/O, no network,
  no subprocess fork. Expected cost: tens of microseconds per file.
- Do **not** run any external command in the default path (no
  `lscpu`, `uname` subprocess, etc. — use syscalls / file reads).
- Collection runs **before** the sampling loop starts, so any delay
  is visible only as "startup latency," not as sampling jitter.
- Failures must be non-fatal: if `/sys/block/sda/device/model` is
  missing, emit an empty string and continue. Never abort recording
  for a metadata read error.

### Questions to answer during triage

- Stream format: add a version field to `CommonHeader`, or rely on
  adding a third header record? What breaks for old `play`/`summary`
  binaries reading a new `.pgr`, and vice versa?
- Where should the new Go type live — `core/internal/perfmonger/` next
  to `CommonHeader`, or a new `sysinfo` package?
- Should `play --show-header` (or a new flag) surface this metadata
  to human users, and how should `summary` / `stat` display it?
- Is there demand to make Tier 2 opt-out per-device (privacy — e.g.
  don't leak a disk serial if we ever include serials)? Current
  recommendation includes *model only*, not serials.
- How do we test this? Suggest a pytest case that records briefly on
  the test host, re-reads the `.pgr`, and asserts the embedded
  metadata is present and non-empty for known-good fields (CPU model,
  kernel version, hostname) while being tolerant of host-specific
  values.

### Deliverable

A written recommendation (added as `## Implementation Notes` during
triage) covering:

1. Chosen storage shape (A.1 vs. A.2) + migration / compat plan.
2. Final field list (confirm Tier 1 + Tier 2, or adjust).
3. Go struct definition sketch.
4. Collection code location and error-handling policy.
5. A minimal pytest that validates end-to-end embedding + readback.

No code changes in this ticket — design only. Implementation follows
in a separate ticket once the shape is agreed.
