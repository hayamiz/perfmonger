"""Tests for the plot subcommand (plot_spec.rb equivalent)."""

import subprocess
from pathlib import Path

import pytest
from conftest import run_perfmonger, requires_gnuplot, DATA_DIR, PROJECT_ROOT

# plot command internally invokes perfmonger-core (plot-formatter subcommand)
_perfmonger_core_missing = not (
    PROJECT_ROOT / "lib" / "exec" / "perfmonger-core_linux_amd64"
).exists()
requires_perfmonger_core = pytest.mark.xfail(
    _perfmonger_core_missing,
    reason="perfmonger-core_linux_amd64 not built (required by plot subcommand)",
)


@requires_perfmonger_core
@requires_gnuplot
@pytest.mark.timeout(30)
def test_plot_creates_pdfs(tmp_path):
    """Should create PDF files for busy100.pgr."""
    result = run_perfmonger(
        "plot", str(DATA_DIR / "busy100.pgr"), cwd=str(tmp_path), timeout=30
    )
    assert result.returncode == 0
    for name in ["disk-iops.pdf", "disk-transfer.pdf", "cpu.pdf", "allcpu.pdf"]:
        assert (tmp_path / name).exists(), f"{name} not found"


@requires_perfmonger_core
@requires_gnuplot
@pytest.mark.timeout(30)
def test_plot_save_creates_files_and_matches(tmp_path):
    """Should create PDFs, data and gnuplot files when --save is given."""
    result = run_perfmonger(
        "plot", "--save", str(DATA_DIR / "busy100.pgr"), cwd=str(tmp_path), timeout=30
    )
    assert result.returncode == 0

    # PDF files
    for name in ["disk-iops.pdf", "disk-transfer.pdf", "cpu.pdf", "allcpu.pdf"]:
        assert (tmp_path / name).exists(), f"{name} not found"

    # Gnuplot and data files
    for name in ["disk.gp", "disk.dat", "cpu.gp", "cpu.dat", "allcpu.gp"]:
        assert (tmp_path / name).exists(), f"{name} not found"

    # Content comparison
    expected_disk = (DATA_DIR / "busy100.pgr.plot-formatted.disk.dat").read_text()
    assert (tmp_path / "disk.dat").read_text() == expected_disk

    expected_cpu = (DATA_DIR / "busy100.pgr.plot-formatted.cpu.dat").read_text()
    assert (tmp_path / "cpu.dat").read_text() == expected_cpu


@requires_perfmonger_core
@requires_gnuplot
@pytest.mark.timeout(30)
def test_plot_gzipped_input(tmp_path):
    """Should work with gzipped perfmonger logfile."""
    result = run_perfmonger(
        "plot", "--save", str(DATA_DIR / "busy100.pgr.gz"),
        cwd=str(tmp_path), timeout=30,
    )
    assert result.returncode == 0

    for name in ["disk-iops.pdf", "disk-transfer.pdf", "cpu.pdf", "allcpu.pdf"]:
        assert (tmp_path / name).exists(), f"{name} not found"

    for name in ["disk.gp", "disk.dat", "cpu.gp", "cpu.dat", "allcpu.gp"]:
        assert (tmp_path / name).exists(), f"{name} not found"

    expected_disk = (DATA_DIR / "busy100.pgr.plot-formatted.disk.dat").read_text()
    assert (tmp_path / "disk.dat").read_text() == expected_disk

    expected_cpu = (DATA_DIR / "busy100.pgr.plot-formatted.cpu.dat").read_text()
    assert (tmp_path / "cpu.dat").read_text() == expected_cpu


@requires_perfmonger_core
@requires_gnuplot
@pytest.mark.timeout(30)
def test_plot_disk_only_option(tmp_path):
    """Should work with --disk-only option."""
    result = run_perfmonger(
        "plot", "--save", str(DATA_DIR / "busy100.pgr.gz"), "--disk-only", "sda1",
        cwd=str(tmp_path), timeout=30,
    )
    assert result.returncode == 0

    # Extract total write IOPS from disk.dat
    disk_dat = (tmp_path / "disk.dat").read_text()
    total_lines = [line for line in disk_dat.splitlines() if "total" in line]
    assert total_lines, "No 'total' line found in disk.dat"
    # Get the last total line's 3rd field (write IOPS)
    total_write_iops = float(total_lines[-1].split()[2])
    assert total_write_iops == pytest.approx(0.01, abs=1.67)
