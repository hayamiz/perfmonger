
require 'spec_helper'

describe '[stat] subcommand' do
  before(:each) do
    skip_if_proc_is_not_available
    @old_pwd = Dir.pwd
    @tmpdir = Dir.mktmpdir
    Dir.chdir(@tmpdir)
  end

  after(:each) do
    Dir.chdir(@old_pwd)
    FileUtils.rm_rf(@tmpdir)
  end

  it 'should print "Execution time: XXX.XXX"' do
    expect(`#{perfmonger_bin} stat -- sleep 1`).to match(/^Execution time: (\d+)\.(\d+)$/)
  end
end
