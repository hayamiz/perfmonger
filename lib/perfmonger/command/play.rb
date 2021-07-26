
require 'optparse'
require 'json'

module PerfMonger
module Command

class PlayCommand < BaseCommand
  register_command 'play', "Play a perfmonger log file in JSON"

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger play [options] LOG_FILE

Options:
EOS

    @color = false
    @parser.on("-c", "--color", "Use colored JSON ouptut") do
      @color = true
    end

    @pretty = false
    @parser.on("-p", "--pretty", "Use human readable JSON ouptut") do
      @pretty = true
    end

    @disk_only_regex = nil
    @parser.on('--disk-only REGEX', "Select disk devices that matches REGEX (Ex. 'sd[b-d]')") do |regex|
      @disk_only_regex = regex
    end
  end

  def parse_args(argv)
    @parser.parse!(argv)

    if argv.size == 0
      puts("ERROR: PerfMonger log file is required")
      puts(@parser.help)
      exit(false)
    end

    @logfile = argv.shift
    if ! File.exists?(@logfile)
      puts("ERROR: No such file: #{@logfile}")
      puts(@parser.help)
      exit(false)
    end
  end

  def run(argv)
    parse_args(argv)

    @player_bin = ::PerfMonger::Command::CoreFinder.player()

    if ! @player_bin
      puts("[ERROR] no executable binary found.")
      exit(false)
    end

    cmd = [@player_bin]
    if @color
      cmd << "-color"
    end
    if @pretty
      cmd << "-pretty"
    end

    if @disk_only_regex
      cmd << "-disk-only"
      cmd << @disk_only_regex
    end

    cmd << @logfile

    Process.exec(*cmd)
  end
end

end # module Command
end # module PerfMonger
