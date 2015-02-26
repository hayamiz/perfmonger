
module PerfMonger
module Command

class RecordOption
  attr_reader :devices
  attr_reader :interval
  attr_reader :verbose
  attr_reader :report_cpu
  attr_reader :no_disk
  attr_reader :logfile

  attr_reader :parser
  attr_accessor :record_bin

  def self.parse(argv)
    option = self.new
    argv = option.parse(argv)

    return argv, option
  end

  def parse(argv)
    argv = @parser.parse(argv)

    @no_cpu = false
    @no_disk = false
    @all_devices = true

    argv
  end

  def make_command
    # try to search perfmonger-record in build environment
    # then search installed directory

    # check os
    case RUBY_PLATFORM
    when /linux/
      os = "linux"
    else
      os = nil
    end

    # check arch
    case RUBY_PLATFORM
    when /x86_64|amd64/
      arch = "amd64"
    when /i\d86/
      arch = "386"
    else
      arch = nil
    end

    if !os || !arch
      puts("[ERROR] unsupported platform: " + RUBY_PLATFORM)
      exit(false)
    end

    suffix = "_" + os + "_" + arch

    @recorder_bin = File.expand_path("../../../exec/perfmonger-recorder#{suffix}", __FILE__)
    @player_bin = File.expand_path("../../../exec/perfmonger-player#{suffix}", __FILE__)

    if ! File.executable?(@recorder_bin) || ! File.executable?(@player_bin)
      puts("ERROR: no executable binaries")
      exit(false)
    end

    cmd = sprintf("%s -interval=%.1fms",
                  @recorder_bin,
                  @interval * 1000)
    if ! @interval_backoff
      cmd += " -no-interval-backoff "
    end
    if @start_delay > 0
      cmd += " -start-delay #{@start_delay*1000}ms "
    end
    if @timeout
      cmd += " -timeout #{@timeout*1000}ms "
    end
    if @no_cpu
      cmd += " -no-cpu "
    end
    if @no_disk
      cmd += " -no-disk "
    end

    if @logfile
      cmd += sprintf(" -output \"%s\" ", @logfile)
    end

    raise NotImplementedError if @verbose

    if ! @logfile
      # output JSON to stdout via player
      cmd += " | " + @player_bin
    end

    cmd
  end

  private
  def initialize
    @interval          = 1.0 # in second
    @interval_backoff  = true
    @start_delay       = 0.0 # in second
    @timeout           = nil # in second, or nil (= no timeout)
    @verbose           = false
    @no_cpu            = false
    @no_disk           = false
    @all_devices       = true
    @devices           = []
    @logfile           = nil

    @parser = OptionParser.new

    @parser.on('-d', '--device DEVICE',
               'Device name to be monitored (e.g. sda, sdb, md0, dm-1).') do |device|
      @devices.push(device)
      @no_disk = false
    end

    @parser.on('-i', '--interval SEC',
               'Amount of time between each measurement report. Floating point is o.k.') do |interval|
      @interval = Float(interval)
    end

    @parser.on('-B', '--no-interval-backoff',
               'Prevent interval to be set longer every after 100 records.') do
      @interval_backoff = false
    end

    @parser.on('-s', '--start-delay SEC',
               'Amount of wait time before starting measurement. Floating point is o.k.') do |start_delay|
      @start_delay = Float(start_delay)
    end

    @parser.on('-t', '--timeout SEC',
               'Amount of measurement time. Floating point is o.k.') do |timeout|
      @timeout = Float(timeout)
    end

    @parser.on('--no-cpu', 'Suppress recording CPU usage.') do
      @no_cpu = true
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
