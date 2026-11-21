"""Tests for the system-update subcommand.

These tests are network-free: they only cover the help output and the refusal
to update a development build. The download/verify/replace flow is covered by
the Go unit tests in core/cmd/perfmonger/systemupdate_test.go, which run
against a local httptest server.
"""

import hashlib
import os
import subprocess

import pytest
from conftest import run_perfmonger


def _version_output(bin_path):
    result = subprocess.run(
        [bin_path, "--version"], capture_output=True, text=True, timeout=30
    )
    return result.stdout.strip()


def _file_state(path):
    st = os.stat(path)
    with open(path, "rb") as f:
        digest = hashlib.sha256(f.read()).hexdigest()
    return digest, st.st_mode, st.st_mtime_ns, st.st_ino


@pytest.mark.timeout(30)
def test_system_update_help():
    result = run_perfmonger("system-update", "--help", timeout=20)
    assert result.returncode == 0
    assert "Usage:" in result.stdout
    assert "system-update" in result.stdout
    assert "--check" in result.stdout
    assert "--yes" in result.stdout
    assert "-y" in result.stdout


@pytest.mark.timeout(30)
def test_system_update_listed_in_root_help():
    result = run_perfmonger("--help", timeout=20)
    assert result.returncode == 0
    assert "system-update" in result.stdout


@pytest.mark.timeout(30)
@pytest.mark.parametrize("args", [[], ["--check"], ["--yes"]])
def test_system_update_refuses_dev_build(perfmonger_bin, args):
    """A dev build (make build) must refuse to update itself, before any
    network access, and leave the binary untouched."""
    if _version_output(perfmonger_bin) != "PerfMonger version dev":
        pytest.skip("the binary under test is not a dev build")

    before = _file_state(perfmonger_bin)
    result = subprocess.run(
        [perfmonger_bin, "system-update", *args],
        stdin=subprocess.DEVNULL,
        capture_output=True,
        text=True,
        timeout=20,
    )
    after = _file_state(perfmonger_bin)

    assert result.returncode == 1
    assert "dev" in result.stderr
    assert "install a release build manually" in result.stderr
    assert "Usage:" not in result.stderr
    assert before == after

    exe_dir = os.path.dirname(os.path.realpath(perfmonger_bin))
    leftovers = [
        name for name in os.listdir(exe_dir) if name.startswith(".perfmonger-update-")
    ]
    assert leftovers == []
