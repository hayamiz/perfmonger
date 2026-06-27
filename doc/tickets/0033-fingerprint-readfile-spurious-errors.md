---
title: fingerprint readFile inflates f.errors for expected-missing files, marking steps failed
type: bug
priority: low
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/fingerprint.go` around lines 120-127, `readFile`
unconditionally appends any error to `f.errors` before returning it. Several
callers intentionally probe for files that may legitimately not exist —
`saveProcInfo` (lines 153, 165) and `saveDistroInfo` (line 304), e.g.
`/etc/redhat-release` on a Debian system, or unreadable `/proc/sys/fs` entries.

Each such expected-absent file increments `f.errors`, and because the error count
is used as the pass/fail indicator, `doWithMessage` reports the enclosing step as
"failed" even though the function handled the absence gracefully. Concretely,
"Saving distro info" is marked failed on every non-RHEL system simply because
`/etc/redhat-release` is absent.

Why it is a bug: normal, expected "file not found" conditions are recorded as
failures, producing misleading status on the majority of systems.

Suggested direction: do not record expected-absent files as errors — either have
`readFile` return the error without appending to `f.errors` and let callers decide,
or add an "optional file" variant that treats `os.IsNotExist` (and expected
permission errors) as a non-error. Reserve `f.errors` for genuinely unexpected
failures.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: readFile unconditionally appends every error to f.errors, marking expected-missing files (e.g. /etc/redhat-release on Debian) as failures. Multiple reasonable error-classification approaches exist.

## Implementation Notes

readFile appends all errors to f.errors before returning; callers that probe optional files inflate the error count, and since pass/fail keys off that count, normal conditions report as failures.

Options:
- Option 1 (recommended): readFile returns the error without appending; callers explicitly append only truly-unexpected failures.
- Option 2: add a readFileOptional() variant treating os.IsNotExist (and expected permission errors) as non-errors.
- Option 3: pass an "expected" flag/list to readFile; only unexpected errors are appended.

Decision points: which files are optional vs required; should callers or readFile decide; how to treat expected permission errors (e.g. /proc/sys/fs).
