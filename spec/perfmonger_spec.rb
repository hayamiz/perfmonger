
require 'spec_helper'

describe "perfmonger command" do
  it "should be an executable" do
    expect(File.executable?(perfmonger_bin)).to be true
  end

  it 'should print help and exit with failure when no arguments given' do
    cmd = "#{perfmonger_bin}"
    run(cmd)
    expect(last_command_started).not_to be_successfully_executed
    expect(last_command_started.stdout).to match(/^Usage: perfmonger/)
  end

  it 'should print help and exit with success when --help is given' do
    ["-h", "--help"].each do |arg|
      cmd = "#{perfmonger_bin} #{arg}"
      run(cmd)
      expect(last_command_started).to be_successfully_executed
      expect(last_command_started.stdout).to match(/^Usage: perfmonger/)
    end
  end

  it 'should print version number if --version given' do
    cmd = "#{perfmonger_bin} --version"
    run(cmd)
    expect(last_command_started).to be_successfully_executed
    expect(last_command_started.stdout).to include(PerfMonger::VERSION)
  end

  it 'fails if unknown subcommand given' do
    cmd = "#{perfmonger_bin} piyo"
    run(cmd)
    expect(last_command_started).not_to be_successfully_executed
  end
end
