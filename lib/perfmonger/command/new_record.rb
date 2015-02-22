
require 'optparse'
require 'json'
require 'tempfile'
require 'tmpdir'

module PerfMonger
module Command

class NewRecordCommand < BaseCommand
  register_command 'new_record', 'Record system performance information'

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::NewRecordOption.parse(argv)

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
