
require 'spec_helper'

describe "PerfmongerCommand" do
  before(:each) do
    @perfmonger_command = File.expand_path('../../bin/perfmonger', __FILE__)
  end

  it "should be an executable" do
    expect(File.executable?(@perfmonger_command)).to be true
  end

  it 'should print version number if --version specified' do
    expect(`#{@perfmonger_command} --version`).to include(PerfMonger::VERSION)
  end

  describe 'stat subcommand' do
    it 'should print "Execution time: XXX.XXX"' do
      if File.exists?("/proc/diskstats")
        expect(`#{@perfmonger_command} stat -- sleep 1`).to match(/^Execution time: (\d+)\.(\d+)$/)
      else
        # do nothing
        expect(true).to eq true
      end
    end
  end

  describe 'summary subcommand' do
    it 'should print expected output with 2devices.log' do
      File.open(data_file('2devices.output'), "w") do |f|
        f.print(`#{@perfmonger_command} summary #{data_file('2devices.log')}`)
      end

      expect(system("diff -u #{data_file('2devices.expected')} #{data_file('2devices.output')}")).to be true
    end
  end
end
