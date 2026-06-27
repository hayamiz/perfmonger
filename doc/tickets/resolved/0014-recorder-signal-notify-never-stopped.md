---
title: signal.Notify in recorder.RunDirect is never paired with signal.Stop
type: bug
priority: medium
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/recorder/recorder.go` at line 408,
`signal.Notify(sigint_ch, os.Interrupt)` registers a channel to receive SIGINT
notifications, but `signal.Stop(sigint_ch)` is never called before `RunDirect`
returns.

The Go runtime continues to hold a reference to `sigint_ch` and will keep
delivering signals to it after the function exits. This matters because
`RunDirect`/`RunWithOption` is a library API that can be invoked more than once in
the same process lifetime — for example from tests, from the `stat` subcommand
(`core/cmd/perfmonger/stat.go` line 141 calls it inside a goroutine), or from
future embedding callers. Each invocation registers a fresh channel and the old
channels are never drained or deregistered, so:

- The old `sigint_ch` (a buffered channel of size 1) is kept alive by the signal
  package and never garbage-collected — a resource leak.
- Signals delivered to a now-unread old channel are silently consumed instead of
  propagated, leading to missed-signal behavior (e.g. a later SIGINT no longer
  triggers graceful shutdown).

Why it is a bug: `signal.Notify` without a matching `signal.Stop` is a documented
leak/misbehavior pattern; the dangling registration changes signal-delivery
semantics for the rest of the process.

Suggested direction: call `defer signal.Stop(sigint_ch)` immediately after the
`signal.Notify` call so the registration is torn down when `RunDirect` returns.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: Add `defer signal.Stop(sigint_ch)` immediately after the signal.Notify call. Standard Go cleanup pattern, single defer in one function.

## Resolution

Fixed via strict TDD.

Files changed:

- `core/cmd/perfmonger-core/recorder/recorder.go`
  - Added package-level overridable seams `signalNotify = signal.Notify` and
    `signalStop = signal.Stop` so signal registration/teardown is observable in
    tests.
  - In `RunDirect`, changed the registration call to `signalNotify(sigint_ch,
    os.Interrupt)` and added `defer signalStop(sigint_ch)` immediately after it,
    so the SIGINT handler is deregistered when `RunDirect` returns. This stops
    the channel from being leaked and from silently consuming signals after the
    function exits.
- `core/cmd/perfmonger-core/recorder/recorder_test.go`
  - Added `TestRunDirectStopsSignalNotify`, a regression test that installs the
    seams, runs `RunDirect` with a short timeout to a temp output file, and
    asserts that the exact channel passed to `signal.Notify` is also passed to
    `signal.Stop` before `RunDirect` returns.

TDD: the test was written first and initially failed to compile (`undefined:
signalNotify` / `undefined: signalStop`), confirming RED. After adding the
seams and the `defer signalStop`, the test passed (GREEN).

Verification (all passed, run from repo root after sourcing
`00_LOAD_GO_DEVENV.sh`):

- `cd core/internal/perfmonger && go test -cover ./...` — ok, coverage 54.4%
- `cd core && go test ./cmd/...` — all packages ok
- `cd core/internal/perfmonger && go vet perfmonger_linux.go $(ls *.go | grep -v perfmonger_)` — exit 0
- `cd core/cmd/perfmonger && go build -o ../../../lib/exec/perfmonger_linux_amd64 .` — build ok
