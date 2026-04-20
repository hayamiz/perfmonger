"""Tests for the top-level perfmonger CLI (perfmonger_spec.rb equivalent)."""

import os
import pytest
from conftest import run_perfmonger, _find_perfmonger_bin


def test_binary_is_executable():
    bin_path = _find_perfmonger_bin()
    assert os.access(bin_path, os.X_OK)


def test_no_args_prints_help():
    result = run_perfmonger()
    # cobra prints help and exits 0 when no args given
    assert result.returncode == 0
    assert "Usage:" in result.stdout


@pytest.mark.parametrize("flag", ["-h", "--help"])
def test_help_flag(flag):
    result = run_perfmonger(flag)
    assert result.returncode == 0
    assert "Usage:" in result.stdout


def test_version_flag():
    result = run_perfmonger("--version")
    assert result.returncode == 0
    # Version string should be present in output
    assert result.stdout.strip() != ""


def test_unknown_subcommand_fails():
    result = run_perfmonger("piyo")
    assert result.returncode != 0
