
require 'optparse'

module PerfMonger
module CLI

class Runner
  def self.register_command(command_name, klass)
    @@commands ||= Hash.new
    @@commands[command_name] = klass
  end

  def initialize

  end

  def run(argv = ARGV)
    parser = OptionParser.new
    parser.banner = <<EOS
Usage: #{File.basename($0)} [options] COMMAND [args]

EOS

    ## make list of subcommands
    commands = @@commands.values.sort_by do |command|
      # important command first: sort by [priority, name]
      command_name = command.command_name
      case command_name
      when "record"
        [0, command_name]
      when "stat"
        [1, command_name]
      when "plot"
        [2, command_name]
      else
        [999, command_name]
      end
    end

    max_len = commands.map(&:command_name).map(&:size).max
    command_list_str = commands.map do |command|
      # pad command names
      command_name = command.command_name
      command_name = command_name + (" " * (max_len - command_name.size))
      "    " + command_name + "   " + command.description
    end.join("\n")

    subcommand_list = <<EOS

Commands:
#{command_list_str}
EOS

    parser.summary_indent = "  "

    parser.on('-h', '--help', 'Show this help') do
      puts(parser.help)
      puts(subcommand_list)
      exit(true)
    end

    parser.on('-v', '--version', 'Show version number') do
      puts("PerfMonger version " + PerfMonger::VERSION)
      exit(true)
    end

    parser.order!(argv)

    if argv.size == 0
      puts(parser.help)
      puts(subcommand_list)
      exit(false)
    end

    command_name = argv.shift
    command_class = @@commands[command_name]

    unless command_class
      puts("No such command: #{command_name}")
      puts(subcommand_list)
      exit(false)
    end

    command_class.new.run(argv)
  end
end

end
end
