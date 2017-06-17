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
      return true
    end

    if @option.status
      unless session_pid
        puts "[ERROR] No perfmonger-recorder is running."
        return false
      end

      begin
        # check if session_pid is valid
        gid = Process.getpgid(session_pid)

        cmdline = File.read("/proc/#{session_pid}/cmdline").split("\0")
        exe = cmdline.shift
        args = cmdline
        start_time = File::Stat.new("/proc/#{session_pid}").mtime
        elapsed_time = Time.now - start_time

        puts <<EOS
==== perfmonger record is running (PID: #{session_pid}) ====

* Running executable: #{exe}
* Arguments: #{args.join(" ")}
* Started at #{start_time} (running #{elapsed_time.to_i} sec)

EOS
      rescue Errno::ESRCH
        puts "[ERROR] No perfmonger-recorder is running."
      end

      return true
    end

    # run perfmonger-recorder (normal path)

    if @option.background
      # If perfmonger is going to start in background mode,
      # there must be an another session running.

      begin
        if session_pid && Process.getpgid(session_pid)
          $stderr.puts("[ERROR] another perfmonger is already running in background mode")
          return false
        end
      rescue Errno::ESRCH
        # Actually there is no perfmonger running. go through.
      end
    end

    exec_record_cmd()

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

    if ENV['PERFMONGER_DEBUG'] != nil && ! ENV['PERFMONGER_DEBUG'].empty?
      $stderr.puts("[debug] cmd: " + cmd.join(" "))
    end

    Process.exec(*cmd)
  end
end

end # module Command
end # module PerfMonger
