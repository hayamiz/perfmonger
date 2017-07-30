require 'optparse'
require 'tempfile'
require 'tmpdir'
require 'json'

module PerfMonger
module Command

class LiveOption < RecordOption
  def initialize
    super

    @color = false
    @parser.on("-c", "--color", "Use colored JSON output") do
      @color = true
    end

    @pretty
    @parser.on("--pretty",  "Use human readable JSON output") do
      @pretty = true
    end

  end

  def make_command
    cmd = super()
    @player_bin = ::PerfMonger::Command::CoreFinder.player()
    cmd += ["-player-bin", @player_bin]
    cmd << "-color" if @color
    cmd << "-pretty" if @pretty

    cmd
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
