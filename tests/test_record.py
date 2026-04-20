"""Tests for the record subcommand (record_spec.rb equivalent)."""

from pathlib import Path
from conftest import run_perfmonger, requires_proc_diskstats


@requires_proc_diskstats
def test_record_creates_gzipped_logfile(tmp_path):
    """Should create a gzipped logfile."""
    result = run_perfmonger("record", "--timeout", "1", cwd=str(tmp_path))
    assert result.returncode == 0
    assert (tmp_path / "perfmonger.pgr.gz").exists()


@requires_proc_diskstats
def test_record_creates_plain_logfile(tmp_path):
    """Should create a non-gzipped logfile with --no-gzip."""
    result = run_perfmonger(
        "record", "--timeout", "1", "--no-gzip",
        "-l", "perfmonger.pgr",
        cwd=str(tmp_path),
    )
    assert result.returncode == 0
    assert (tmp_path / "perfmonger.pgr").exists()


@requires_proc_diskstats
def test_record_no_gzip_default_filename(tmp_path):
    """--no-gzip without -l should default to perfmonger.pgr (not .pgr.gz)."""
    result = run_perfmonger(
        "record", "--timeout", "1", "--no-gzip",
        cwd=str(tmp_path),
    )
    assert result.returncode == 0
    assert (tmp_path / "perfmonger.pgr").exists(), (
        "Expected perfmonger.pgr but got: "
        + ", ".join(p.name for p in tmp_path.iterdir())
    )
    assert not (tmp_path / "perfmonger.pgr.gz").exists(), (
        "--no-gzip should not create .pgr.gz file"
    )
