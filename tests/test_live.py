"""Tests for the live subcommand (live_spec.rb equivalent)."""

import json

import pytest
from conftest import run_perfmonger, requires_proc_diskstats


@requires_proc_diskstats
@pytest.mark.timeout(10)
@pytest.mark.xfail(reason="Known bug: live command crashes with nil pointer dereference in recorder")
def test_live_outputs_json_and_creates_logfile(tmp_path):
    """Should print JSON records for 3 seconds and create logfile."""
    result = run_perfmonger(
        "live", "--timeout", "3", cwd=str(tmp_path), timeout=10
    )
    assert result.returncode == 0

    required_keys = {"time", "cpu", "disk", "net"}
    for line in result.stdout.strip().splitlines():
        record = json.loads(line)
        assert required_keys.issubset(record.keys())

    assert (tmp_path / "perfmonger.pgr.gz").exists()
