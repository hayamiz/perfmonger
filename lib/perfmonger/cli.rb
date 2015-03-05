
require 'optparse'

module PerfMonger
module CLI

class Runner
  def self.register_command(command_name, klass)
    @@commands ||= Hash.new
    @@aliases ||= Hash.new

    @@commands[command_name] = klass
  end

  def self.register_alias(alias_name, command_name)
    if @@commands.nil?
      raise RuntimeError.new("No command is registered yet.")
    end

    if ! @@commands.has_key?(command_name)
      raise RuntimeError.new("Command '#{command_name}' is not registered.")
    end

    @@aliases[alias_name] = command_name
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
      when "live"
        [0, command_name]
      when "record"
        [1, command_name]
      when "play"
        [2, command_name]
      when "stat"
        [3, command_name]
      when "plot"
        [4, command_name]
      else
        [999, command_name]
      end
    end

    max_len = commands.map(&:command_name).map(&:size).max
    command_list_str = commands.map do |command|
      # pad command names
      command_name = command.command_name
      command_name = command_name + (" " * (max_len - command_name.size))

      str = "    " + command_name + "   " + command.description

      if command.aliases && command.aliases.size > 0
        str += "\n" + "    " + (" " * max_len) + "   " +
          "Aliases: " + command.aliases.join(", ")
      end

      str
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

    if @@aliases[command_name]
      command_name = @@aliases[command_name]
    end
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
