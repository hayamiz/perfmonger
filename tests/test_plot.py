"""Tests for the plot subcommand (plot_spec.rb equivalent)."""

from pathlib import Path

import pytest
from conftest import run_perfmonger, requires_gnuplot, DATA_DIR


@requires_gnuplot
@pytest.mark.timeout(30)
def test_plot_creates_pdfs(tmp_path):
    """Should create PDF files for busy100.pgr."""
    result = run_perfmonger(
        "plot", str(DATA_DIR / "busy100.pgr"), "-o", str(tmp_path),
        cwd=str(tmp_path), timeout=30,
    )
    assert result.returncode == 0
    for name in ["disk-iops.pdf", "disk-transfer.pdf", "cpu.pdf", "allcpu.pdf"]:
        assert (tmp_path / name).exists(), f"{name} not found"


@requires_gnuplot
@pytest.mark.timeout(30)
def test_plot_save_creates_files_and_matches(tmp_path):
    """Should create PDFs, data and gnuplot files when --save is given."""
    result = run_perfmonger(
        "plot", "--save", str(DATA_DIR / "busy100.pgr"), "-o", str(tmp_path),
        cwd=str(tmp_path), timeout=30,
    )
    assert result.returncode == 0

    # PDF files
    for name in ["disk-iops.pdf", "disk-transfer.pdf", "cpu.pdf", "allcpu.pdf"]:
        assert (tmp_path / name).exists(), f"{name} not found"

    # Gnuplot and data files
    for name in ["disk-iops.gp", "disk-transfer.gp", "disk.dat", "cpu.gp", "cpu.dat", "allcpu.gp"]:
        assert (tmp_path / name).exists(), f"{name} not found"

    # Content comparison with golden fixtures
    expected_disk = (DATA_DIR / "busy100.pgr.plot-formatted.disk.dat").read_text()
    assert (tmp_path / "disk.dat").read_text() == expected_disk

    expected_cpu = (DATA_DIR / "busy100.pgr.plot-formatted.cpu.dat").read_text()
    assert (tmp_path / "cpu.dat").read_text() == expected_cpu


@requires_gnuplot
@pytest.mark.timeout(30)
def test_plot_gzipped_input(tmp_path):
    """Should work with gzipped perfmonger logfile."""
    result = run_perfmonger(
        "plot", "--save", str(DATA_DIR / "busy100.pgr.gz"), "-o", str(tmp_path),
        cwd=str(tmp_path), timeout=30,
    )
    assert result.returncode == 0

    for name in ["disk-iops.pdf", "disk-transfer.pdf", "cpu.pdf", "allcpu.pdf"]:
        assert (tmp_path / name).exists(), f"{name} not found"

    for name in ["disk-iops.gp", "disk-transfer.gp", "disk.dat", "cpu.gp", "cpu.dat", "allcpu.gp"]:
        assert (tmp_path / name).exists(), f"{name} not found"

    expected_disk = (DATA_DIR / "busy100.pgr.plot-formatted.disk.dat").read_text()
    assert (tmp_path / "disk.dat").read_text() == expected_disk

    expected_cpu = (DATA_DIR / "busy100.pgr.plot-formatted.cpu.dat").read_text()
    assert (tmp_path / "cpu.dat").read_text() == expected_cpu


@requires_gnuplot
@pytest.mark.timeout(30)
def test_plot_disk_only_option(tmp_path):
    """Should work with --disk-only option."""
    result = run_perfmonger(
        "plot", "--save", str(DATA_DIR / "busy100.pgr.gz"), "--disk-only", "sda1",
        "-o", str(tmp_path),
        cwd=str(tmp_path), timeout=30,
    )
    assert result.returncode == 0

    # Extract total write IOPS from disk.dat
    # Data lines in the "total" section follow the "# device: total" header
    disk_dat = (tmp_path / "disk.dat").read_text()
    lines = disk_dat.splitlines()
    in_total = False
    total_data_lines = []
    for line in lines:
        if "# device: total" in line:
            in_total = True
            continue
        if in_total and line.startswith("# device:"):
            break
        if in_total and line.strip() and not line.startswith("#"):
            total_data_lines.append(line)
    assert total_data_lines, "No data lines in 'total' section of disk.dat"
    # Column 3 is write IOPS
    total_write_iops = float(total_data_lines[0].split()[2])
    assert total_write_iops == pytest.approx(0.01, abs=1.67)
