
require 'optparse'
require 'json'

module PerfMonger
module Command

class SummaryCommand < BaseCommand
  register_command 'summary', "Show a summary of a perfmonger log file"

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger summary [options] LOG_FILE

Options:
EOS

    @json = false
    @pager = nil

    @parser.on('--json', "Output summary in JSON") do
      @json = true
    end

    @parser.on('-p', '--pager [PAGER]', "Use pager to see summary output.") do |pager|
      if pager.nil?
        if ENV['PAGER'].nil?
          puts("ERROR: No pager is available.")
          puts("ERROR: Please set PAGER or give pager name to --pager option.")
          puts(@parser.help)
          exit(false)
        else
          @pager = ENV['PAGER']
        end
      else
        @pager = pager
      end
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

  def run(argv, summary_title = nil)
    parse_args(argv)

    summary_title ||= @logfile

    @summarizer_bin = ::PerfMonger::Command::CoreFinder.summarizer()

    if ! @summarizer_bin
      puts("[ERROR] no executable binary found.")
      exit(false)
    end

    cmd = [@summarizer_bin]

    if @json
      cmd << "-json"
    end

    cmd << "-title"
    cmd << summary_title

    cmd << @logfile

    Process.exec(*cmd)
  end
end

end # module Command
end # module PerfMonger
