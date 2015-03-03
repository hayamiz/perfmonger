
require 'spec_helper'

describe '[stat] subcommand' do
  before(:each) do
    skip_if_proc_is_not_available
  end

  it 'should print "Execution time: XXX.XXX"' do
    cmd = "#{perfmonger_bin} stat -- sleep 1"
    run(cmd)
    assert_success(true)
    check_file_presence("perfmonger.pgr")
  end
end
