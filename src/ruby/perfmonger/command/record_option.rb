
module PerfMonger
module Command

class RecordOption
  attr_reader :devices
  attr_reader :interval
  attr_reader :verbose
  attr_reader :report_cpu
  attr_reader :report_io
  attr_reader :report_ctx_switch
  attr_reader :logfile
  attr_reader :system_device_list

  attr_reader :parser

  def self.parse(argv)
    option = self.new
    argv = option.parse(argv)

    return argv, option
  end

  def parse(argv)
    argv = @parser.parse(argv)

    if ! @report_io && ! @report_ctx_switch
      @report_cpu = true
    end

    argv
  end

  private
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

    @parser = OptionParser.new

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
end

end # module Command
end # module PerfMonger
