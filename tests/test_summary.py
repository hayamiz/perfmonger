"""Tests for the summary subcommand (summary_spec.rb equivalent)."""

import json
import os
import re
import subprocess
from pathlib import Path

import pytest
from conftest import run_perfmonger, DATA_DIR, _find_perfmonger_bin


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


# -- Pager-related tests (ticket #0001) --------------------------------------
#
# All of these tests run with stdout captured by pytest, i.e. a pipe (not a
# TTY). Per the documented behavior, the pager must NOT be invoked in that
# case; output should be written directly to stdout. The integration tests
# here therefore assert that the summary output is correct and, when
# applicable, that no pager-related warnings appear on stderr.


def _run_summary_with_env(*extra_args, env_overrides=None):
    """Run `perfmonger summary <extra_args> busy100.pgr` with env overrides."""
    bin_path = _find_perfmonger_bin()
    env = os.environ.copy()
    if env_overrides is not None:
        for k, v in env_overrides.items():
            if v is None:
                env.pop(k, None)
            else:
                env[k] = v
    args = [bin_path, "summary", *extra_args, str(DATA_DIR / "busy100.pgr")]
    return subprocess.run(
        args,
        capture_output=True,
        text=True,
        timeout=60,
        env=env,
    )


def test_summary_piped_stdout_does_not_use_pager():
    """Piped stdout (non-TTY) should write summary directly to stdout."""
    result = _run_summary_with_env(env_overrides={"PAGER": "cat"})
    assert result.returncode == 0
    assert "performance summary" in result.stdout
    assert "Duration:" in result.stdout
    # No pager warnings should appear on stderr (pager path not taken at all).
    assert "Warning: failed to start pager" not in result.stderr


def test_summary_no_pager_flag_disables_pager():
    """--no-pager always writes directly to stdout, regardless of PAGER."""
    result = _run_summary_with_env(
        "--no-pager", env_overrides={"PAGER": "cat"}
    )
    assert result.returncode == 0
    assert "performance summary" in result.stdout
    assert "Duration:" in result.stdout
    assert "Warning:" not in result.stderr


def test_summary_pager_unset_writes_to_stdout():
    """With PAGER unset, output must go directly to stdout."""
    result = _run_summary_with_env(env_overrides={"PAGER": None})
    assert result.returncode == 0
    assert "performance summary" in result.stdout


def test_summary_pager_empty_writes_to_stdout():
    """PAGER="" must be treated the same as PAGER unset: no pager."""
    result = _run_summary_with_env(env_overrides={"PAGER": ""})
    assert result.returncode == 0
    assert "performance summary" in result.stdout
    assert "Warning:" not in result.stderr


def test_summary_pager_nonexistent_with_pipe_still_works():
    """Nonexistent PAGER + piped stdout: pager path isn't attempted, so no
    warning, and summary is written to stdout."""
    result = _run_summary_with_env(
        env_overrides={"PAGER": "/does/not/exist/pager"}
    )
    assert result.returncode == 0
    assert "performance summary" in result.stdout
    # Pager branch is skipped because stdout is not a TTY, so no warning.
    assert "Warning: failed to start pager" not in result.stderr
