---
title: Errors from os.ReadFile and os.WriteFile silently discarded when saving gnuplot files
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/plot.go`, the `SaveGpfiles` block around lines 301-310
discards I/O errors. `os.ReadFile` is called with its error return ignored via
`_`, and the subsequent `os.WriteFile`'s error return is also ignored.

Failure modes:

- If the read fails — e.g. a `.gp` file was never written because that plot was
  skipped — `os.WriteFile` then writes zero bytes to the destination with no
  diagnostic.
- If `os.WriteFile` fails — e.g. the output directory is read-only — the failure
  is silently swallowed and the command still exits 0.

In either case the user gets no indication that their `--save` request was
partially or fully unsuccessful.

Why it is a bug: a user-visible "save my plot scripts" operation can fail
completely while reporting success.

Suggested direction: check the error returns from both `os.ReadFile` and
`os.WriteFile`, and propagate any failure so the `plot` command exits non-zero with
a clear message identifying the file that could not be saved.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: os.ReadFile/os.WriteFile errors in SaveGpfiles are discarded. Fix: check both returns and propagate with context, exit non-zero. Unambiguous.
