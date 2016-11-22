require 'optparse'
require 'tempfile'
require 'tmpdir'
require 'json'
require 'etc'

module PerfMonger
module Command

class RecordCommand < BaseCommand
  LOCKFILE = File.expand_path(".perfmonger.lock", Dir.tmpdir())

  register_command 'record', 'Record system performance information'

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::RecordOption.parse(argv)

    session_file = File.expand_path(sprintf("perfmonger-%s-session.pid", Etc.getlogin),
                                    Dir.tmpdir)
    begin
      session_pid = File.read(session_file).to_i
    rescue Errno::ENOENT
      # No session file
      session_pid = nil
    end

    if @option.kill
      unless session_pid
        # There is nothing to be killed
        return true
      end

      begin
        Process.kill(:INT, session_pid)
      rescue Errno::ESRCH
        # Session file has invalid (already dead) PID
        File.open(LOCKFILE, "w") do |f|
          f.flock(File::LOCK_EX)
          FileUtils.rm(session_file)
          f.flock(File::LOCK_UN)
        end
      end
    else
      if session_pid && Process.getpgid(session_pid)
        $stderr.puts("[ERROR] another perfmonger is already running.")
        return false
      end
      exec_record_cmd()
    end

    true
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
