---
title: plotformatter double-closes disk_dat and cpu_dat temp files (FD-reuse hazard)
type: bug
priority: medium
status: resolved
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

## Resolution

Applied the recommended fix: removed the explicit `Close()` calls from the
post-loop cleanup loops in `runPlotFormat`
(`core/cmd/perfmonger-core/plotformatter/plotformatter.go`). Each `disk_dat` and
`cpu_dat` temp file is now closed exactly once, by the deferred close registered
at creation time. The cleanup loops still call `Writer.Flush()` so buffered data
reaches disk before the subsequent `ioutil.ReadFile`; the deferred close (which
fires at function return) only releases the fd, which does not affect the read.

A package-level `closeTmpFile` seam was introduced so the close path is
observable from tests.

Test added: `core/cmd/perfmonger-core/plotformatter/plotformatter_test.go`
(`TestRunPlotFormatClosesTmpFilesOnce`) drives `runPlotFormat` over the
`spec/data/busy100.pgr` fixture and asserts each temp file is closed exactly
once. Before the fix it reported every temp file closed 2 times (RED); after the
fix it passes (GREEN).

Verification: `go test -cover ./...` in `core/internal/perfmonger` passes; the
`core/cmd/...` packages pass (the `./cmd/...` glob fails only due to a
pre-existing untracked `recorder/godevenv/` GOPATH artifact, unrelated to this
change); `go vet` clean; binary rebuilds successfully.
