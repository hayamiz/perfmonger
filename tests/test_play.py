"""Tests for the play subcommand (play_spec.rb equivalent)."""

import json
from pathlib import Path
from conftest import run_perfmonger, DATA_DIR


def test_play_outputs_json_records():
    """Should print 3 JSON records with correct keys for busy100.pgr."""
    result = run_perfmonger("play", str(DATA_DIR / "busy100.pgr"))
    assert result.returncode == 0

    lines = result.stdout.strip().splitlines()
    assert len(lines) == 3

    required_keys = {"time", "cpu", "disk", "net"}
    for line in lines:
        record = json.loads(line)
        assert required_keys.issubset(record.keys())


def test_play_plain_file_matches_fixture():
    """Should play plain pgr file and match expected output."""
    result = run_perfmonger("play", str(DATA_DIR / "busy100.pgr"))
    assert result.returncode == 0

    expected = (DATA_DIR / "busy100.pgr.played").read_text()
    assert result.stdout == expected


def test_play_gzipped_file_matches_fixture():
    """Should play gzipped pgr file and match expected output."""
    result = run_perfmonger("play", str(DATA_DIR / "busy100.pgr.gz"))
    assert result.returncode == 0

    expected = (DATA_DIR / "busy100.pgr.played").read_text()
    assert result.stdout == expected
