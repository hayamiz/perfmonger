
module PerfMonger
module Command

class BaseCommand
  class << self
    attr_accessor :command_name
    attr_accessor :description

    def register_command(command_name, description = "")
      PerfMonger::CLI::Runner.register_command(command_name, self)
      self.command_name = command_name
      self.description = description
    end
  end

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: #{File.basename($0)} #{self.class.command_name} [options]

Options:
EOS
  end
end

end
end
