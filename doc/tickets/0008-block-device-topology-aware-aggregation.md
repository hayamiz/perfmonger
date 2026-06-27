---
title: Record block device topology and use it for accurate system-wide I/O aggregation
type: feature
priority: medium
status: open
created: 2026-04-20
updated: 2026-06-27
---

## Description

Today `perfmonger` records every entry in `/proc/diskstats` verbatim and, when
summarizing, reports per-entry statistics plus a naive sum across all entries.
This conflates stacked devices with their backing devices: for example, an
`md0` RAID built on `nvme1n1` and `nvme2n1` appears as three independent
entries, and the reported system-wide I/O count double-counts the same
physical requests (once at the `md` layer, again at each backing device).
The tool therefore cannot give an accurate picture of how many real I/Os the
system is issuing.

We want to preserve per-entry recording (the raw `/proc/diskstats` data is
still useful), but additionally capture the **containment relationships
between block devices** — at minimum:

- MD RAID (`/proc/mdstat`, `/sys/block/mdX/md/dev-*`)
- device-mapper / LVM (`/sys/block/dm-*/slaves/`)
- partition → whole-disk (`/sys/block/<dev>/<part>`)

and possibly bcache, loop, dm-crypt, multipath as follow-ups.

The aggregator should then use this topology to compute system-wide I/O
counts without double-counting (e.g., by selecting a canonical layer —
physical leaves, or the topmost layer the user exposed via `-d`, depending
on what the user asks for).

### Out of scope (for this ticket)

- Dynamically detecting topology changes mid-recording.
- Filesystem-level aggregation (btrfs pools, ZFS).
- Non-Linux platforms (Linux-only today).

---

## Current implementation (investigated 2026-04-20)

### .pgr file format

Gob stream, optionally wrapped in gzip. Reader auto-detects gzip via magic
bytes `0x1f 0x8b` ([core/cmd/perfmonger-core/player/utils.go:13](../../core/cmd/perfmonger-core/player/utils.go#L13)).
Write order at
[recorder.go:373-381](../../core/cmd/perfmonger-core/recorder/recorder.go#L373-L381):

1. `CommonHeader` (gob-encoded)
2. `PlatformHeader` (gob-encoded — alias for `LinuxHeader`)
3. Stream of `StatRecord` (one per sample)

Read order at
[player.go:227-250](../../core/cmd/perfmonger-core/player/player.go#L227-L250)
mirrors it. The format is already a **"header-then-stream"** layout — there
is no dedicated "magic number / schema version" marker, but the two header
structs reliably decode first.

### Existing header structs

[core/internal/perfmonger/perfmonger.go:37-55](../../core/internal/perfmonger/perfmonger.go#L37-L55):

```go
type CommonHeader struct {
    Platform  PlatformType  // = 1 for Linux
    Hostname  string
    StartTime time.Time
}

type LinuxDevice struct {
    Name  string
    Parts []string      // partition names (leaf basenames, no parent ref)
}

type LinuxHeader struct {
    Devices   map[string]LinuxDevice  // whole-disk devices only (those with /sys/block/<name>)
    DevsParts []string                // every line in /proc/diskstats (devs + parts mixed)
}
```

So `LinuxHeader` **already** captures the partition → whole-disk
relationship (implicitly, via `LinuxDevice.Parts`), but **does not** capture
MD / DM / LVM / bcache / loop / multipath containment. It also does not
record device kinds (nvme vs. virtual vs. md vs. dm), sizes, or whether a
name is a logical or physical entity.

### Topology sources not yet read

`NewPlatformHeader` ([perfmonger_linux.go:19-54](../../core/internal/perfmonger/perfmonger_linux.go#L19-L54))
only walks `/proc/diskstats` and `/sys/block/<name>/<part>/stat`. It does
**not** read:

- `/sys/block/<name>/slaves/` — lower devices of dm/md
- `/sys/block/<name>/holders/` — upper devices
- `/sys/block/<name>/md/level` and `/sys/block/<name>/md/dev-*` — RAID level + members
- `/sys/block/<name>/dm/name`, `/sys/block/<name>/dm/uuid` — dm friendly names
- `/proc/mdstat`
- `/sys/block/<name>/device/` (presence implies physical device)
- `/sys/block/<name>/queue/rotational`, `.../size`

### Naive aggregation (the bug)

`GetDiskUsage1` at
[core/internal/perfmonger/usage.go:360-456](../../core/internal/perfmonger/usage.go#L360-L456).
Lines 428-438 sum every `d1.Entries` item into `total` unconditionally —
partitions are summed alongside their whole-disk, and stacked devices are
summed alongside their backing devices. Summarizer prints this `"total"`
line only when there are ≥2 devices
([summarizer.go:257-258](../../core/cmd/perfmonger-core/summarizer/summarizer.go#L257-L258)).

The player/live views do not compute a cross-device total — they emit
per-device JSON — so the fix is localized to the summarizer aggregation
path and the (future) plotformatter equivalents. Double-counting shows up
in the summary output, not in the raw JSON stream.

---

## Proposed design

### 1. Extend `LinuxHeader` with topology

Add fields to the **existing** `LinuxHeader` struct. Gob tolerates
field-addition on the decode side (unknown extra fields are ignored, missing
fields decode as zero values), so this is wire-compatible with older
recorders but new recordings gain the topology.

```go
// perfmonger.go

type DeviceKind string

const (
    KindPhysical  DeviceKind = "physical"  // nvme0n1, sda, vda — has /sys/block/<n>/device
    KindPartition DeviceKind = "partition" // nvme0n1p1, sda1
    KindMDRaid    DeviceKind = "md-raid"   // md0, md127
    KindDM        DeviceKind = "dm"        // dm-0 (generic dm; subtype in DMSubkind)
    KindLoop      DeviceKind = "loop"      // loop0
    KindBcache    DeviceKind = "bcache"    // bcache0
    KindUnknown   DeviceKind = "unknown"
)

type LinuxDevice struct {
    Name  string
    Parts []string  // existing — kept for backwards compat

    // New fields (zero values on pre-topology recordings):
    Kind     DeviceKind // physical / partition / md-raid / dm / loop / bcache / unknown
    Parent   string     // for KindPartition: the whole-disk name. "" otherwise.
    Slaves   []string   // lower devices (from /sys/block/<n>/slaves/). nil for physical/partition.
    Holders  []string   // upper devices (from /sys/block/<n>/holders/). For cross-check / debugging.
    MDLevel  string     // "raid0", "raid1", "raid5", ... — only when Kind == md-raid
    DMName   string     // dm friendly name (e.g. "vg0-root") — only when Kind == dm
    SizeSect int64      // /sys/block/<n>/size (in 512-byte sectors). 0 if unreadable.
}

type LinuxHeader struct {
    Devices      map[string]LinuxDevice
    DevsParts    []string

    // New:
    TopologyVer  int                       // 0 = pre-topology (old file), 1 = this design
    // Devices map now also contains partitions, dm-*, md*, loop*, etc.
    // DevsParts remains the authoritative list of "things found in /proc/diskstats".
}
```

**Decision needed:** whether to promote partitions into `Devices` map
(currently only whole-disks are keyed there). Recommendation: **yes** —
having every entry keyed by name simplifies topology lookup. The existing
`LinuxDevice.Parts` slice becomes redundant but is kept to avoid breaking
any external consumer; new code should traverse via `Parent` / `Slaves`.

**Detection rules** (new helper `buildTopology()` in `perfmonger_linux.go`):

| Signal | Kind |
|---|---|
| `/sys/block/<n>/md/level` exists | md-raid, read level |
| `/sys/block/<n>/dm/uuid` exists | dm, read `dm/name` |
| name matches `^loop[0-9]+` or `/sys/block/<n>/loop/backing_file` | loop |
| `/sys/block/<n>/bcache/` exists | bcache |
| `/sys/block/<n>/device/` exists AND no md/dm/loop | physical |
| parent dir of `<n>/stat` is a whole-disk sysfs dir | partition |
| else | unknown |

`Slaves` is read from `/sys/block/<n>/slaves/` for every non-partition kind
(empty directory for physical devices, which is fine).

### 2. Topology-aware aggregation

Add a helper to `usage.go`:

```go
// Returns the set of device names that form a non-overlapping cover of
// actual physical I/O for this recording, given the recorded topology.
// Default policy: "leaves" — devices with no Slaves in the recorded set.
// This naturally picks physical devices when md/dm layers are present,
// and picks the physical devices themselves when there are no stacked
// layers. Partitions of a whole-disk that is itself a leaf are EXCLUDED
// (partition I/O is already accounted for in the whole-disk counters).
func (h *LinuxHeader) CanonicalLeaves() []string
```

Aggregation policies to expose via a new `--aggregate` flag on
`perfmonger summary` (and the corresponding `SummaryOption` field):

- `leaves` *(default when TopologyVer ≥ 1)* — use `CanonicalLeaves()`.
  Correct for md-on-nvme, lvm-on-nvme, plain-disk setups. Partitions are
  excluded because kernel counts partition I/O under the parent whole-disk
  too.
- `physical` — only `Kind == KindPhysical`. Same as `leaves` in the common
  case; differs only if the user has a bcache/dm layer exposed without a
  physical leaf (rare).
- `top` — devices that have no entry in `Holders`. Useful when the user
  wants "what the application issued" rather than "what hit the hardware".
- `all` — preserve current (double-counting) behaviour for parity checks.
  Emits a warning.
- `selected` — only the devices passed via `-d` at record time
  (`TargetDisks`). Falls back to `leaves` if none.

When `TopologyVer == 0` (old recording), **force `all` + warn** so users
know the total may double-count. The warning should be printed to stderr
in the summary's first line, e.g.:
`warning: recording has no device topology; "total" line may double-count stacked devices`.

### 3. Fix `GetDiskUsage1`

Add a new entrypoint that takes a device-set and only sums those into
`total`; keep `GetDiskUsage1` as-is for the per-entry output (all entries
still appear in `(*usage)[name]`, only the `"total"` row changes):

```go
func GetDiskUsage1WithPolicy(
    t1 time.Time, d1 *DiskStat, t2 time.Time, d2 *DiskStat,
    filter *regexp.Regexp,
    totalSet map[string]struct{},  // names eligible for the "total" aggregation
) (*DiskUsage, error)
```

Summarizer chooses `totalSet` from the policy + header.

### 4. Player / live / plot

Out of scope for the first cut: they emit per-device data and don't compute
cross-device totals. If we later add a "total" row to those views, they
must reuse the same policy code. A follow-up ticket should track this.

### 5. Backwards compatibility

- **Old .pgr file → new binary:** `LinuxHeader.TopologyVer` decodes to 0.
  Summarizer detects this and falls back to `all` with a stderr warning.
  Per-entry output is unchanged.
- **New .pgr file → old binary:** Extra `LinuxDevice` fields are ignored by
  gob (gob's schema negotiation handles unknown fields). Old binaries
  continue to see only `Name` and `Parts`. No breakage.
- **Schema version bump:** `TopologyVer` starts at 1. Future additions
  (bcache caching tier, multipath) bump it. Summarizer compares to
  constants it knows about.

### 6. Tests

- Unit test for `buildTopology()` using a fake `/sys/block` tree under a
  `tmpfs` test root — the existing code uses hardcoded `/sys/block/...`
  paths, so the helper needs a path-prefix parameter (injected via
  function var, similar to existing patterns in `perfmonger_linux.go`).
- Unit test for `CanonicalLeaves()` over synthetic `LinuxHeader` fixtures:
  - md on two nvme + partitions → leaves = `{nvme0n1, nvme1n1}`
  - lvm (dm) on md on nvme → leaves = `{nvme0n1, nvme1n1}`
  - plain disk with partitions → leaves = `{sda}`
  - no topology (`TopologyVer == 0`) → nil (caller falls back)
- Integration test (pytest) that records, then runs `perfmonger summary`
  and asserts the "total" line is absent or warned on an old fixture, and
  reflects only leaves on a new recording.
- Golden `.pgr` fixture with known topology stored under `spec/data/` for
  deterministic summary-output tests.

### 7. Work breakdown (rough)

1. Add `DeviceKind`, extend `LinuxDevice`, bump `LinuxHeader.TopologyVer`.
2. Implement `buildTopology()` reading sysfs + mdstat; wire into
   `NewPlatformHeader()`.
3. Implement `CanonicalLeaves()` and policy resolution.
4. Add `GetDiskUsage1WithPolicy` and update summarizer.
5. Add `--aggregate` CLI flag to `perfmonger summary`.
6. Add tests + golden fixture.
7. Update README section on summary output + document the policy.

### Open questions

- **Partition handling when a whole-disk is a leaf.** The current
  `/proc/diskstats` already double-counts at this layer (the kernel
  aggregates partition I/O into the whole-disk counters). The design above
  excludes partitions from `leaves`, which is correct. Worth calling out
  in docs — some users may be surprised partitions don't contribute to
  the total.
- **Multipath.** `/sys/block/dm-*/dm/uuid` starts with `mpath-` for
  multipath. For v1 we classify these as `KindDM` and rely on `Slaves` to
  describe the topology; a dedicated `KindMultipath` can come later.
- **MD recovery / reshape.** Topology is snapshotted at record start. If
  the user starts a recording during a RAID rebuild and the array layout
  changes mid-recording, the header will be stale. Documented as
  out-of-scope above.
- **Should `--aggregate` be persisted into the .pgr?** No — it's a
  playback-time question. Only the raw topology goes into the file.

## Triage

- Complexity: high
- Mechanical fix: no
- Requires user decision: yes
- Notes: Requires designing topology-aware device detection across sysfs (MD RAID, device-mapper, partitions) and refactoring GetDiskUsage1 to use a pluggable aggregation policy. The ticket provides an extensive design that needs validation (CanonicalLeaves algorithm, partition edge cases) and integration testing.

## Implementation Notes

The ticket already contains a comprehensive design; the work is to validate/refine it.

- Data model (perfmonger.go): extend LinuxDevice with Kind/Parent/Slaves/Holders/MDLevel/DMName/SizeSect; add TopologyVer to LinuxHeader for forward-compat (gob tolerates additive fields). Decision: promote partitions into the Devices map or keep separate.
- Topology discovery (perfmonger_linux.go): new buildTopology() detecting device kind via sysfs (md/level → MD RAID, dm/uuid → DM, loop/backing_file or loopN → loop, bcache/ → bcache, device/ → physical), reading slaves/; inject a configurable sysfs root for testability; handle missing files gracefully.
- Aggregation (usage.go): add CanonicalLeaves(header) computing a non-overlapping device set (leaves with no slaves, excluding partitions whose parent is a leaf); returns nil when TopologyVer==0. Add GetDiskUsage1WithPolicy(totalSet) to filter which devices feed `total`; keep GetDiskUsage1 unchanged for per-entry output.
- Summarizer: add --aggregate flag (leaves default for TopologyVer>=1, plus physical/top/all/selected); when TopologyVer==0 force `all` and warn on stderr that total may double-count.
- Backward-compat: old .pgr decodes TopologyVer=0 (force all + warn); new .pgr → old binary ignores unknown fields.
- Testing: unit tests for buildTopology over a synthetic /sys/block tree (MD/LVM/partitions/bcache), CanonicalLeaves over synthetic headers, GetDiskUsage1WithPolicy; pytest integration with a golden .pgr in spec/data/.
- Open questions for the user: document partition exclusion from total; measure sysfs-read cost on large systems; treat multipath as generic DM or add a dedicated kind (design says defer).
