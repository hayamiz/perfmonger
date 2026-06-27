---
title: runInitShell() is dead code with an os.Exit error contract instead of returning an error
type: enhancement
priority: low
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/initshell.go` around lines 133-163, `runInitShell()` is
defined but never called: `newInitShellCommand()` wires up `opts.run()`
exclusively, and there is no other call site in the codebase.

The dead function uses `os.Exit(1)` on an unsupported shell, whereas the active
code path (`opts.run()`) returns an error that cobra converts to a non-zero exit.
If anyone were to call `runInitShell` in the future (e.g. from a test or a new
command path), they would get a hard `os.Exit` that bypasses `defer` statements,
cleanup handlers, and cobra's error printing — a footgun with an inconsistent error
contract.

Why it is a hardening issue rather than a live defect: the function is currently
unreachable, so it causes no runtime misbehavior today, but it is a latent hazard
and dead weight.

Suggested direction: remove `runInitShell()` (preferred), or reconcile it with the
active path by having it return an `error` instead of calling `os.Exit`, and add a
call site if it is intended to be used.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: runInitShell() is dead code (never wired; only opts.run() is used) and uses os.Exit(1) instead of returning an error, bypassing defers if ever called. Fix: remove the function.

## Resolution

Removed the dead `runInitShell()` function from
`core/cmd/perfmonger/initshell.go` (the former lines 162-191). It was never
wired up — `newInitShellCommand()` exclusively uses `opts.run()` — and it used
`os.Exit(1)` on an unsupported shell, bypassing defers and cobra's error
handling.

Safety net (TDD): before removing, confirmed the existing
`TestNormalizeShellName` and `TestRunDetectionFailureMessage` tests
(`core/cmd/perfmonger/initshell_test.go`) pass green, since they cover the
surviving `normalizeShellName`/`errorForShell` helpers that the live
`opts.run()` path relies on. Strengthened the net by adding
`TestErrorForShellUnsupported`, which asserts the non-empty unsupported-shell
message used by `run()`'s default branch, and watched it pass.

After removal: `grep -rn "runInitShell" --include="*.go"` showed no code
references (only a comment in the new test referencing the ticket history).
`errorForShell` and `normalizeShellName` remain in use by `run()` /
`getParentShell()`, so nothing became unused. `os` is still used by
`getParentShell()` (`os.Getppid`, `os.Getenv`), so no import cleanup was needed.

Verification:
- `cd core && go test -count=1 ./cmd/perfmonger/` — PASS
- `cd core/cmd/perfmonger && go build -o ../../../lib/exec/perfmonger_linux_amd64 .` — OK
