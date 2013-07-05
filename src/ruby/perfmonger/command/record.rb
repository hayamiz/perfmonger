
require 'optparse'
require 'json'
require 'tempfile'
require 'tmpdir'

module PerfMonger
module Command

class RecordCommand < BaseCommand
  register_command 'record'

  def initialize
    @devices           = []
    @interval          = 1.0
    @verbose           = false
    @report_cpu        = false
    @report_io         = false
    @report_ctx_switch = false
    @logfile           = STDOUT

    @system_device_list = File.read("/proc/diskstats").each_line.map do |line|
      _, _, device = *line.strip.split
      device
    end

    super
  end

  def run(argv)
    setup_parser()

    @parser.parse!(argv)

    if ! @report_io && ! @report_ctx_switch
      @report_cpu = true
    end

    exec_record_cmd()
  end

private
  def setup_parser()
    @parser.on('-d', '--device DEVICE',
               'Device name to be monitored (e.g. sda, sdb, md0, dm-1).') do |device|
      unless @system_device_list.include?(device)
        raise OptionParser::InvalidArgument.new("No such device: #{device}")
      end
      @devices.push(device)
      @report_io = true
    end

    @parser.on('-i', '--interval SEC',
               'Amount of time between each measurement report.') do |interval|
      @interval = Float(interval)
    end

    @parser.on('-C', '--cpu', 'Report CPU usage.') do
      @report_cpu = true
    end

    @parser.on('-S', '--context-switch', 'Report context switches per sec.') do
      @report_ctx_switch = true
    end

    @parser.on('-l', '--logfile FILE') do |file|
      @logfile = file
    end

    @parser.on('-v', '--verbose') do
      @verbose = true
    end
  end

  def exec_record_cmd()
    # try to search perfmonger-record in build environment
    cmd = File.expand_path("../../../../perfmonger-record", __FILE__)

    # then search installed directory
    if ! File.executable?(cmd)
      cmd = File.expand_path("perfmonger-record", PerfMonger::BINDIR)
    end

    if ! File.executable?(cmd)
      puts("ERROR: perfmonger-record(1) not found!")
      exit(false)
    end

    args = []
    args << '-i'
    args << @interval.to_s
    args << '-C' if @report_cpu
    args << '-S' if @report_ctx_switch
    args << '-l' if @logfile != STDOUT
    args << @logfile if @logfile != STDOUT
    @devices.each do |device|
      args << '-d'
      args << device
    end
    args << 'v' if @verbose

    Process.exec(cmd, *args)
  end
end

end # module Command
end # module PerfMonger
