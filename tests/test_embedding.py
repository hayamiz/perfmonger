"""Tests for the C-embedding start/stop fd contract (--ready-fd / --stop-fd).

Exercises the record subcommand the way a C caller would: it forks the binary
with two os.pipe() pairs passed through as inherited file descriptors, waits for
the machine-readable ready event, then stops recording by closing the stop
pipe. #0065 will extend this with the C harness, latency benchmark, and --meta.
"""

import json
import os
import subprocess

from conftest import _find_perfmonger_bin, run_perfmonger, requires_proc_diskstats


@requires_proc_diskstats
def test_embedding_ready_fd_and_stop_fd(tmp_path):
    """Ready event is emitted on ready-fd; closing stop-fd stops with exit 0."""
    bin_path = _find_perfmonger_bin()
    out = tmp_path / "embed.pgr.gz"

    # ready pipe: child writes the ready event to ready_w, parent reads ready_r.
    # stop pipe:  child reads stop_r, parent signals stop by closing stop_w.
    ready_r, ready_w = os.pipe()
    stop_r, stop_w = os.pipe()

    proc = subprocess.Popen(
        [
            bin_path, "record",
            "-l", str(out),
            "-i", "0.1",
            "--ready-fd", str(ready_w),
            "--stop-fd", str(stop_r),
        ],
        pass_fds=(ready_w, stop_r),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )

    # The parent's copies of the child-owned ends must be closed so EOF
    # propagates correctly (child is the only writer of ready_w / reader of
    # stop_r once these are closed).
    os.close(ready_w)
    os.close(stop_r)

    stop_w_open = True
    try:
        # Block until the recorder writes its ready line (after the first sample
        # is flushed). fdopen takes ownership of ready_r and closes it on exit.
        with os.fdopen(ready_r, "r") as ready_f:
            line = ready_f.readline()

        assert line, "no ready event was received on the ready fd"
        event = json.loads(line)
        assert event["event"] == "ready"
        assert event["version"] == 1
        assert event["pid"] == proc.pid
        assert event["output"] == str(out)
        assert "start_time" in event and event["start_time"]

        # Signal stop by closing the write end of the stop pipe: the recorder's
        # stop goroutine sees EOF and stops on the graceful exit path.
        os.close(stop_w)
        stop_w_open = False

        rc = proc.wait(timeout=10)
        stderr = proc.stderr.read().decode()
        assert rc == 0, f"record exited with {rc}; stderr={stderr}"
    finally:
        if stop_w_open:
            try:
                os.close(stop_w)
            except OSError:
                pass
        if proc.poll() is None:
            proc.kill()
            proc.wait()
        proc.stdout.close()
        proc.stderr.close()

    # The output must exist and be playable back to JSON.
    assert out.exists(), "recorder did not create the output file"
    played = run_perfmonger("play", str(out))
    assert played.returncode == 0, f"play failed: {played.stderr}"
    assert played.stdout.strip(), "play produced no JSON output"
