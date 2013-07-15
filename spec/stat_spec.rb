
require 'spec_helper'

describe PerfMonger::Command::StatCommand do
  before(:each) do
    @stat = PerfMonger::Command::StatCommand.new
    @logfile = data_file('test.log')
  end

  describe 'read_logfile method' do
    it 'should return 3 valid records' do
      records = @stat.read_logfile(@logfile)
      records.size.should == 3
      records.each do |record|
        record.should be_a Hash
        record.should include "ioinfo"
        record["ioinfo"].should include "devices"
        record["ioinfo"]["devices"].should be_a Array
        record["ioinfo"]["devices"].should include "sda"
        record["ioinfo"].should include "sda"
        record["ioinfo"]["sda"].should be_a Hash
        record["ioinfo"]["sda"].should include "r/s"
        record["ioinfo"]["sda"].should include "w/s"
        record["ioinfo"]["sda"].should include "rsec/s"
        record["ioinfo"]["sda"].should include "wsec/s"
        record["ioinfo"]["sda"].should include "r_await"
        record["ioinfo"]["sda"].should include "w_await"

        record.should include "cpuinfo"
        record["cpuinfo"].should be_a Hash
        record["cpuinfo"].should include "nr_cpu"
        record["cpuinfo"].should include "cpus"
        record["cpuinfo"].should include "all"
      end
    end
  end
end
