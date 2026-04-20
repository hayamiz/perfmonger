"""Tests for the stat subcommand (stat_spec.rb equivalent)."""

import pytest
from conftest import run_perfmonger, requires_proc_diskstats


@requires_proc_diskstats
def test_stat_creates_logfile(tmp_path):
    """Should run command and create logfile."""
    result = run_perfmonger(
        "stat", "--", "sleep", "1", cwd=str(tmp_path)
    )
    assert result.returncode == 0
    assert (tmp_path / "perfmonger.pgr").exists()


@requires_proc_diskstats
def test_stat_shows_summary(tmp_path):
    """Should show performance summary."""
    result = run_perfmonger(
        "stat", "--", "sleep", "1", cwd=str(tmp_path)
    )
    assert result.returncode == 0
    output = result.stdout + result.stderr
    assert "Performance Summary" in output
    assert "CPU usage" in output
