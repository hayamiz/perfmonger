require 'spec_helper'
require 'tmpdir'

describe '[fingerprint] subcommand' do
  before(:each) do
    @old_pwd = Dir.pwd
    @tmpdir = Dir.mktmpdir
    Dir.chdir(@tmpdir)
  end

  after(:each) do
    Dir.chdir(@old_pwd)
    FileUtils.rm_rf(@tmpdir)
  end

  it 'should create output tgz successfully' do
    run("#{perfmonger_bin} fingerprint output.tgz")
    assert_success(true)
  end
end
