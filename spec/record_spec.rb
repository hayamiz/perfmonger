require 'spec_helper'

describe '[stat] subcommand' do
  before(:each) do
    skip_if_proc_is_not_available
  end

  it 'should create a valid logfile with no output to stdout' do
    cmd = "#{perfmonger_bin} record --timeout 1"
    run(cmd)
    assert_success(true)
    check_file_presence("perfmonger.pgr") # default file name
    expect(stdout_from(cmd)).to be_empty
  end
end
