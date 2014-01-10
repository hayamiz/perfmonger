
require 'spec_helper'

describe PerfMonger::Command::SummaryCommand do
  before(:each) do
    @summary = PerfMonger::Command::SummaryCommand.new
    @logfile = data_file('test.log')
    @logfile_2devices = data_file('2devices.log')
  end

  describe 'read_logfile method' do
    it 'should return 3 valid records' do
      records = @summary.read_logfile(@logfile)
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
      @records = @summary.read_logfile(@logfile)
    end

    it 'should return nil for empty records' do
      @summary.make_summary([]).should be_nil
    end

    it 'should return valid format result' do
      summary = @summary.make_summary(@records)

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
      cpu_entries = [summary["cpuinfo"]["all"], *summary["cpuinfo"]["cpus"]]
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

    it 'should calculate avg. IOPS correctly against non-equal intervals' do
      @records[0]["time"] = 0.0
      @records[1]["time"] = 1.0
      @records[2]["time"] = 3.0

      @records[0]["ioinfo"]["sda"]["r/s"] = 0.0 # no effect for result
      @records[1]["ioinfo"]["sda"]["r/s"] = 3.0
      @records[2]["ioinfo"]["sda"]["r/s"] = 6.0

      # avg. r/s should be ((3.0 * 1.0 + 6.0 * 3.0) / 3.0) == 5.0
      summary = @summary.make_summary(@records)

      summary["ioinfo"]["sda"]["r/s"].should be_within(0.005).of(5.0)
    end

    it 'should return 0.0 for r_await if all values are zero' do
      @records[0]["ioinfo"]["sda"]["r_await"] = 0.0
      @records[1]["ioinfo"]["sda"]["r_await"] = 0.0
      @records[2]["ioinfo"]["sda"]["r_await"] = 0.0

      summary = @summary.make_summary(@records)

      summary["ioinfo"]["sda"]["r_await"].should == 0.0
    end

    it 'should calculate avg. r_await/w_await correctly' do
      @records[0]["ioinfo"]["sda"]["r/s"] = 0.0
      @records[0]["ioinfo"]["sda"]["r_await"] = 0.0
      @records[1]["ioinfo"]["sda"]["r/s"] = 100.0
      @records[1]["ioinfo"]["sda"]["r_await"] = 1.0
      @records[2]["ioinfo"]["sda"]["r/s"] = 200.0
      @records[2]["ioinfo"]["sda"]["r_await"] = 4.0

      summary = @summary.make_summary(@records)

      summary["ioinfo"]["sda"]["r_await"].should be_within(0.003).of(3.0)
    end
  end

  describe 'make_summary method with 2 devices' do
    before(:each) do
      @records = @summary.read_logfile(@logfile_2devices)
    end

    it 'should calculate avg. r/s in total' do
      @records[0]["time"] = 0.0
      @records[1]["time"] = 0.5

      @records[0]["ioinfo"]["total"]["r/s"] = 0.0
      @records[1]["ioinfo"]["total"]["r/s"] = 10.0

      summary = @summary.make_summary(@records)

      summary["ioinfo"]["total"]["r/s"].should be_within(0.005).of(10.0)
    end
  end

  it "should respond to make_accumulation" do
    @summary.should respond_to(:make_accumulation)
  end

  describe "make_accumulation" do
    before(:each) do
      @records = @summary.read_logfile(@logfile)
    end

    it "should return valid format" do
      accum = @summary.make_accumulation(@records)
      accum.should be_a Hash
      accum.keys.should include "ioinfo"
      # accum.keys.should include "cpuinfo"
    end

    it "should return nil if no ioinfo" do
      @records.each do |record|
        record.delete("ioinfo")
      end

      @summary.make_accumulation(@records).should be_nil
    end

    it "should return nil if only 1 record given" do
      @summary.make_accumulation([@records.first]).should be_nil
    end

    it "should return valid IO data volume accumulation" do
      @records[0]["time"] = 0.0
      @records[0]["ioinfo"]["sda"]["r/s"] = 0.0
      @records[0]["ioinfo"]["sda"]["rsec/s"] = 0.0
      @records[0]["ioinfo"]["sda"]["avgrq-sz"] = 16.0
      @records[1]["time"] = 2.0
      @records[1]["ioinfo"]["sda"]["r/s"] = 2.0
      @records[1]["ioinfo"]["sda"]["rsec/s"] = 2.0
      @records[1]["ioinfo"]["sda"]["avgrq-sz"] = 16.0
      @records[2]["time"] = 4.0
      @records[2]["ioinfo"]["sda"]["r/s"] = 1.0
      @records[2]["ioinfo"]["sda"]["rsec/s"] = 4.0
      @records[2]["ioinfo"]["sda"]["avgrq-sz"] = 16.0

      accum = @summary.make_accumulation(@records)

      accum["ioinfo"]["sda"]["read_requests"].should be_within(0.006).of(6.0)
      accum["ioinfo"]["sda"]["read_bytes"].should be_within(1.0).of(6144.0)
    end
  end
end
