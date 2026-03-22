"""Tests for the summary subcommand (summary_spec.rb equivalent)."""

import json
import re
from pathlib import Path

import pytest
from conftest import run_perfmonger, DATA_DIR


def _strip_file_path_line(text):
    """Remove '== performance summary of ...' line from output."""
    return re.sub(r"^== performance summary of .*$", "", text, flags=re.MULTILINE)


def test_summary_text_output():
    """Should print valid text output with correct numeric values."""
    result = run_perfmonger("summary", str(DATA_DIR / "busy100.pgr"))
    assert result.returncode == 0
    output = result.stdout

    # Measurement duration
    assert re.search(r"^Duration: (\d+\.\d+) sec$", output, re.MULTILINE)

    # CPU usage patterns
    non_idle_match = re.search(r"Non-idle usage: (\d+\.\d+) %$", output, re.MULTILINE)
    usr_match = re.search(r"%usr: (\d+\.\d+) %$", output, re.MULTILINE)
    idle_match = re.search(r"Idle usage: (\d+\.\d+) %$", output, re.MULTILINE)

    assert non_idle_match
    assert usr_match
    assert idle_match

    non_idle_usage = float(non_idle_match.group(1))
    usr_usage = float(usr_match.group(1))
    idle_usage = float(idle_match.group(1))

    assert non_idle_usage == pytest.approx(100.0, abs=1.0)
    assert usr_usage == pytest.approx(100.0, abs=1.0)
    assert idle_usage == pytest.approx(99.0, abs=1.0)
    assert non_idle_usage + idle_usage == pytest.approx(200.0, abs=0.1)

    # Disk usage
    assert re.search(r"^\* Average DEVICE usage: .+$", output, re.MULTILINE)


def test_summary_json_output():
    """Should print valid JSON with correct keys when --json is given."""
    result = run_perfmonger("summary", "--json", str(DATA_DIR / "busy100.pgr"))
    assert result.returncode == 0

    data = json.loads(result.stdout)
    required_keys = {"cpu", "disk", "net", "exectime"}
    assert required_keys.issubset(data.keys())


def test_summary_gzipped_text_matches_fixture():
    """Should produce same text output for gzipped input."""
    result = run_perfmonger("summary", str(DATA_DIR / "busy100.pgr.gz"))
    assert result.returncode == 0

    expected = (DATA_DIR / "busy100.pgr.summary").read_text()
    assert _strip_file_path_line(result.stdout) == _strip_file_path_line(expected)


def test_summary_gzipped_json_matches_fixture():
    """Should produce same JSON output for gzipped input."""
    result = run_perfmonger("summary", "--json", str(DATA_DIR / "busy100.pgr.gz"))
    assert result.returncode == 0

    expected = (DATA_DIR / "busy100.pgr.summary.json").read_text()
    assert result.stdout == expected
