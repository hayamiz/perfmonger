require 'spec_helper'

describe '[record] subcommand' do
  before(:each) do
    skip_if_proc_is_not_available
  end

  it 'should create a valid logfile with no output to stdout' do
    cmd = "#{perfmonger_bin} record --timeout 1"
    run(cmd)
    expect(last_command_started).to be_successfully_executed
    expect("perfmonger.pgr.gz").to be_an_existing_file # default file name
    expect(last_command_started.stdout).to be_empty
  end

  it 'should create a non-gzipped logfile' do
    cmd = "#{perfmonger_bin} record --timeout 1 --no-gzip"
    run(cmd)
    expect(last_command_started).to be_successfully_executed
    expect("perfmonger.pgr").to be_an_existing_file # default file name
    expect(last_command_started.stdout).to be_empty
  end

end
