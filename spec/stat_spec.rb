
require 'spec_helper'

describe '[stat] subcommand' do
  before(:each) do
    skip_if_proc_is_not_available
  end

  it 'should print "Execution time: XXX.XXX"' do
    cmd = "#{perfmonger_bin} stat -- sleep 1"
    run(cmd)
    expect(last_command_started).to be_successfully_executed
    expect("perfmonger.pgr").to be_an_existing_file
  end
end
