---
title: out.Flush() error ignored in recording loop, silently dropping data on full disk
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 2/3 skeptics.

In `core/cmd/perfmonger-core/recorder/recorder.go` around lines 429-433,
`out.Flush()` is called after each successful `enc.Encode(record)`, but its error
return value is completely ignored.

If the underlying write fails — for example the disk fills up, the `gzip.Writer`
errors, or the output file descriptor is closed externally — the flush error is
swallowed and the sampling loop continues. Subsequent `enc.Encode` calls keep
writing into the `bufio` buffer, which either keeps failing silently or grows
without ever reaching durable storage. The output `.pgr`/`.pgr.gz` file ends up
truncated or corrupted, and the process keeps running as if nothing happened
instead of exiting with an error.

Why it is a bug: the recorder reports success and produces an unusable output file
on a common, recoverable failure (disk full). The user has no signal that
recording stopped persisting data.

Suggested direction: check the return value of `out.Flush()` (and ideally the
`enc.Encode` return), and on error break the loop and propagate the failure so the
process exits non-zero with a clear message.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: Check out.Flush() return value in the recording loop; break and propagate on error so the process exits non-zero. Mechanical.
