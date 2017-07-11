require 'spec_helper'
require 'tmpdir'

RSpec.describe '[fingerprint] subcommand' do
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
    run("#{perfmonger_bin} fingerprint output.tgz", 20)
    expect(last_command_started).to be_successfully_executed
    expect("output.tgz").to be_an_existing_file
  end

  it 'should create output tgz successfully with alias invocation' do
    run("#{perfmonger_bin} fp output.tgz", 20)
    expect(last_command_started).to be_successfully_executed
    expect("output.tgz").to be_an_existing_file
  end

  it "should create output tgz successfully with content" do
    run("#{perfmonger_bin} fingerprint output.tgz", 20)
    expect(last_command_started).to be_successfully_executed
    run("tar xf output.tgz")
    expect(last_command_started).to be_successfully_executed
    expect("output").to be_an_existing_directory
  end
end
