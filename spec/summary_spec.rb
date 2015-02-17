
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
      expect(records.size).to eq 3
      records.each do |record|
        expect(record).to be_a Hash
        expect(record).to include "time"

        expect(record).to include "ioinfo"
        expect(record["ioinfo"]).to include "devices"
        expect(record["ioinfo"]["devices"]).to be_a Array
        expect(record["ioinfo"]["devices"]).to include "sda"
        expect(record["ioinfo"]).to include "sda"
        expect(record["ioinfo"]["sda"]).to be_a Hash
        expect(record["ioinfo"]["sda"]).to include "riops"
        expect(record["ioinfo"]["sda"]).to include "wiops"
        expect(record["ioinfo"]["sda"]).to include "rsecps"
        expect(record["ioinfo"]["sda"]).to include "wsecps"
        expect(record["ioinfo"]["sda"]).to include "r_await"
        expect(record["ioinfo"]["sda"]).to include "w_await"

        expect(record).to include "cpuinfo"
        expect(record["cpuinfo"]).to be_a Hash
        expect(record["cpuinfo"]).to include "nr_cpu"
        expect(record["cpuinfo"]).to include "cpus"
        expect(record["cpuinfo"]).to include "all"
      end
    end
  end

  describe 'make_summary method' do
    before(:each) do
      @records = @summary.read_logfile(@logfile)
    end

    it 'should return nil for empty records' do
      expect(@summary.make_summary([])).to eq nil
    end

    it 'should return valid format result' do
      summary = @summary.make_summary(@records)

      expect(summary).to be_a Hash
      expect(summary).to include "time"

      expect(summary).to include "ioinfo"
      expect(summary["ioinfo"]).to include "devices"
      expect(summary["ioinfo"]["devices"]).to be_a Array
      expect(summary["ioinfo"]["devices"]).to include "sda"
      expect(summary["ioinfo"]).to include "sda"
      expect(summary["ioinfo"]["sda"]).to be_a Hash
      expect(summary["ioinfo"]["sda"]).to include "riops"
      expect(summary["ioinfo"]["sda"]).to include "wiops"
      expect(summary["ioinfo"]["sda"]).to include "rsecps"
      expect(summary["ioinfo"]["sda"]).to include "wsecps"
      expect(summary["ioinfo"]["sda"]).to include "r_await"
      expect(summary["ioinfo"]["sda"]).to include "w_await"

      expect(summary).to include "cpuinfo"
      expect(summary["cpuinfo"]).to include "nr_cpu"
      expect(summary["cpuinfo"]).to include "cpus"
      expect(summary["cpuinfo"]).to include "all"
      cpu_entries = [summary["cpuinfo"]["all"], *summary["cpuinfo"]["cpus"]]
      cpu_entries.each do |entry|
        expect(entry).to be_a Hash
        expect(entry).to include "usr"
        expect(entry).to include "nice"
        expect(entry).to include "sys"
        expect(entry).to include "iowait"
        expect(entry).to include "irq"
        expect(entry).to include "soft"
        expect(entry).to include "steal"
        expect(entry).to include "guest"
        expect(entry).to include "idle"
      end
    end

    it 'should calculate avg. IOPS correctly against non-equal intervals' do
      @records[0]["time"] = 0.0
      @records[1]["time"] = 1.0
      @records[2]["time"] = 3.0

      @records[0]["ioinfo"]["sda"]["riops"] = 0.0 # no effect for result
      @records[1]["ioinfo"]["sda"]["riops"] = 3.0
      @records[2]["ioinfo"]["sda"]["riops"] = 6.0

      # avg. riops should be ((3.0 * 1.0 + 6.0 * 3.0) / 3.0) == 5.0
      summary = @summary.make_summary(@records)

      expect(summary["ioinfo"]["sda"]["riops"]).to be_within(5.0e-6).of(5.0)
    end

    it 'should return 0.0 for r_await if all values are zero' do
      @records[0]["ioinfo"]["sda"]["r_await"] = 0.0
      @records[1]["ioinfo"]["sda"]["r_await"] = 0.0
      @records[2]["ioinfo"]["sda"]["r_await"] = 0.0

      summary = @summary.make_summary(@records)

      expect(summary["ioinfo"]["sda"]["r_await"]).to eq 0.0
    end

    it 'should calculate avg. r_await/w_await correctly' do
      @records[0]["ioinfo"]["sda"]["riops"] = 0.0
      @records[0]["ioinfo"]["sda"]["r_await"] = 0.0
      @records[1]["ioinfo"]["sda"]["riops"] = 100.0
      @records[1]["ioinfo"]["sda"]["r_await"] = 1.0
      @records[2]["ioinfo"]["sda"]["riops"] = 200.0
      @records[2]["ioinfo"]["sda"]["r_await"] = 4.0

      summary = @summary.make_summary(@records)

      expect(summary["ioinfo"]["sda"]["r_await"]).to be_within(3.0e-6).of(3.0)
    end
  end

  describe 'make_summary method with 2 devices' do
    before(:each) do
      @records = @summary.read_logfile(@logfile_2devices)
    end

    it 'should calculate avg. riops in total' do
      @records[0]["time"] = 0.0
      @records[1]["time"] = 0.5

      @records[0]["ioinfo"]["total"]["riops"] = 0.0
      @records[1]["ioinfo"]["total"]["riops"] = 10.0

      summary = @summary.make_summary(@records)

      expect(summary["ioinfo"]["total"]["riops"]).to be_within(10.0e-6).of(10.0)
    end
  end

  it "should respond to make_accumulation" do
    expect(@summary).to respond_to(:make_accumulation)
  end

  describe "make_accumulation" do
    before(:each) do
      @records = @summary.read_logfile(@logfile)
    end

    it "should return valid format" do
      accum = @summary.make_accumulation(@records)
      expect(accum).to be_a Hash
      expect(accum.keys).to include "ioinfo"
    end

    it "should return nil if no ioinfo" do
      @records.each do |record|
        record.delete("ioinfo")
      end

      expect(@summary.make_accumulation(@records)).to be_nil
    end

    it "should return nil if only 1 record given" do
      expect(@summary.make_accumulation([@records.first])).to be_nil
    end

    it "should return valid IO data volume accumulation" do
      @records[0]["time"] = 0.0
      @records[0]["ioinfo"]["sda"]["riops"] = 0.0
      @records[0]["ioinfo"]["sda"]["rsecps"] = 0.0
      @records[0]["ioinfo"]["sda"]["avgrq-sz"] = 16.0
      @records[1]["time"] = 2.0
      @records[1]["ioinfo"]["sda"]["riops"] = 2.0
      @records[1]["ioinfo"]["sda"]["rsecps"] = 2.0
      @records[1]["ioinfo"]["sda"]["avgrq-sz"] = 16.0
      @records[2]["time"] = 4.0
      @records[2]["ioinfo"]["sda"]["riops"] = 1.0
      @records[2]["ioinfo"]["sda"]["rsecps"] = 4.0
      @records[2]["ioinfo"]["sda"]["avgrq-sz"] = 16.0

      accum = @summary.make_accumulation(@records)

      expect(accum["ioinfo"]["sda"]["read_requests"]).to be_within(6.0e-6).of(6.0)
      expect(accum["ioinfo"]["sda"]["read_bytes"]).to be_within(1e-6).of(6144.0)
    end
  end
end
