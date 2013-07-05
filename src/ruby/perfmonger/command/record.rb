
require 'optparse'
require 'json'
require 'tempfile'
require 'tmpdir'

module PerfMonger
module Command

class RecordCommand < BaseCommand
  register_command 'record'

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::RecordOption.parse(argv)

    exec_record_cmd()
  end

private
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
    args << @option.interval.to_s
    args << '-C' if @option.report_cpu
    args << '-S' if @option.report_ctx_switch
    args << '-l' if @option.logfile != STDOUT
    args << @option.logfile if @option.logfile != STDOUT
    @option.devices.each do |device|
      args << '-d'
      args << device
    end
    args << 'v' if @option.verbose

    Process.exec(cmd, *args)
  end
end

end # module Command
end # module PerfMonger
