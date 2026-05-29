---
title: Embedding-friendly invocation interface for C callers (e.g. DBMS query tracing)
type: feature
priority: medium
status: open
created: 2026-04-20
updated: 2026-04-20
---

## Description

Today perfmonger is designed primarily as a CLI tool for interactive/human
use. A new usage mode under consideration is **embedded measurement driven
from a C program** (e.g. a DBMS that wants to bracket every query with a
perfmonger recording and tag the resulting `.pgr` with query metadata).

Concretely, the expected flow on the C side is:

1. At query start: spawn perfmonger and have it begin recording immediately.
2. At query end: tell perfmonger to stop cleanly, flush, and exit.
3. Associate the produced `.pgr` with query info (SQL text, query id,
   parameters, timing, exit status).

The goal of this ticket is to enumerate what perfmonger's **invocation
interface** would need to offer so that a C caller can drive start/stop
reliably, pair output with query metadata, and fail loudly when something
goes wrong — without having to reverse-engineer stdout banners, scrape
`/proc`, or race on `--background`'s pidfile.

### Current interface (for reference)

- `record` in foreground: runs until `--timeout` elapses or a signal
  arrives. `--no-intr` is on by default so SIGINT is ignored; stopping
  relies on timeout or SIGTERM.
- `record --background`: daemonizes via re-exec, writes a per-user session
  PID file to `$TMPDIR/perfmonger-<user>-session.pid`.
- `record --status` / `record --kill`: inspect/terminate the single
  per-user background session. `--kill` sends SIGINT with exponential
  backoff and unlinks the session file.
- No stable machine-readable "recording has started" signal, no
  caller-supplied session id, and no way to attach arbitrary metadata
  (e.g. query text) into the `.pgr` stream.

### Candidate interface improvements to evaluate

These are **candidates for discussion**, not a committed design. The
ticket is to research, prototype-on-paper, and recommend which subset is
worth implementing.

1. **Ready signal on stdout/fd.** A structured "I am recording" line
   (e.g. `{"event":"started","pid":1234,"output":"/tmp/q42.pgr"}`) written
   on a dedicated fd (`--ready-fd=N`) or stdout, before the first sample,
   so the C caller can `read()` until it sees it and know recording is
   armed. Removes the current race where the child may not yet be
   sampling when the parent returns from `fork+exec`.

2. **Stop-on-stdin-close / stop-on-fd.** `--stop-when-fd-closes=N` (or
   stop when stdin hits EOF). The caller keeps a pipe open for the life
   of the query and `close()`s it to end the recording. This is more
   robust than SIGINT/SIGTERM because it can't be lost, doesn't require
   the caller to track a PID, and is automatically triggered if the
   caller crashes (kernel closes the fd).

3. **Caller-supplied session id / output path.** Allow
   `--session-id=<str>` or simply a well-defined `-l <path>` contract so
   multiple concurrent recordings (one per query) can coexist. The
   current single-per-user session file assumption needs to be relaxed
   or bypassed for this mode.

4. **Concurrent-session support.** Either:
   - Drop the global session-file assumption when `--background` is not
     used (foreground mode driven by pipe/fd is already safe), or
   - Namespace the session file by `--session-id` so N queries can
     record in parallel.

5. **Metadata injection.** A way to attach opaque key/value metadata to
   the recorded stream: `--meta key=value` (repeatable) and/or reading
   a JSON blob from a side fd (`--meta-fd=N`). The recorder would emit
   these as a header record in the `.pgr` so `play`/`summary` can
   surface the query context. Alternative: leave metadata to the caller
   and standardize only the filename/sidecar convention.

6. **Deterministic exit & flush semantics.** On stop signal (SIGTERM,
   fd close, or `--kill` equivalent), guarantee: (a) final sample is
   flushed, (b) gzip trailer written, (c) exit code reflects
   success/failure, (d) output file is `fsync`'d. Document this
   contract; today some of it is implicit.

7. **Structured errors on stderr.** One-line JSON error records (e.g.
   `{"event":"error","code":"device_not_found","detail":"sdz"}`) so the
   C caller doesn't have to parse `[ERROR] ...` strings.

8. **Low-overhead start.** Measure/reduce startup cost so bracketing a
   short (sub-second) query is viable. If startup is >50ms, consider a
   long-running `record --server` mode that accepts
   start/stop/metadata commands over a UNIX socket — one perfmonger
   process per DBMS, multiple recordings over its lifetime.

9. **Library / cgo surface (stretch).** Optionally expose
   `perfmonger-core` as a C-callable shared library so the DBMS can
   avoid `fork+exec` altogether. Much larger scope; likely out of the
   first cut, but worth noting so the CLI interface doesn't preclude it.

### Deliverable for this ticket

A written recommendation (posted back into this ticket as
`## Implementation Notes` during triage) that answers:

- Which of the above are in-scope for a first C-embedding milestone?
- Which can be layered in later without breaking the first cut?
- What is the proposed CLI contract (flags, exit codes, stdout/stderr
  format) that a C caller would code against?
- Does concurrent-session support require changes to the session-file
  and lock-file logic in [record.go](core/cmd/perfmonger/record.go)?
- What minimal integration test (pytest) would prove the
  start/record/stop/metadata flow end-to-end?

No code changes in this ticket — design only. A follow-up ticket (or
tickets) should carry the implementation once the shape is agreed.
