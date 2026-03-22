import json
import os
import re
import shutil
import subprocess
from pathlib import Path

import pytest

# Paths
PROJECT_ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = PROJECT_ROOT / "spec" / "data"


def _find_perfmonger_bin():
    """Find the perfmonger binary (Go unified binary)."""
    bin_path = PROJECT_ROOT / "lib" / "exec" / "perfmonger_linux_amd64"
    if bin_path.exists():
        return str(bin_path)
    raise FileNotFoundError(f"perfmonger binary not found at {bin_path}")


@pytest.fixture
def perfmonger_bin():
    """Return path to the perfmonger binary."""
    return _find_perfmonger_bin()


@pytest.fixture
def data_file():
    """Return a function that resolves test data file paths."""
    def _data_file(name):
        p = DATA_DIR / name
        assert p.exists(), f"Test data file not found: {p}"
        return str(p)
    return _data_file


def run_perfmonger(*args, timeout=120, cwd=None):
    """Run perfmonger command and return CompletedProcess.

    Does NOT raise on non-zero exit code (check result.returncode).
    """
    bin_path = _find_perfmonger_bin()
    return subprocess.run(
        [bin_path] + list(args),
        capture_output=True,
        text=True,
        timeout=timeout,
        cwd=cwd,
    )


# Skip condition markers
requires_proc_diskstats = pytest.mark.skipif(
    not Path("/proc/diskstats").exists(),
    reason="/proc/diskstats not available",
)
requires_gnuplot = pytest.mark.skipif(
    shutil.which("gnuplot") is None,
    reason="gnuplot not available",
)
