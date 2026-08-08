---
title: Embedding-friendly invocation interface for C callers (e.g. DBMS query tracing)
type: feature
priority: medium
status: resolved
created: 2026-04-20
updated: 2026-08-08
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
  arrives. Both SIGINT and SIGTERM trigger a graceful stop (a final
  sample is taken, then flush + exit). Note: `--no-intr` only suppresses
  interrupt-count sampling and has nothing to do with signal handling —
  the name merely collides with the legacy Ruby `--no-interrupt`.
- `record --background`: daemonizes via re-exec, writes a per-user session
  PID file to `$TMPDIR/perfmonger-<user>-session.pid`. This session PID
  file is written **only** in background mode; foreground recordings do
  not touch it and are therefore already safe to run concurrently (one
  process per unique `-l` output path).
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

## Triage

- Complexity: high
- Mechanical fix: no
- Requires user decision: yes
- Notes: This is a design-phase ticket requesting a written recommendation (CLI interface spec for C callers), not code. Multiple candidate mechanisms exist (ready-fd, stop-on-fd-close, session-id, metadata, structured errors) with different trade-offs; scope hinges on the user's chosen subset.

## Implementation Notes

### Confirmed design decisions (reviewed 2026-08-07)

All nine decision points were resolved in favor of the reviewer's
recommended option. Two carry additional requirements captured below.

- **D1 — Stop mechanism: `--stop-fd=N` only.** A goroutine reads fd N and
  closes the existing `StopCh` on EOF/error. stdin users pass
  `--stop-fd=0`. No pipe auto-detection (a `/dev/null` stdin would stop
  immediately).
- **D2 — Ready signal: `--ready-fd=N`, one JSON line then close.**
- **D3 — Ready fires after the first sample is flushed.** *Semantics
  (clarified):* the **perfmonger recorder child process** is the emitter,
  not the C caller. "Ready" means the recorder has captured and flushed
  the baseline sample, so the measurement window is armed. The C caller
  blocks on `read(ready_fd)` and only starts the query once it sees the
  line (or EOF). This closes the fork+exec race where the parent returns
  before the child is actually sampling, so the recording window already
  covers the instant the caller is unblocked.
- **D4 — Metadata in v1: `--meta key=value` (repeatable).** *Encoding
  requirement:* both key and value are **percent-encoded (RFC 3986)** so
  arbitrary strings (including `=`, spaces, newlines, non-ASCII) can be
  carried while simple values stay human-readable. Split on the first raw
  `=` (a literal `=` inside key/value is `%3D`), then percent-decode each
  side. Store decoded strings in `CommonHeader.Meta map[string]string`;
  `play`/`summary` surface them. gob field addition is forward/backward
  compatible, so no format version bump. (`--meta-fd` for very large
  values such as full SQL text stays v2.)
- **D5 — Stop-requested exit is 0.** Caller-driven stop is the normal
  path; do not use exit 3.
- **D6 — `--ready-fd`/`--stop-fd` are incompatible with `--background`**
  (daemonization detaches fds); using them together exits 1.
- **D7 — v1 ships exit codes 0/1/2 only.** stderr stays human-readable;
  `--json-errors` is v2.
- **D8 — Measure startup latency first.** Bundle a latency benchmark in
  the pytest suite and record the result in this ticket before
  considering a server mode.
- **D9 — Add a minimal C test harness** (fork/exec/dup2 path) in addition
  to the pytest pipe fixtures; skip when no C compiler is available.

### Prerequisite refactor (blocks the exit-code contract)

`RunDirect` currently returns no error and `panic`s on failures (e.g.
`os.Create`). Before the exit-code contract is possible it must be
refactored to return `error`, and `file.Sync()` must be added before
`Close`. This is the bulk of the v1 cost and was under-scoped in the
original skeleton.

### Exit codes

- `0` — normal (timeout, stop-fd close, SIGINT/SIGTERM all included)
- `1` — setup error (bad flag, output not creatable, invalid ready/stop
  fd); no sample written
- `2` — collection/write error (e.g. ENOSPC); output may be incomplete

### CLI contract (what a C caller codes against)

    perfmonger record --ready-fd 3 --stop-fd 4 \
        -l /var/tmp/q42.pgr.gz -i 0.1 \
        --meta query_id=42 --meta db=tpch

- ready-fd: after the first sample is flushed, write one newline-terminated
  JSON object to fd 3 and close it:
  `{"event":"ready","version":1,"pid":1234,"output":"...","start_time":"...Z"}`
- stop-fd: when fd 4 reaches EOF, take one final sample, then
  Flush → gzip Close → `file.Sync()` → Close → exit. SIGTERM/SIGINT and
  `--timeout` share this exit path.
- `--ready-fd` suppresses the `[recording to ...]` stdout banner.
- C-side: `pipe(rp); pipe(sp); fork()` → child `dup2(rp[1],3);
  dup2(sp[0],4); execvp()` → parent `read(rp[0])` to await armed, run the
  query, `close(sp[1])` to stop, `waitpid` for status.

### v1 scope / deferred

v1: RunDirect error refactor + exit codes; `--ready-fd`; `--stop-fd`;
`--meta` (percent-encoded); flush/fsync contract + docs (including the
foreground-concurrency note); `tests/test_embedding.py` (pipe fixtures +
latency benchmark) and a minimal C harness.

Deferred: `--meta-fd`, `--json-errors`, `--session-id` background
namespacing, finished event, output flock; server mode only if the D8
benchmark shows a problem; cgo library declined.

### Follow-up implementation tickets to file

1. Refactor `RunDirect` to return `error` and add `file.Sync()`
   (prerequisite; also lets other subcommands drop panic-based exits).
2. `--ready-fd` + `--stop-fd` with the exit-code contract; replace the
   `stat` 100 ms start sleep with the ready signal.
3. `--meta key=value` (percent-encoded) + `CommonHeader.Meta` +
   `play`/`summary` display.
4. Docs + `tests/test_embedding.py` + C harness + startup-latency
   benchmark.

### Implementation status (2026-08-08)

All four follow-up tickets are implemented:

- #0062 (RunDirect error return + fsync) — landed on `master`.
- #0063 (`--ready-fd` / `--stop-fd`) — landed on `master`.
- #0064 (`--meta key=value`, percent-encoded) — landed on `master`.
- #0065 (docs + pytest + C harness + latency benchmark) — reviewed
  (PASS, low-risk), ready to apply.

**D8 resolved — startup latency measured** (`record --ready-fd`, `-i 0.1`,
cc present, C harness exercised): time-to-ready min ~13.8 ms / median
~17.5 ms; total wall for a ~0.1 s recording min ~17.3 ms / median
~22.0 ms. Comfortably under the ~50 ms threshold, so the long-running
`record --ctl-socket` server mode is **not** warranted and stays
deferred. See `doc/embedding.md` for the finalized C-embedding contract.

## Resolution

Design delivered and implemented. This ticket was design-only; the agreed
CLI contract (see the "Confirmed design decisions" and "CLI contract"
sections above) was implemented across four follow-up tickets, all landed
on `master`:

- #0062 — `RunDirect`/`RunWithOption` return `error`; `SetupError`/
  `CollectionError`; exit codes 0/1/2; fsync before close.
- #0063 — `record --ready-fd` / `--stop-fd` (ready after first flush,
  stop on fd EOF); incompatible with `--background`; `stat` sleep replaced
  by the ready signal.
- #0064 — `record --meta key=value` (RFC 3986 percent-encoded) →
  `CommonHeader.Meta`; surfaced by `play`/`summary`; gob-compatible.
- #0065 — `doc/embedding.md` contract docs, pytest coverage + C harness,
  and the startup-latency benchmark.

D8 resolved: measured time-to-ready median ~17.5 ms / total ~22.0 ms
(< 50 ms), so the server mode stays deferred. Deferred to future tickets
as agreed: `--meta-fd`, `--json-errors`, `--session-id` background
namespacing, finished event, output flock, and the cgo library (declined).
A follow-up nicety noted in #0065: caveat/guard the `--stop-fd` write-end
misuse.
