
module PerfMonger
module Command

class RecordOption
  attr_reader :devices
  attr_reader :interval
  attr_reader :verbose
  attr_reader :report_cpu
  attr_reader :no_disk
  attr_reader :logfile
  attr_reader :background
  attr_reader :kill
  attr_reader :status

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
    @recorder_bin = ::PerfMonger::Command::CoreFinder.recorder()

    if ! @recorder_bin
      puts("ERROR: no executable binaries")
      exit(false)
    end

    cmd = [@recorder_bin]
    cmd << sprintf("-interval=%.1fms", @interval * 1000)
    if ! @interval_backoff
      cmd << "-no-interval-backoff"
    end
    if @start_delay > 0
      cmd << "-start-delay"
      cmd << "#{@start_delay*1000}ms"
    end
    if @timeout
      cmd << "-timeout"
      cmd << "#{@timeout*1000}ms"
    end
    if @no_cpu
      cmd << "-no-cpu"
    end
    if @no_disk
      cmd << "-no-disk"
    end
    if @no_intr
      cmd << "-no-intr"
    end
    if @devices.size > 0
      cmd << "-disks"
      cmd << @devices.join(",")
    end
    if @background
      cmd << "-background"
    end

    # TODO: implement device filter

    cmd << "-output"
    cmd << @logfile

    raise NotImplementedError if @verbose

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
    @no_intr           = true
    @devices           = []
    @logfile           = "perfmonger.pgr"
    @background            = false
    @kill              = false

    @parser = OptionParser.new

    @parser.on('-d', '--disk DEVICE',
               'Device name to be monitored (e.g. sda, sdb, md0, dm-1).') do |device|
      @devices.push(device)
      @no_disk = false
    end

    @parser.on('--record-intr', 'Record per core interrupts count (experimental)') do
      @no_intr = false
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

    @parser.on('--background', 'Run in background') do
      @background = true
    end

    @parser.on('--kill', 'Stop currently running perfmonger-reocrd') do
      @kill = true
    end

    @parser.on('--status', 'Show currently running perfmonger-record status') do
      @status = true
    end

    @parser.on('-v', '--verbose') do
      @verbose = true
    end
  end
end

end # module Command
end # module PerfMonger
