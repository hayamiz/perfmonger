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
    check_file_presence("output.tgz")
  end

  it 'should create output tgz successfully with alias invocation' do
    run("#{perfmonger_bin} fp output.tgz")
    assert_success(true)
    check_file_presence("output.tgz")
  end

  it "should create output tgz successfully with content" do
    run("#{perfmonger_bin} fingerprint output.tgz")
    assert_success(true)
    run("tar xf output.tgz")
    assert_success(true)
    check_directory_presence(["output"], true)
    check_file_presence(%w{output/dmidecode.log})
  end
end
