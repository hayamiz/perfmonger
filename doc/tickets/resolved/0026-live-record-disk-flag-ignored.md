---
title: --disk flag silently ignored in live/record direct path (TargetDisks map never populated)
type: bug
priority: high
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/live.go` around lines 92-93, `applyRubySpecificLogic()`
copies `DevsParts` into the `Disks` string field but never builds the
`TargetDisks` map that `RunDirect` actually uses for filtering. In `RunDirect`,
`ss.ReadDiskStats` is called with `option.TargetDisks`, which is `nil` on this
path. `ReadDiskStats` treats a nil `targets` pointer as "record all devices", so
specifying `-d sda` is silently ignored and every disk device is monitored
instead.

The same defect exists in `core/cmd/perfmonger/record.go` around lines 302-303.

The `TargetDisks` map is only constructed in `parseArgs`
(`core/cmd/perfmonger-core/recorder/recorder.go` lines 110-118), which is bypassed
by the direct-API path used by `live` and `record`.

Why it is a bug: a documented filtering flag (`-d`/`--disk`) has no effect,
producing far more output than the user requested.

Suggested direction: populate `option.TargetDisks` from the parsed device list in
the direct-API path (factor out the map-building logic from `parseArgs` so both
paths share it), and add a test asserting that `-d sda` restricts the recorded
devices.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: --disk is ignored because the direct-API path never populates option.TargetDisks. Fix: extract the map-building logic from parseArgs into a helper and call it from the live/record direct paths. Direct, no ambiguity.

## Resolution

Extracted the TargetDisks map-building logic out of `parseArgs` into a reusable
public helper `BuildTargetDisks(disks string) *map[string]bool` in
`core/cmd/perfmonger-core/recorder/recorder.go`. It returns `nil` for an empty
list (meaning "record all devices") and a populated map otherwise.

- `parseArgs` now calls the helper instead of inlining the map construction.
- The direct-API paths now call the helper after building the comma-separated
  `Disks` string, so `-d`/`--disk` is honored:
  - `core/cmd/perfmonger/live.go` (`applyRubySpecificLogic`)
  - `core/cmd/perfmonger/record.go` (`applyRubySpecificLogic`)

Test added: `TestBuildTargetDisks` in
`core/cmd/perfmonger-core/recorder/recorder_test.go` asserts the empty list
yields `nil` and that comma-separated lists populate the map correctly. It
failed RED with `undefined: BuildTargetDisks` before the helper existed.

Verification: `go test ./cmd/...`, `go vet`, and the binary build all pass.
