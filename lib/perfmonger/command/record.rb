require 'optparse'
require 'tempfile'
require 'tmpdir'
require 'json'
require 'etc'

module PerfMonger
module Command

class RecordCommand < BaseCommand
  register_command 'record', 'Record system performance information'

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::RecordOption.parse(argv)

    if @option.kill
      session_file = File.expand_path(sprintf("perfmonger-%s-session.pid", Etc.getlogin),
                                      Dir.tmpdir)
      Process.kill(:INT, File.read(session_file).to_i)
    else
      exec_record_cmd()
    end
  end

private
  def exec_record_cmd()
    cmd = @option.make_command

    if @option.background
      Process.daemon(true)
    else
      $stdout.puts("[recording to #{@option.logfile}]")
    end
    Process.exec(*cmd)
  end
end

end # module Command
end # module PerfMonger
