
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

# class StatCommand < RecordCommand
#   register_command 'stat'
# 
#   def initialize
#     super
#     @logfile = "./perfmonger.log"
#   end
# 
#   def run(argv)
#     setup_parser()
#     @parser.parse!(argv)
#     @command = argv
# 
#     if ! @report_io && ! @report_ctx_switch
#       @report_cpu = true
#     end
# 
#     exec_stat_cmd()
#   end
# 
# private
#   def setup_parser()
#     super
#     @parser.banner = <<EOS
# Usage: #{File.basename($0)} stat [options] -- COMMANDS
# 
# Options:
# EOS
#   end
# 
#   def exec_stat_cmd()
#     cmd = File.expand_path("../perfmonger-record", __FILE__)
# 
#     args = []
#     args << '-i'
#     args << @interval.to_s
#     args << '-C' if @report_cpu
#     args << '-S' if @report_ctx_switch
#     args << '-l' if @logfile != STDOUT
#     args << @logfile if @logfile != STDOUT
#     @devices.each do |device|
#       args << '-d'
#       args << device
#     end
#     args << 'v' if @verbose
# 
#     record_pid = Process.spawn(cmd, *args)
#     system(*@command)
#     Process.kill("INT", record_pid)
#   end
# end

end
end
