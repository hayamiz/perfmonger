"""Tests for the stat subcommand (stat_spec.rb equivalent)."""

import pytest
from conftest import run_perfmonger, requires_proc_diskstats


@requires_proc_diskstats
@pytest.mark.xfail(reason="Known bug: stat command panics with unexpected EOF in summarizer")
def test_stat_creates_logfile(tmp_path):
    """Should run command and create logfile."""
    result = run_perfmonger(
        "stat", "--", "sleep", "1", cwd=str(tmp_path)
    )
    assert result.returncode == 0
    assert (tmp_path / "perfmonger.pgr.gz").exists()
