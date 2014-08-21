
module PerfMonger
module Command

class BaseCommand
  class << self
    attr_accessor :command_name
    attr_accessor :description
    attr_accessor :aliases

    def register_command(command_name, description = "")
      PerfMonger::CLI::Runner.register_command(command_name, self)
      self.command_name = command_name
      self.description = description
    end

    def register_alias(alias_name)
      if self.command_name
        RuntimeError.new("#{self} does not have registered command name.")
      end

      self.aliases ||= []
      self.aliases.push(alias_name)
      PerfMonger::CLI::Runner.register_alias(alias_name, self.command_name)
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
