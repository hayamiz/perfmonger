---
title: checkPdfCairoSupported passes GnuplotBin unsanitized into a shell string (command injection)
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/plot.go` around lines 249-255, `checkPdfCairoSupported`
interpolates the gnuplot binary path from `cmd.GnuplotBin` directly into a shell
command string via `fmt.Sprintf`, which is then passed to `sh -c`.

If `GnuplotBin` contains shell metacharacters (a space, semicolon, quote, etc.) the
resulting shell command is malformed or executes unintended commands. Because the
value originates from the `--with-gnuplot` flag (or an environment variable that
supplies it), an attacker who can influence that value can inject arbitrary shell
commands that run with the privileges of the `perfmonger` process.

Why it is a bug: untrusted/attacker-controllable input is concatenated into a shell
command, which is a command-injection vulnerability and also breaks legitimately
on paths containing spaces.

Suggested direction: exec gnuplot directly with `exec.Command(cmd.GnuplotBin,
...)` (no `sh -c`) and write the `set terminal` probe script to its stdin; if a
shell is truly required, properly quote the path. Avoid building shell strings from
user-supplied paths.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: checkPdfCairoSupported passes GnuplotBin unsanitized into a shell string. Fix: use exec.Command(gnuplotBin, "-e", ...) instead of shell interpolation, eliminating injection and handling spaces.
