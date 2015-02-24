
require 'optparse'
require 'rubygems' # for ruby 1.8
require 'json'
require 'tempfile'
require 'tmpdir'

module PerfMonger
module Command

class RecordCommand < BaseCommand
  register_command 'record', 'Record system performance information (deprecated)'

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::RecordOption.parse(argv)

    exec_record_cmd()
  end

private
  def exec_record_cmd()
    cmd = @option.make_command

    Process.exec(*cmd)
  end
end

end # module Command
end # module PerfMonger
