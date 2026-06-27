---
title: runInitShell() is dead code with an os.Exit error contract instead of returning an error
type: enhancement
priority: low
status: open
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
