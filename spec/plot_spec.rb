require 'spec_helper'

# TODO: examples for options

RSpec.describe '[plot] subcommand' do
  let(:busy100_disk_dat) {
    File.read(data_file "busy100.pgr.plot-formatted.disk.dat")
  }
  let(:busy100_cpu_dat) {
    File.read(data_file "busy100.pgr.plot-formatted.cpu.dat")
  }


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
    expect("disk-iops.pdf").to be_an_existing_file
    expect("disk-transfer.pdf").to be_an_existing_file
    expect("cpu.pdf").to be_an_existing_file
    expect("allcpu.pdf").to be_an_existing_file
  end

  it 'should create PDFs, data and gnuplot files when --save is given' do
    busy100 = data_file "busy100.pgr"

    cmd = "#{perfmonger_bin} plot --save #{busy100}"
    run(cmd, 30)
    expect(last_command_started).to be_successfully_executed

    expect("disk-iops.pdf").to be_an_existing_file
    expect("disk-transfer.pdf").to be_an_existing_file
    expect("cpu.pdf").to be_an_existing_file
    expect("allcpu.pdf").to be_an_existing_file

    expect("disk.gp").to be_an_existing_file
    expect("disk.dat").to be_an_existing_file
    expect("cpu.gp").to be_an_existing_file
    expect("cpu.dat").to be_an_existing_file
    expect("allcpu.gp").to be_an_existing_file

    # cpu.dat content check
    expect("disk.dat").to have_file_content busy100_disk_dat
    expect("cpu.dat").to have_file_content busy100_cpu_dat
  end

  it 'should work with gzipped perfmonger logfile' do
    busy100 = data_file "busy100.pgr.gz"

    cmd = "#{perfmonger_bin} plot --save #{busy100}"
    run(cmd, 30)
    expect(last_command_started).to be_successfully_executed

    expect("disk-iops.pdf").to be_an_existing_file
    expect("disk-transfer.pdf").to be_an_existing_file
    expect("cpu.pdf").to be_an_existing_file
    expect("allcpu.pdf").to be_an_existing_file

    expect("disk.gp").to be_an_existing_file
    expect("disk.dat").to be_an_existing_file
    expect("cpu.gp").to be_an_existing_file
    expect("cpu.dat").to be_an_existing_file
    expect("allcpu.gp").to be_an_existing_file

    # cpu.dat content check
    expect("disk.dat").to have_file_content busy100_disk_dat
    expect("cpu.dat").to have_file_content busy100_cpu_dat
  end

  it "should work with --disk-only option" do
    busy100 = data_file "busy100.pgr.gz"

    cmd = "#{perfmonger_bin} plot --save #{busy100} --disk-only sda1"

    run(cmd, 30)

    expect(last_command_started).to be_successfully_executed

    disk_dat = File.expand_path("disk.dat", last_command_started.working_directory)
    total_write_iops = `cat #{disk_dat}|grep total -A 2|tail -n1|awk '{print $3}'`.to_f

    expect(total_write_iops).to be_within(1.67).of(0.01)
  end
end
