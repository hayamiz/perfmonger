"""Tests for per-core CPU frequency recording (ticket #0067)."""

import glob
import json
import platform
import re

import pytest
from conftest import run_perfmonger, DATA_DIR


def _kernel_reads_are_cheap():
    """Mirror the recorder's gate: x86 kernels < 5.19 IPI on every read."""
    if platform.machine() not in ("x86_64", "i386", "i686"):
        return True
    m = re.match(r"(\d+)\.(\d+)", platform.release())
    if not m:
        return True
    return (int(m.group(1)), int(m.group(2))) >= (5, 19)


def _cpufreq_readable():
    for path in glob.glob("/sys/devices/system/cpu/cpu[0-9]*/cpufreq/scaling_cur_freq"):
        try:
            with open(path) as f:
                if int(f.read().strip()) > 0:
                    return True
        except (OSError, ValueError):
            continue
    return False


CPUFREQ_RECORDED = _cpufreq_readable() and _kernel_reads_are_cheap()

requires_cpufreq = pytest.mark.skipif(
    not CPUFREQ_RECORDED,
    reason="cpufreq not readable or x86 kernel < 5.19 on this host",
)


def _record(tmp_path, *extra):
    logfile = tmp_path / "out.pgr"
    result = run_perfmonger(
        "record", "--timeout", "1", "--interval", "0.2", "--no-gzip",
        "-l", str(logfile), *extra,
        cwd=str(tmp_path),
    )
    assert result.returncode == 0, result.stderr
    assert logfile.exists()
    return logfile, result


def _play(logfile):
    result = run_perfmonger("play", str(logfile))
    assert result.returncode == 0, result.stderr
    records = [json.loads(line) for line in result.stdout.strip().splitlines()]
    assert records, "play produced no records"
    return records


@requires_cpufreq
def test_record_includes_cpufreq_when_available(tmp_path):
    logfile, _ = _record(tmp_path)
    for rec in _play(logfile):
        assert "cpufreq" in rec
        freq = rec["cpufreq"]
        assert freq["num_core"] == len(freq["cores"])
        assert freq["num_core"] >= 1
        assert all(isinstance(v, (int, float)) and v >= 0 for v in freq["cores"])
        assert any(v > 0 for v in freq["cores"])
        # cpufreq sits next to the CPU usage section and shares its core count.
        assert freq["num_core"] == rec["cpu"]["num_core"]


def test_record_no_cpufreq_flag_omits_frequencies(tmp_path):
    logfile, _ = _record(tmp_path, "--no-cpufreq")
    for rec in _play(logfile):
        assert "cpufreq" not in rec
        assert "cpu" in rec


@pytest.mark.skipif(CPUFREQ_RECORDED, reason="cpufreq is recorded on this host")
def test_record_without_cpufreq_support_keeps_cpu(tmp_path):
    """Hosts without usable cpufreq silently fall back to the existing metrics."""
    logfile, _ = _record(tmp_path)
    for rec in _play(logfile):
        assert "cpufreq" not in rec
        assert "cpu" in rec


def test_play_legacy_file_has_no_cpufreq():
    """Recordings made before cpufreq support still play without the key."""
    for rec in _play(DATA_DIR / "busy100.pgr"):
        assert "cpufreq" not in rec


@requires_cpufreq
def test_summary_reports_average_cpufreq(tmp_path):
    logfile, _ = _record(tmp_path)

    result = run_perfmonger("summary", "--json", str(logfile))
    assert result.returncode == 0, result.stderr
    summary = json.loads(result.stdout)
    assert "cpufreq" in summary
    freq = summary["cpufreq"]
    assert freq["num_core"] == len(freq["cores"])
    assert freq["avg"] > 0

    result = run_perfmonger("summary", str(logfile))
    assert result.returncode == 0, result.stderr
    assert re.search(r"^\* Average CPU frequency: \d+\.\d MHz$", result.stdout, re.MULTILINE)


def test_summary_legacy_file_has_no_cpufreq():
    result = run_perfmonger("summary", "--json", str(DATA_DIR / "busy100.pgr"))
    assert result.returncode == 0, result.stderr
    assert "cpufreq" not in json.loads(result.stdout)
