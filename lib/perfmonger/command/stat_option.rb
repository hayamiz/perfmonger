
module PerfMonger
module Command

class StatOption < RecordOption
  attr_reader :json

  private
  def initialize
    super()
    @parser.banner = <<EOS
Usage: perfmonger stat [options] -- <command>

Run a command and gather performance information during its execution.

Options:
EOS

    @logfile = './perfmonger.pgr'
    @json = false

    @parser.on('--json', "Output summary in JSON") do
      @json = true
    end
  end
end

end # module Command
end # module PerfMonger
