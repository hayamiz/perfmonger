
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

    parser.summary_indent = "  "

    parser.on('-v', '--version', 'Show version number') do
      puts("PerfMonger version " + PerfMonger::VERSION)
      exit(true)
    end

    parser.order!(argv)

    if argv.size == 0
      puts parser.help
      puts <<EOS

Commands:
#{@@commands.keys.sort.map{|sc| "  " + sc}.join("\n")}
EOS
      exit(false)
    end

    command_name = argv.shift
    command_class = @@commands[command_name]

    unless command_class
      puts "No such command: #{command_name}"
      puts <<EOS

Commands:
#{@@commands.keys.sort.map{|sc| "  " + sc}.join("\n")}
EOS
      exit(false)
    end

    command_class.new.run(argv)
  end
end

end
end
