
module PerfMonger
module Command

class StatOption < RecordOption
  private
  def initialize
    super()
    @parser.banner = <<EOS
Usage: perfmonger stat [options] -- <command>

Run a command and gather performance information during its execution.

Options:
EOS

    @logfile = './perfmonger.log'
  end
end

end # module Command
end # module PerfMonger
