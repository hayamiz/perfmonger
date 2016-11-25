
require 'optparse'
require 'json'
require 'tempfile'
require 'tmpdir'

module PerfMonger
module Command

class InitShellCommand < BaseCommand
  register_command 'init-shell', "Generate shell script to init shell completion"

  def run(argv)
    gem_dir = File.expand_path("../../../", __dir__)

    shell = `ps -p #{Process.ppid()} -o 'args='`.strip
    shell = File.basename(shell.split.first)

    if shell == "zsh"
      if argv.first == "-"
        puts <<EOS
source #{File.expand_path("misc/perfmonger.zsh", gem_dir)}
EOS
      else
        puts <<EOS
# Add a following line to ~/.zshrc

eval "$(perfmonger init-shell -)"
EOS
      end
    end

    true
  end
end

end # module Command
end # module PerfMonger
