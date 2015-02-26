
if RUBY_VERSION <= '1.8.7'
  require 'rubygems'
end

require 'optparse'
require 'json'
require 'tempfile'
require 'tmpdir'

module PerfMonger
module Command

class OldRecordCommand < BaseCommand
  register_command 'old_record', 'Record system performance information (deprecated)'

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::OldRecordOption.parse(argv)

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