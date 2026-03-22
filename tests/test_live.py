"""Tests for the live subcommand (live_spec.rb equivalent)."""

import json

import pytest
from conftest import run_perfmonger, requires_proc_diskstats


@requires_proc_diskstats
@pytest.mark.timeout(10)
def test_live_outputs_json(tmp_path):
    """Should print JSON records to stdout."""
    result = run_perfmonger(
        "live", "--timeout", "2", cwd=str(tmp_path), timeout=10
    )
    assert result.returncode == 0

    lines = result.stdout.strip().splitlines()
    assert len(lines) >= 1, "Expected at least one JSON record"

    # Default is --no-net, so 'net' key is not present
    required_keys = {"time", "cpu", "disk"}
    for line in lines:
        record = json.loads(line)
        assert required_keys.issubset(record.keys()), (
            f"Missing keys: {required_keys - record.keys()}"
        )


@requires_proc_diskstats
@pytest.mark.timeout(10)
def test_live_color_and_pretty(tmp_path):
    """--color and --pretty options should not cause errors."""
    result = run_perfmonger(
        "live", "--timeout", "2", "--color", "--pretty",
        cwd=str(tmp_path), timeout=10,
    )
    assert result.returncode == 0
    lines = result.stdout.strip().splitlines()
    assert len(lines) >= 1
