"""Tests for the --meta key=value metadata injection (#0064)."""

import json

from conftest import run_perfmonger, requires_proc_diskstats


@requires_proc_diskstats
def test_record_meta_survives_to_play_and_summary(tmp_path):
    """--meta values should be percent-decoded and round-trip into the header,
    surfaced by both `play` (on stderr) and `summary --json`."""
    pgr = tmp_path / "meta.pgr.gz"
    rec = run_perfmonger(
        "record", "--timeout", "1",
        "--meta", "a=b",
        "--meta", "k=%20has%20spaces%3D",
        "-l", str(pgr),
    )
    assert rec.returncode == 0, rec.stderr
    assert pgr.exists()

    # play surfaces metadata on stderr (stdout stays valid JSON records).
    play = run_perfmonger("play", str(pgr))
    assert play.returncode == 0, play.stderr
    assert "a" in play.stderr and "b" in play.stderr
    # The %20 decodes to spaces and %3D decodes to a literal '='.
    assert " has spaces=" in play.stderr

    # summary --json exposes a decoded "meta" object.
    summ = run_perfmonger("summary", "--json", str(pgr))
    assert summ.returncode == 0, summ.stderr
    data = json.loads(summ.stdout)
    assert data["meta"] == {"a": "b", "k": " has spaces="}


@requires_proc_diskstats
def test_record_meta_absent_is_unobtrusive(tmp_path):
    """Without --meta, summary --json should not contain a meta key and play
    should not emit metadata lines."""
    pgr = tmp_path / "nometa.pgr.gz"
    rec = run_perfmonger("record", "--timeout", "1", "-l", str(pgr))
    assert rec.returncode == 0, rec.stderr

    summ = run_perfmonger("summary", "--json", str(pgr))
    assert summ.returncode == 0, summ.stderr
    data = json.loads(summ.stdout)
    assert "meta" not in data

    play = run_perfmonger("play", str(pgr))
    assert play.returncode == 0, play.stderr
    assert "# meta:" not in play.stderr


@requires_proc_diskstats
def test_record_meta_malformed_arg_is_error(tmp_path):
    """An argument without '=' is malformed and must fail with exit code 1
    without creating the output file."""
    pgr = tmp_path / "bad.pgr.gz"
    rec = run_perfmonger(
        "record", "--timeout", "1",
        "--meta", "no-equals",
        "-l", str(pgr),
    )
    assert rec.returncode == 1, rec.stderr
    assert not pgr.exists()
