
require 'spec_helper'

describe "perfmonger command" do
  it "should be an executable" do
    expect(File.executable?(perfmonger_bin)).to be true
  end

  it 'should print help and exit with failure when no arguments given' do
    cmd = "#{perfmonger_bin}"
    run(cmd)
    assert_success(false)
    expect(stdout_from(cmd)).to match(/^Usage: perfmonger/)
  end

  it 'should print help and exit with success when --help is given' do
    ["-h", "--help"].each do |arg|
      cmd = "#{perfmonger_bin} #{arg}"
      run(cmd)
      assert_success(true)
      expect(stdout_from(cmd)).to match(/^Usage: perfmonger/)
    end
  end

  it 'should print version number if --version given' do
    cmd = "#{perfmonger_bin} --version"
    run(cmd)
    assert_success(true)
    expect(stdout_from(cmd)).to include(PerfMonger::VERSION)
  end
end
