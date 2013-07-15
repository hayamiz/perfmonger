
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
        record.should include "time"

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

  describe 'make_summary method' do
    before(:each) do
      @records = @stat.read_logfile(@logfile)
    end

    it 'should return nil for empty records' do
      @stat.make_summary([]).should be_nil
    end

    it 'should return valid format result' do
      summary = @stat.make_summary(@records)

      summary.should be_a Hash
      summary.should include "time"

      summary.should include "ioinfo"
      summary["ioinfo"].should include "devices"
      summary["ioinfo"]["devices"].should be_a Array
      summary["ioinfo"]["devices"].should include "sda"
      summary["ioinfo"].should include "sda"
      summary["ioinfo"]["sda"].should be_a Hash
      summary["ioinfo"]["sda"].should include "r/s"
      summary["ioinfo"]["sda"].should include "w/s"
      summary["ioinfo"]["sda"].should include "rsec/s"
      summary["ioinfo"]["sda"].should include "wsec/s"
      summary["ioinfo"]["sda"].should include "r_await"
      summary["ioinfo"]["sda"].should include "w_await"

      summary.should include "cpuinfo"
      summary["cpuinfo"].should include "nr_cpu"
      summary["cpuinfo"].should include "cpus"
      summary["cpuinfo"].should include "all"
      cpu_entries = [summary["cpuinfo"], *summary["cpus"]]
      cpu_entries.each do |entry|
        entry.should be_a Hash
        entry.should include "%usr"
        entry.should include "%nice"
        entry.should include "%sys"
        entry.should include "%iowait"
        entry.should include "%irq"
        entry.should include "%soft"
        entry.should include "%steal"
        entry.should include "%guest"
        entry.should include "%idle"
      end
    end
  end
end
