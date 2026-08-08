# Embedding PerfMonger in C Programs

This document describes the invocation contract that lets a C program (or any
process that can `fork`/`exec` and pass file descriptors) drive a `perfmonger
record` session reliably: start recording, learn *when* it is actually armed,
attach opaque metadata to the output, and stop it cleanly without racing on
signals or a pidfile.

The intended use case is bracketing short units of work — for example a DBMS
that wants to record system activity for the duration of a single query and tag
the resulting `.pgr` with the query text and id. See
[doc/tickets/0009-embedding-api-for-c-callers.md](tickets/0009-embedding-api-for-c-callers.md)
for the design discussion behind this contract.

Everything below reflects the behavior of the `record` subcommand as
implemented in
[core/cmd/perfmonger/record.go](../core/cmd/perfmonger/record.go) and
[core/cmd/perfmonger-core/recorder/recorder.go](../core/cmd/perfmonger-core/recorder/recorder.go).

---

## 1. The contract at a glance

| Concern            | Mechanism                                                        |
|--------------------|------------------------------------------------------------------|
| Start & arm        | `record ... --ready-fd=N`; caller `read()`s the ready line       |
| Stop cleanly       | `record ... --stop-fd=M`; caller `close()`s the pipe's write end |
| Attach metadata    | `record ... --meta key=value` (repeatable, percent-encoded)      |
| Choose output      | `-l <path>` (each session gets its own unique path)              |
| Success/failure    | Exit code `0` / `1` / `2`                                         |

`--ready-fd` and `--stop-fd` are **incompatible with `--background`**: the
background mode re-execs a detached child, which severs the inherited file
descriptors the two flags depend on. Passing either together with
`--background` fails at startup with exit code `1`.

---

## 2. `--ready-fd=N` — "recording is armed" signal

`--ready-fd=N` tells the recorder to write a single, newline-terminated JSON
object to file descriptor `N` **after the first sample has been flushed**, then
`close()` the descriptor. Because the event fires only once the first sample is
on disk, a caller that blocks reading `N` until it gets a line knows the
measurement window is genuinely open — there is no race where the child has been
`exec`'d but is not yet sampling.

Wiring `--ready-fd` also **suppresses the human-readable
`[recording to <path>]` banner** on stderr, so the machine-readable event is the
single source of truth for "started".

### Ready event schema (version 1)

```json
{"event":"ready","version":1,"pid":12345,"output":"/tmp/q42.pgr.gz","start_time":"2026-08-08T09:41:12.837Z"}
```

| Field        | Type   | Meaning                                                   |
|--------------|--------|-----------------------------------------------------------|
| `event`      | string | Always `"ready"` for this event.                          |
| `version`    | number | Schema version. Currently `1`.                            |
| `pid`        | number | PID of the recording process (use it if you must signal). |
| `output`     | string | The resolved output path (`-l` value).                    |
| `start_time` | string | RFC 3339 / ISO 8601 timestamp (UTC) of arming.            |

The line is exactly one JSON object followed by `\n`, and the descriptor is
closed immediately after. A reader should read until newline (or EOF) and parse
the accumulated bytes as JSON.

An **invalid or non-writable** `--ready-fd` (a descriptor that is not open, or
is open read-only) is rejected during setup with exit code `1`, before any
output file is written.

---

## 3. `--stop-fd=M` — stop-on-EOF signal

`--stop-fd=M` tells the recorder to watch file descriptor `M` in a background
goroutine and stop recording **as soon as that descriptor reaches EOF** (or a
read error). On stop, the recorder takes the graceful shutdown path: it flushes
the final sample, writes the gzip trailer (for `.gz` output), `fsync`s and
closes the file, and exits `0`.

The idiomatic setup is a `pipe(2)`: the caller passes the **read end** to the
child as `--stop-fd` and keeps the **write end** open for the lifetime of the
work being measured. To stop, the caller `close()`s the write end; the child
sees EOF and shuts down. This is more robust than sending SIGINT/SIGTERM:

- it cannot be lost or delivered to the wrong process,
- the caller does not have to track the child's PID,
- if the caller crashes, the kernel closes the write end automatically, so the
  recording stops on its own instead of leaking a runaway process.

`--stop-fd=0` (stdin) is a valid shorthand when the caller drives the child
over its stdin.

### ⚠ Sharp edge: pass the READ end, not the write end

`--stop-fd` stops on EOF **or any read error**. If you accidentally hand the
child the **write end** of the pipe, the very first read on it fails
immediately, which the recorder treats as a stop condition — so recording ends
almost as soon as it begins. The symptom is a recording that contains only one
or two samples no matter how long you wait.

Always pass the **read end** of the pipe as `--stop-fd`, keep the write end in
the parent, and close the write end when you want to stop. As with any pipe
handoff, the parent must also close its own copy of the read end after `fork`
so that closing the write end actually produces EOF for the child.

An invalid (unopened) `--stop-fd` is rejected during setup with exit code `1`.

---

## 4. `--meta key=value` — metadata injection

`--meta key=value` attaches an opaque key/value pair to the recording's header;
it is repeatable. The pair is split on the **first raw `=`**, and both key and
value are **percent-decoded** per RFC 3986 (`url.PathUnescape`). Encode any
literal `=`, space, or other reserved character in either half so it survives
the split and decode:

```
--meta query_id=q42
--meta sql=SELECT%20*%20FROM%20t%20WHERE%20id%3D42
```

The second example decodes to the key `sql` and the value
`SELECT * FROM t WHERE id=42` (the `%3D` becomes a literal `=` that does *not*
trigger another split, because only the first raw `=` is used as the
separator).

An argument with no raw `=`, or with an invalid percent-escape in either half,
is malformed and fails at startup with exit code `1`.

Stored metadata round-trips into the `.pgr` header and is surfaced downstream:

- `perfmonger play <file>` prints `# meta: key = value` lines to **stderr**
  (stdout stays valid line-delimited JSON records).
- `perfmonger summary --json <file>` includes a decoded `"meta"` object;
  the text summary shows a `Metadata:` block.

---

## 5. Exit codes

| Code | Meaning                                                                      |
|------|------------------------------------------------------------------------------|
| `0`  | Success — recording completed and was flushed/closed cleanly.                |
| `1`  | Setup error — bad flags/args before recording began (invalid `--ready-fd`/`--stop-fd`, `--background` combined with them, malformed `--meta`, unresolvable device, etc.). No usable output is produced. |
| `2`  | Collection error — a failure *after* recording started, e.g. a write/flush error to the output file (a full disk). |

A C caller should treat `1` as "I mis-invoked perfmonger" and `2` as "the
recording started but could not be completed", and surface both loudly rather
than assuming the `.pgr` is valid.

---

## 6. Concurrency & output paths

Foreground recordings (i.e. anything **not** using `--background`) do **not**
touch the per-user session PID file (`$TMPDIR/perfmonger-<user>-session.pid`)
and do not take the session lock. They are therefore safe to run concurrently:
run one `perfmonger record` process per unit of work, each with its own unique
`-l <path>`. There is no global single-session limit in this mode — the only
requirement is that concurrent recordings write to distinct output paths.

The single-session pidfile machinery (`--status`, `--kill`, the
"already running" check) applies only to `--background` sessions, which is the
other reason the fd-driven contract deliberately excludes `--background`.

---

## 7. C-side recipe

The full pattern is: create two pipes, `fork`, in the child `dup2` the pipe ends
onto chosen fds and `execvp` perfmonger, in the parent read the ready line, do
the work, then close the stop pipe's write end and `waitpid`.

```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/wait.h>

/* Record system activity around a unit of work, tagging it with metadata.
 * Returns the child's exit status (0 on success), or -1 on a local error. */
int record_bracketed(const char *out_path, const char *meta_kv) {
    int ready[2], stop[2];
    if (pipe(ready) != 0 || pipe(stop) != 0) return -1;

    pid_t pid = fork();
    if (pid < 0) return -1;

    if (pid == 0) {
        /* Child: wire the pipe ends onto fixed fds, then exec perfmonger.
         * ready write end -> fd 3, stop read end -> fd 4. */
        dup2(ready[1], 3);
        dup2(stop[0],  4);
        /* Close every original descriptor so only the dup'd copies remain. */
        close(ready[0]); close(ready[1]);
        close(stop[0]);  close(stop[1]);
        execlp("perfmonger", "perfmonger", "record",
               "-l", out_path,
               "-i", "0.1",
               "--ready-fd", "3",
               "--stop-fd",  "4",          /* the READ end — never the write end */
               "--meta", meta_kv,
               (char *)NULL);
        _exit(127);                        /* exec failed */
    }

    /* Parent: keep the ready read end and the stop WRITE end; close the
     * child-owned ends so EOF/close semantics work correctly. */
    close(ready[1]);   /* child is the only writer of the ready pipe */
    close(stop[0]);    /* child is the only reader of the stop pipe  */

    /* Block until the recorder reports it is armed (first sample flushed). */
    char line[512];
    ssize_t n = read(ready[0], line, sizeof(line) - 1);
    close(ready[0]);
    if (n <= 0) { close(stop[1]); waitpid(pid, NULL, 0); return -1; }
    line[n] = '\0';
    /* line is the ready JSON, e.g. {"event":"ready",...}; parse if desired. */

    /* ---- do the work being measured here ---- */

    /* Stop: closing the write end makes the child see EOF on its --stop-fd. */
    close(stop[1]);

    int status = 0;
    waitpid(pid, &status, 0);
    return WIFEXITED(status) ? WEXITSTATUS(status) : -1;
}
```

Key points the recipe encodes:

- The parent **must** close its copy of the stop **read** end (`stop[0]`) after
  the fork; otherwise the child never observes EOF when the parent closes
  `stop[1]`, and the recording never stops.
- The parent likewise closes the ready **write** end so its own `read()` sees
  EOF if the child dies before emitting the event.
- The child passes the pipe's **read** end as `--stop-fd` (see §3's sharp edge).
- Inspect the child's exit status: `0` success, `1` setup error, `2` collection
  error (§5).

A minimal, self-contained version of this harness lives at
[tests/embedding_harness.c](../tests/embedding_harness.c) and is exercised by
the pytest suite in
[tests/test_embedding.py](../tests/test_embedding.py).
