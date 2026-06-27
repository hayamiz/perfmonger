---
title: createTarball silently produces a corrupt archive when gzip/tar Close() fails
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/fingerprint.go` around lines 358-364, `createTarball`
closes the writers via `defer tarWriter.Close()`, `defer gzipWriter.Close()`, and
`defer tarFile.Close()`, which discards their return values.

`gzip.Writer.Close()` writes the GZIP trailer (CRC32 checksum + uncompressed
size), and `tar.Writer.Close()` writes the two 512-byte end-of-archive blocks. If
either close fails (e.g. the disk fills while flushing the final blocks, or the
underlying file was already closed), the error is lost and `createTarball` returns
`nil` (success). The caller in `run()` then prints "done" and exits 0, leaving a
truncated, unreadable `.tar.gz` on disk.

Why it is a bug: a corrupt archive is produced and reported as a successful
fingerprint.

Suggested direction: close `tarWriter` and `gzipWriter` explicitly (in order)
before returning, check each `Close()` error, and propagate any failure from
`createTarball` so `run()` reports the error and exits non-zero. Keep the deferred
closes only as a safety net for early-return error paths.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: createTarball's deferred Close() calls (tarWriter/gzipWriter/tarFile) discard errors, risking corrupt truncated archives. Fix: explicit ordered Close() with error checks before return, keeping defers as a safety net. Idiomatic.
