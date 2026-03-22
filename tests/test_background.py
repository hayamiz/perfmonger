"""Tests for the record --background / --kill / --status options."""

import os
import signal
import time
from pathlib import Path

import pytest
from conftest import run_perfmonger, requires_proc_diskstats


def _session_file():
    """Return the expected session file path."""
    username = os.environ.get("USER") or os.environ.get("LOGNAME") or "unknown"
    return Path("/tmp") / f"perfmonger-{username}-session.pid"


def _cleanup_session():
    """Remove stale session file if present."""
    sf = _session_file()
    if sf.exists():
        try:
            pid = int(sf.read_text().strip())
            os.kill(pid, 0)
            # Process is alive — send SIGINT to stop it
            os.kill(pid, signal.SIGINT)
            time.sleep(0.5)
        except (ValueError, ProcessLookupError, PermissionError):
            pass
        sf.unlink(missing_ok=True)


@pytest.fixture(autouse=True)
def clean_session():
    """Ensure no stale session before/after each test."""
    _cleanup_session()
    yield
    _cleanup_session()


@requires_proc_diskstats
def test_background_creates_logfile_in_cwd(tmp_path):
    """--background should create a logfile in the original working directory, not in /."""
    logfile = tmp_path / "perfmonger.pgr.gz"

    result = run_perfmonger(
        "record", "--background", "--timeout", "2",
        cwd=str(tmp_path),
    )
    # Parent process exits with 0
    assert result.returncode == 0

    # Wait for the background recorder to finish (timeout=2s + margin)
    time.sleep(4)

    assert logfile.exists(), (
        f"Logfile not found at {logfile}. "
        f"Check if daemonize() changed cwd before resolving the output path."
    )
    assert logfile.stat().st_size > 0, "Logfile is empty"


@requires_proc_diskstats
def test_background_creates_logfile_with_explicit_path(tmp_path):
    """--background with -l should write to the specified file."""
    logfile = tmp_path / "custom.pgr"

    result = run_perfmonger(
        "record", "--background", "--timeout", "2",
        "--no-gzip", "-l", str(logfile),
        cwd=str(tmp_path),
    )
    assert result.returncode == 0

    time.sleep(4)

    assert logfile.exists(), f"Logfile not found at {logfile}"
    assert logfile.stat().st_size > 0


@requires_proc_diskstats
def test_background_creates_session_file(tmp_path):
    """--background should create a session PID file in /tmp."""
    result = run_perfmonger(
        "record", "--background", "--timeout", "3",
        cwd=str(tmp_path),
    )
    assert result.returncode == 0

    # Give daemon a moment to start
    time.sleep(0.5)

    sf = _session_file()
    assert sf.exists(), f"Session file not found at {sf}"

    pid = int(sf.read_text().strip())
    assert pid > 0

    # Process should be alive
    os.kill(pid, 0)  # raises if not alive


@requires_proc_diskstats
def test_background_prevents_duplicate_session(tmp_path):
    """Starting a second --background while one is running should fail."""
    # Start first background session
    run_perfmonger(
        "record", "--background", "--timeout", "5",
        cwd=str(tmp_path),
    )
    time.sleep(0.5)

    # Second attempt should fail
    result = run_perfmonger(
        "record", "--background", "--timeout", "2",
        cwd=str(tmp_path),
    )
    assert result.returncode != 0
    assert "already running" in result.stderr.lower()


@requires_proc_diskstats
def test_kill_stops_background_session(tmp_path):
    """--kill should stop a running background session."""
    # Start background session
    run_perfmonger(
        "record", "--background", "--timeout", "30",
        cwd=str(tmp_path),
    )
    time.sleep(0.5)

    sf = _session_file()
    assert sf.exists(), "Session file should exist"
    pid = int(sf.read_text().strip())

    # Kill the session
    result = run_perfmonger("record", "--kill", cwd=str(tmp_path))
    assert result.returncode == 0

    # Process should be gone
    time.sleep(1)
    with pytest.raises(ProcessLookupError):
        os.kill(pid, 0)

    # Session file should be cleaned up
    assert not sf.exists(), "Session file should be removed after --kill"


@requires_proc_diskstats
def test_kill_with_no_session(tmp_path):
    """--kill with no running session should report an error."""
    result = run_perfmonger("record", "--kill", cwd=str(tmp_path))
    assert result.returncode != 0
    assert "no" in result.stderr.lower() or "not" in result.stderr.lower()


@requires_proc_diskstats
def test_status_shows_running_session(tmp_path):
    """--status should display info about a running background session."""
    run_perfmonger(
        "record", "--background", "--timeout", "10",
        cwd=str(tmp_path),
    )
    time.sleep(0.5)

    result = run_perfmonger("record", "--status", cwd=str(tmp_path))
    assert result.returncode == 0

    output = result.stdout + result.stderr
    assert "pid" in output.lower() or "running" in output.lower()


@requires_proc_diskstats
def test_status_with_no_session(tmp_path):
    """--status with no running session should report an error."""
    result = run_perfmonger("record", "--status", cwd=str(tmp_path))
    assert result.returncode != 0
    assert "no" in result.stderr.lower() or "not" in result.stderr.lower()


@requires_proc_diskstats
def test_background_session_file_cleaned_on_normal_exit(tmp_path):
    """Session file should be removed when background recording finishes normally."""
    run_perfmonger(
        "record", "--background", "--timeout", "2",
        cwd=str(tmp_path),
    )
    time.sleep(0.5)

    sf = _session_file()
    assert sf.exists(), "Session file should exist while recording"

    # Wait for timeout to expire
    time.sleep(3)

    assert not sf.exists(), "Session file should be removed after recording completes"
