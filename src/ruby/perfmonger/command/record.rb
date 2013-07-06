
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

    cmd = @option.make_command

    Process.exec(*cmd)
  end
end

end # module Command
end # module PerfMonger
