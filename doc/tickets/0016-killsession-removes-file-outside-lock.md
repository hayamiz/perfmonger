---
title: killSession removes the session file outside the lock, racing with concurrent recorder start
type: bug
priority: low
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 2/3 skeptics.

In `core/cmd/perfmonger/record.go`, `killSession` verifies the target process is
dead at line 171 (`syscall.Kill(pid, 0) != nil`) and then removes the session file
at line 179 **without holding the flock**.

This opens a race with a concurrent `perfmonger record --background` invocation.
Between lines 171 and 179, the concurrent start can call `getRunningSessionPID`,
read the still-existing PID file, find the process dead, remove the session file
itself, conclude no session is running, and proceed to start a new recorder —
which then calls `WriteSessionFile` and writes a fresh PID file. `killSession`
then executes its `Remove` at line 179 and deletes that brand-new session file,
leaving a running recorder with no session file and corrupting session-tracking
state (`--status`/`--kill` can no longer find the live process).

Why it is a bug: the check-then-remove sequence is not atomic with respect to
other processes, so cleanup of a dead session can clobber a freshly started one.

Suggested direction: perform the dead-process check and the session-file removal
while holding the same exclusive flock used by `WriteSessionFile`, and re-validate
that the PID in the file still matches the dead PID immediately before removing it.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: Race between killSession removal and concurrent recorder start; the correct locking discipline (lock-protected check-and-remove vs centralizing cleanup) is a design choice.

## Implementation Notes

killSession() performs a dead-process check and then removes the session file without holding any lock. A concurrent `perfmonger record --background` can read the still-existing PID file, clean it up itself, start a new recorder and write a fresh PID file — which killSession's later Remove then deletes, orphaning the new recorder.

Approaches:
- Option A (recommended): Lock-protected check-and-remove in killSession. Acquire the same flock used by WriteSessionFile/RemoveSessionFile; inside the lock re-read the PID file, verify it still matches the target PID and the process is still dead, then remove. Mirrors the atomic pattern already used elsewhere.
- Option B: Centralize the dead-process cleanup logic into a shared function called from both getRunningSessionPID and killSession.

Decision point: whether re-validation inside the lock is sufficient, or a separate check-and-remove helper is preferred.
