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
    expect(last_command_started).to be_successfully_executed
    expect("iops.pdf").to be_an_existing_file
    expect("transfer.pdf").to be_an_existing_file
    expect("cpu.pdf").to be_an_existing_file
    expect("allcpu.pdf").to be_an_existing_file
  end

  it 'should create PDFs, data and gnuplot files when --save is given' do
    busy100 = data_file "busy100.pgr"

    cmd = "#{perfmonger_bin} plot --save #{busy100}"
    run(cmd, 30)
    expect(last_command_started).to be_successfully_executed

    expect("iops.pdf").to be_an_existing_file
    expect("transfer.pdf").to be_an_existing_file
    expect("cpu.pdf").to be_an_existing_file
    expect("allcpu.pdf").to be_an_existing_file

    expect("io.gp").to be_an_existing_file
    expect("io.dat").to be_an_existing_file
    expect("cpu.gp").to be_an_existing_file
    expect("cpu.dat").to be_an_existing_file
    expect("allcpu.gp").to be_an_existing_file
    expect("allcpu.dat").to be_an_existing_file
  end
end
