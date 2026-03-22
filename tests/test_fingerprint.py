"""Tests for the fingerprint subcommand (fingerprint_spec.rb equivalent)."""

import subprocess

import pytest
from conftest import run_perfmonger


@pytest.mark.timeout(30)
def test_fingerprint_creates_tgz(tmp_path):
    """Should create output tgz successfully."""
    result = run_perfmonger(
        "fingerprint", "output.tgz", cwd=str(tmp_path), timeout=20
    )
    assert result.returncode == 0
    assert (tmp_path / "output.tgz").exists()


@pytest.mark.timeout(30)
def test_fingerprint_alias_fp(tmp_path):
    """Should create output tgz successfully with alias invocation."""
    result = run_perfmonger("fp", "output.tgz", cwd=str(tmp_path), timeout=20)
    assert result.returncode == 0
    assert (tmp_path / "output.tgz").exists()


@pytest.mark.timeout(30)
def test_fingerprint_archive_content(tmp_path):
    """Should create output tgz with extractable content."""
    result = run_perfmonger(
        "fingerprint", "output.tgz", cwd=str(tmp_path), timeout=20
    )
    assert result.returncode == 0

    # Extract the archive
    extract = subprocess.run(
        ["tar", "xf", "output.tgz"],
        capture_output=True,
        text=True,
        cwd=str(tmp_path),
    )
    assert extract.returncode == 0
    assert (tmp_path / "output").is_dir()
