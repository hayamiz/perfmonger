require 'optparse'
require 'tempfile'
require 'tmpdir'
require 'json'

module PerfMonger
module Command

class LiveOption < RecordOption
  def make_command
    cmd = super()
    @player_bin = ::PerfMonger::Command::CoreFinder.player()
    cmd += ["-player-bin", @player_bin]
  end
end

class LiveCommand < BaseCommand
  register_command 'live', 'Record and play system performance information in JSON'

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::LiveOption.parse(argv)

    exec_live_cmd()
  end

private
  def exec_live_cmd()
    cmd = @option.make_command

    Process.exec(*cmd)
  end
end

end # module Command
end # module PerfMonger
