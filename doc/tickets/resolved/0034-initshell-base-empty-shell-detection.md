---
title: filepath.Base("") returns "." causing misleading "unsupported shell: ." error
type: bug
priority: low
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/initshell.go` around lines 113-131, shell detection runs
`ps` against the parent PID. If `ps` succeeds but returns empty output — possible
when the parent process has already exited and been reaped between `os.Getppid()`
and the `ps` execution — `strings.TrimSpace` yields an empty string, and
`filepath.Base("")` returns `"."` per the Go spec.

The switch in `opts.run()` then falls through to the default case and emits
`unsupported shell: . Only bash and zsh are supported`, which is unintelligible.
The real cause is a detection failure, not an unsupported shell, so the message
misleads the user.

Why it is a bug: a transient detection failure produces a confusing, incorrect
error message.

Suggested direction: detect the empty/`"."` result before the switch and emit a
clear "could not detect parent shell" message (and/or fall back to the `SHELL`
environment variable), rather than feeding `"."` into the shell-name switch.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: getParentShell() can return "" (ps succeeds but empty), and filepath.Base("") returns ".", yielding "unsupported shell: .". Fix: check for empty before the switch and return a clear "could not detect parent shell" error.

## Resolution

Root cause: `getParentShell()` passed raw `ps`/`SHELL` output straight to
`filepath.Base()`. When `ps` succeeded but returned an empty string,
`filepath.Base("")` yielded `"."`, which fell through the shell switch to the
default case and produced the misleading `unsupported shell: .` message.

Changes in `core/cmd/perfmonger/initshell.go`:

- Extracted a pure, testable helper `normalizeShellName(raw string) string`
  that trims input and returns `""` (instead of `"."`) for empty/whitespace-only
  output, so a failed detection is distinguishable.
- Added `errorForShell(shell string) error`, which returns
  `"could not detect parent shell"` when the shell name is empty and the
  previous `"unsupported shell: %s. Only bash and zsh are supported"` message
  otherwise.
- Rewired `getParentShell()` to use `normalizeShellName`, falling back to the
  `SHELL` env var when `ps` output normalizes to empty, and updated the default
  branches of both `run()` and `runInitShell()` to use `errorForShell`.

Tests added in `core/cmd/perfmonger/initshell_test.go`:

- `TestNormalizeShellName` (table-driven): covers bash, full-path zsh,
  surrounding whitespace, empty output, and whitespace-only output.
- `TestRunDetectionFailureMessage`: asserts the empty-shell case maps to the
  clear `"could not detect parent shell"` error.

Verification: `go test -cover ./...` (internal/perfmonger) and
`go test ./cmd/perfmonger/` pass; `go vet` clean; binary rebuilt.
