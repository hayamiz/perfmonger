"""Tests for the C-embedding start/stop fd contract (--ready-fd / --stop-fd).

Exercises the record subcommand the way a C caller would: it forks the binary
with two os.pipe() pairs passed through as inherited file descriptors, waits for
the machine-readable ready event, then stops recording by closing the stop
pipe. #0065 will extend this with the C harness, latency benchmark, and --meta.
"""

import json
import os
import shutil
import statistics
import subprocess
import time
from pathlib import Path

from conftest import (
    _find_perfmonger_bin,
    run_perfmonger,
    requires_cc,
    requires_proc_diskstats,
)


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


def _run_embedding_session(bin_path, out, extra_args=(), stop_after=0.0):
    """Drive one record session over ready/stop pipes the way a C caller would.

    Returns (returncode, ready_event_dict, stderr_text). The stop pipe's write
    end is closed after `stop_after` seconds to end recording gracefully.
    """
    ready_r, ready_w = os.pipe()
    stop_r, stop_w = os.pipe()

    proc = subprocess.Popen(
        [
            bin_path, "record",
            "-l", str(out),
            "-i", "0.1",
            "--ready-fd", str(ready_w),
            "--stop-fd", str(stop_r),
            *extra_args,
        ],
        pass_fds=(ready_w, stop_r),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    os.close(ready_w)
    os.close(stop_r)

    stop_w_open = True
    try:
        with os.fdopen(ready_r, "r") as ready_f:
            line = ready_f.readline()
        event = json.loads(line) if line else None

        if stop_after:
            time.sleep(stop_after)
        os.close(stop_w)
        stop_w_open = False

        rc = proc.wait(timeout=10)
        stderr = proc.stderr.read().decode()
        return rc, event, stderr
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


@requires_proc_diskstats
def test_ready_event_schema(tmp_path):
    """The ready JSON event must carry the full, well-typed v1 schema."""
    bin_path = _find_perfmonger_bin()
    out = tmp_path / "schema.pgr.gz"

    rc, event, stderr = _run_embedding_session(bin_path, out)
    assert rc == 0, f"record exited with {rc}; stderr={stderr}"
    assert event is not None, "no ready event was received"

    # Exactly the documented keys, no more, no less.
    assert set(event.keys()) == {
        "event", "version", "pid", "output", "start_time"
    }, event
    assert event["event"] == "ready"
    assert event["version"] == 1
    assert isinstance(event["pid"], int) and event["pid"] > 0
    assert event["output"] == str(out)
    assert isinstance(event["start_time"], str) and event["start_time"]

    # The banner is suppressed when a ready fd is wired up.
    assert "[recording to" not in stderr


def test_invalid_ready_fd_is_rejected(tmp_path):
    """An unopened --ready-fd must fail at setup with exit code 1."""
    out = tmp_path / "bad.pgr.gz"
    # fd 9 is not opened in the child (run_perfmonger does not pass it through).
    rec = run_perfmonger(
        "record", "-i", "0.1", "--timeout", "1",
        "--ready-fd", "9",
        "-l", str(out),
    )
    assert rec.returncode == 1, rec.stderr
    assert not out.exists()


def test_invalid_stop_fd_is_rejected(tmp_path):
    """An unopened --stop-fd must fail at setup with exit code 1."""
    out = tmp_path / "bad.pgr.gz"
    rec = run_perfmonger(
        "record", "-i", "0.1", "--timeout", "1",
        "--stop-fd", "9",
        "-l", str(out),
    )
    assert rec.returncode == 1, rec.stderr
    assert not out.exists()


def test_ready_fd_with_background_is_rejected(tmp_path):
    """--ready-fd is incompatible with --background (exit 1)."""
    out = tmp_path / "bg.pgr.gz"
    # fd 1 (stdout) is valid, proving the rejection is about --background,
    # not about the fd being invalid.
    rec = run_perfmonger(
        "record", "--background", "--timeout", "1",
        "--ready-fd", "1",
        "-l", str(out),
    )
    assert rec.returncode == 1, rec.stderr


def test_stop_fd_with_background_is_rejected(tmp_path):
    """--stop-fd is incompatible with --background (exit 1)."""
    out = tmp_path / "bg.pgr.gz"
    rec = run_perfmonger(
        "record", "--background", "--timeout", "1",
        "--stop-fd", "0",
        "-l", str(out),
    )
    assert rec.returncode == 1, rec.stderr


@requires_proc_diskstats
def test_ready_stop_and_meta_end_to_end(tmp_path):
    """--ready-fd + --stop-fd + --meta coexist: record, arm, stop, then the
    injected metadata survives into play and summary."""
    bin_path = _find_perfmonger_bin()
    out = tmp_path / "combo.pgr.gz"

    rc, event, stderr = _run_embedding_session(
        bin_path, out,
        extra_args=["--meta", "query_id=q42",
                    "--meta", "sql=SELECT%20*%20WHERE%20id%3D42"],
        stop_after=0.3,
    )
    assert rc == 0, f"record exited with {rc}; stderr={stderr}"
    assert event is not None and event["event"] == "ready"
    assert out.exists()

    # play surfaces metadata on stderr; stdout stays valid JSON records.
    played = run_perfmonger("play", str(out))
    assert played.returncode == 0, played.stderr
    assert played.stdout.strip(), "play produced no JSON output"
    assert "query_id" in played.stderr and "q42" in played.stderr
    # %20 -> space, %3D -> literal '=' (only the first raw '=' splits the pair).
    assert "SELECT * WHERE id=42" in played.stderr

    # summary --json exposes a decoded "meta" object.
    summ = run_perfmonger("summary", "--json", str(out))
    assert summ.returncode == 0, summ.stderr
    data = json.loads(summ.stdout)
    assert data["meta"] == {
        "query_id": "q42",
        "sql": "SELECT * WHERE id=42",
    }


@requires_proc_diskstats
def test_collection_error_on_full_disk_exits_2(tmp_path):
    """A write failure after recording started is a collection error (exit 2).

    Writing to /dev/full always fails with ENOSPC once the recorder tries to
    flush a sample, so the setup succeeds but collection fails.
    """
    if not os.path.exists("/dev/full"):
        import pytest
        pytest.skip("/dev/full not available")

    rec = run_perfmonger(
        "record", "-i", "0.1", "--timeout", "2", "--no-gzip",
        "-l", "/dev/full",
    )
    assert rec.returncode == 2, f"expected exit 2, got {rec.returncode}: {rec.stderr}"


@requires_cc
@requires_proc_diskstats
def test_c_embedding_harness(tmp_path):
    """Compile and run the minimal C harness, proving a real C caller can drive
    the ready/stop fd contract end to end via fork/exec/dup2/waitpid."""
    bin_path = _find_perfmonger_bin()
    src = Path(__file__).resolve().parent / "embedding_harness.c"
    assert src.exists(), f"C harness source missing: {src}"

    harness = tmp_path / "embedding_harness"
    compile_proc = subprocess.run(
        [shutil.which("cc"), str(src), "-o", str(harness)],
        capture_output=True, text=True,
    )
    assert compile_proc.returncode == 0, (
        f"cc failed to compile harness: {compile_proc.stderr}"
    )

    out = tmp_path / "harness.pgr.gz"
    run = subprocess.run(
        [str(harness), bin_path, str(out)],
        capture_output=True, text=True, timeout=30,
    )
    assert run.returncode == 0, (
        f"harness exited {run.returncode}; stdout={run.stdout!r} "
        f"stderr={run.stderr!r}"
    )

    # The harness prints the ready line it received; it must be the ready event.
    event = json.loads(run.stdout.strip())
    assert event["event"] == "ready"
    assert event["version"] == 1

    # The recording the harness produced must be playable.
    assert out.exists(), "harness did not produce an output file"
    played = run_perfmonger("play", str(out))
    assert played.returncode == 0, f"play failed: {played.stderr}"
    assert played.stdout.strip(), "play produced no JSON output"


@requires_proc_diskstats
def test_startup_latency_benchmark(tmp_path):
    """Measure time-to-ready (spawn -> ready line) and total wall time for a
    short recording over several iterations, report min/median, and assert
    loosely so the benchmark is informative but not flaky (#0009 D8)."""
    bin_path = _find_perfmonger_bin()
    iterations = 7

    ttr_samples = []
    wall_samples = []

    for i in range(iterations):
        out = tmp_path / f"bench-{i}.pgr.gz"
        ready_r, ready_w = os.pipe()
        stop_r, stop_w = os.pipe()

        t0 = time.monotonic()
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
        os.close(ready_w)
        os.close(stop_r)

        try:
            with os.fdopen(ready_r, "r") as ready_f:
                line = ready_f.readline()
            t_ready = time.monotonic()
            assert line, "no ready event received during benchmark"

            os.close(stop_w)
            rc = proc.wait(timeout=10)
            t_done = time.monotonic()
            assert rc == 0, f"record exited {rc}: {proc.stderr.read().decode()}"

            ttr_samples.append(t_ready - t0)
            wall_samples.append(t_done - t0)
        finally:
            if proc.poll() is None:
                proc.kill()
                proc.wait()
            proc.stdout.close()
            proc.stderr.close()

    ttr_min = min(ttr_samples)
    ttr_med = statistics.median(ttr_samples)
    wall_min = min(wall_samples)
    wall_med = statistics.median(wall_samples)

    report = (
        "perfmonger embedding startup-latency benchmark "
        f"({iterations} iterations, -i 0.1)\n"
        f"time-to-ready: min={ttr_min * 1000:.1f}ms "
        f"median={ttr_med * 1000:.1f}ms\n"
        f"total wall:    min={wall_min * 1000:.1f}ms "
        f"median={wall_med * 1000:.1f}ms\n"
    )
    # Visible under `pytest -s`.
    print("\n" + report)
    # Durable artifact the caller can read back.
    artifact = Path(__file__).resolve().parent / "embedding_latency.txt"
    artifact.write_text(report)

    # Loose assertion: arming should be well under a second on any sane host.
    assert ttr_med < 1.0, f"time-to-ready median too high: {ttr_med:.3f}s"
