require 'spec_helper'

# TODO: examples for options

describe '[plot] subcommand' do
  before(:each) do
    if ! system("type gnuplot >/dev/null 2>&1")
      skip "gnuplot is not available"
    end
  end

  it 'should create PDFs for busy100.pgr' do
    busy100 = data_file "busy100.pgr"

    cmd = "#{perfmonger_bin} plot #{busy100}"
    run(cmd, 30)
    assert_success(true)
    check_file_presence("iops.pdf")
    check_file_presence("transfer.pdf")
    check_file_presence("cpu.pdf")
    check_file_presence("allcpu.pdf")
  end

  it 'should create PDFs, data and gnuplot files when --save is given' do
    busy100 = data_file "busy100.pgr"

    cmd = "#{perfmonger_bin} plot --save #{busy100}"
    run(cmd, 30)
    assert_success(true)
    check_file_presence("iops.pdf")
    check_file_presence("transfer.pdf")
    check_file_presence("cpu.pdf")
    check_file_presence("allcpu.pdf")

    check_file_presence("io.gp")
    check_file_presence("io.dat")
    check_file_presence("cpu.gp")
    check_file_presence("cpu.dat")
    check_file_presence("allcpu.gp")
    check_file_presence("allcpu.dat")
  end
end
