---
title: getPartitions leaks a directory file descriptor on every call
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/internal/perfmonger/perfmonger_linux.go` around lines 65-88,
`getPartitions` opens `/sys/block/<name>` via `os.Open` into `dir` but never calls
`dir.Close()` and has no `defer` to do so.

`NewPlatformHeader` calls `getPartitions` once per block device found in
`/proc/diskstats`, so on a system with many block devices every startup
permanently leaks one OS file descriptor per device. Under repeated use (if
`NewPlatformHeader` is ever called more than once) or on a host with many
loop/device-mapper devices, this can exhaust the process's file-descriptor limit.

Why it is a bug: an opened directory handle is never released, leaking fds
proportional to the number of block devices.

Suggested direction: add `defer dir.Close()` immediately after the successful
`os.Open` in `getPartitions` (and ensure the error path does not leak either).

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: getPartitions opens /sys/block/<name> but never closes the directory handle. Fix: add `defer dir.Close()` right after the successful os.Open. Trivial.
