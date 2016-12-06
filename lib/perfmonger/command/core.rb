
module PerfMonger
module Command

class CoreFinder
  class << self
    def find(name, os = nil, arch = nil)
      # check os
      unless os
        case RUBY_PLATFORM
        when /linux/
          os = "linux"
        when /darwin/
          os = "darwin"
        else
          os = nil
        end
      end

      # check arch
      unless arch
        case RUBY_PLATFORM
        when /x86_64|amd64/
          arch = "amd64"
        when /i\d86/
          arch = "386"
        else
          arch = nil
        end
      end

      if !os || !arch
        return nil
      end

      suffix = "_" + os + "_" + arch

      path = File.expand_path("../../../exec/perfmonger-#{name}#{suffix}", __FILE__)

      if File.executable?(path)
        return path
      else
        return nil
      end
    end

    def recorder
      self.find("recorder")
    end

    def player
      self.find("player")
    end

    def summarizer
      self.find("summarizer")
    end

    def plot_formatter
      self.find("plot-formatter")
    end
  end
end

end # module
end # module
