
require 'optparse'
require 'json'

module PerfMonger
module Command

class StatCommand < BaseCommand
  register_command 'stat', "Run a command and record system performance during execution"

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::StatOption.parse(argv)

    if @argv.size == 0
      puts("ERROR: No command given.")
      exit(false)
    end

    # Search perfmonger-record binary: First, search perfmonger-record
    # in build environment, then in installed directory
    record_bin = [File.expand_path("../../../../perfmonger-record", __FILE__),
                  File.expand_path("perfmonger-record", PerfMonger::BINDIR)].find do |bin|
      File.executable?(bin)
    end

    if record_bin.nil?
      puts("ERROR: perfmonger-record not found!")
      exit(false)
    end

    record_cmd = @option.make_command

    begin
      record_pid = Process.spawn(*record_cmd)

      @start_time = Time.now
      command_pid = Process.spawn(*@argv)
      Process.wait(command_pid)
    ensure
      @end_time = Time.now
      Process.kill(:INT, record_pid)
      Process.wait(record_pid)
    end

    puts("")
    printf("Execution time: %.4f\n", @end_time - @start_time)
    summary_command = SummaryCommand.new.run([@option.logfile], @argv.join(" "))
  end
end

end # module Command
end # module PerfMonger
