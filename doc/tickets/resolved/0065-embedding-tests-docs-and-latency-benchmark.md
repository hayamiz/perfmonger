---
title: Embedding interface docs, pytest + C harness, and startup-latency benchmark
type: test
priority: medium
status: resolved
created: 2026-08-07
updated: 2026-08-08
---

## Description

Verification and documentation layer for the C-embedding interface
(#0009). Depends on #0063 (`--ready-fd`/`--stop-fd`) and #0064 (`--meta`);
the latency benchmark also needs #0063's ready signal.

### Documentation

- Document the CLI contract from #0009: `--ready-fd`/`--stop-fd`
  semantics, `--meta` encoding, exit codes 0/1/2, the `--background`
  incompatibility, and the C-side `pipe`/`fork`/`dup2`/`waitpid` recipe.
- Note that foreground recordings do not use the session PID file and are
  safe to run concurrently (one process per unique `-l` path).

### pytest (`tests/test_embedding.py`)

- Two `os.pipe()` pairs; child gets `ready`/`stop` fds via
  `pass_fds`/`dup2`.
- Assert the ready JSON schema (event/version/pid/output/start_time),
  then close the stop fd after ~0.3 s and assert exit 0.
- Assert `play` restores the recording and shows the injected `--meta`.
- Assert exit 2 on a write failure (e.g. output to `/dev/full`).

### Minimal C test harness (#0009 D9)

- A ~20–30 line C program exercising `fork`/`exec`/`dup2` of the ready
  and stop fds, invoked from the pytest suite.
- Skip (marker) when no C compiler is available, mirroring the existing
  `@requires_gnuplot` / `@requires_proc_diskstats` pattern.

### Startup-latency benchmark (#0009 D8)

- Measure time-to-ready and total wall time for a ~0.1 s recording via
  `--ready-fd`.
- Record the observed numbers back into #0009. Server mode
  (`--ctl-socket`) is only reconsidered (as a future v3 ticket) if this
  benchmark shows startup exceeds the ~50 ms threshold.

### Acceptance

- `uv run pytest tests/test_embedding.py -v` passes (C harness skipped
  when no compiler).
- Benchmark numbers are appended to #0009.
- Docs updated to describe the contract.

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Fix strategy: worktree (new docs + pytest + C harness + benchmark).
- Affected files: tests/test_embedding.py (extend), new tests/ files
  (C harness + latency benchmark), a docs file for the embedding contract.
- Notes: Verification/docs layer. #0063 and #0064 are now landed, so the
  flags (--ready-fd/--stop-fd/--meta) exist and can be exercised. Follows
  existing test conventions (conftest run_perfmonger, @requires_* skip
  markers). C harness skips when no compiler is present. Also fold in the
  #0063 reviewer observation: document the --stop-fd read-end contract and
  add a check/test that a wrong (write) end is rejected or clearly caveated.
  Latency numbers must be reported back so the parent can append them to
  #0009 (the ticket file cannot ride the branch).

## Implementation Notes

Plan and acceptance criteria are in the Description above (docs, pytest,
C harness, latency benchmark). Append the measured latency numbers back
into #0009 per D8.

## Resolution

Fixed on branch `ticket/0065-embedding-tests-docs-and-latency-benchmark`
(commit `6a8d736`), not yet landed on `master`.

What changed (docs/tests only — no production source change):
- `doc/embedding.md` (new): the full C-embedding contract — `--ready-fd`/
  `--stop-fd` semantics + ready JSON schema, exit codes 0/1/2,
  `--background` incompatibility, `--meta` RFC 3986 encoding, the
  pipe/fork/dup2/read/close/waitpid C recipe, the #0063 caveat that
  `--stop-fd` stops on EOF *or read error* (pass the READ end), and the
  foreground-concurrency note. README gets a minimal pointer.
- `tests/conftest.py`: `requires_cc` skip marker (mirrors
  `@requires_gnuplot`).
- `tests/test_embedding.py` extended: invalid fd → 1, `--ready-fd`/
  `--stop-fd` + `--background` → 1, ready JSON schema, `/dev/full` → 2,
  combined ready+stop+meta E2E, C-harness compile+run, latency benchmark.
- `tests/embedding_harness.c` (new): pipe/fork/dup2/execvp harness that
  passes the read end as stop-fd, reads the ready line, closes the stop
  write end, and asserts on the child exit status + ready line.
- `.gitignore`: ignore the regenerated `tests/embedding_latency.txt`.

Measured startup latency (`-i 0.1`, cc present, harness RAN not skipped):
time-to-ready min ~13.8 ms / median ~17.5 ms; total wall min ~17.3 ms /
median ~22.0 ms — well under the ~50 ms D8 threshold, so a server mode is
NOT warranted. Numbers appended to #0009.

Evaluator: PASS — Sonnet evaluator (Codex CLI unavailable: 401
Unauthorized). Ran `go test -v -cover`, `go test ./cmd/...`, `go vet`,
`make build`, full pytest 54/54; confirmed the C-harness test actually
ran (cc=/usr/bin/cc), the invalid-fd/`--background`/`/dev/full` tests
assert the specific exit codes, the harness is non-tautological (asserts
exit status + ready line), the docs contain the stop-fd write-end caveat
and ready schema, and `git diff ...-- core/` is empty (no source change).

human-review: optional (docs + tests + C harness + benchmark; no
production source change, no runtime behavior change, tests present and
confirmed to run) — review skipped as low-risk.

Review-state: ready-to-apply
