---
title: plotformatter double-closes disk_dat and cpu_dat temp files (FD-reuse hazard)
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 2/3 skeptics.

In `core/cmd/perfmonger-core/plotformatter/plotformatter.go`, each `disk_dat` temp
file is explicitly closed at line 444 (in the post-loop cleanup loop,
`disk_dat.File.Close()`) and each `cpu_dat` temp file at line 473. However, defers
are also registered for every such file at lines 385 and 420 respectively. When
`runPlotFormat` returns normally, all of those defers fire and call `Close()` a
second time on already-closed `*os.File` handles.

While Go's `os.File.Close()` on an already-closed file returns an error rather than
crashing, that second error is silently discarded. More importantly, this is a
classic FD-reuse hazard: on Linux, if a new file was opened (and assigned the same
numeric fd) between the first and second close, the deferred close could
erroneously close the unrelated new descriptor.

Why it is a bug: redundant closes risk closing an unrelated, recycled file
descriptor.

Suggested direction: close each temp file exactly once — either remove the
explicit `Close()` calls and rely on the defers, or set the handle to a state that
makes the deferred close a no-op (e.g. close-once guard), so no descriptor is
closed twice.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: disk_dat/cpu_dat files are Closed explicitly in the normal path AND via deferred Close, a double-close / FD-reuse hazard. Fix: remove the explicit Close() calls and rely on the defers (recommended), or guard with a close-once flag.
