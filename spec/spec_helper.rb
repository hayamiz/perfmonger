$LOAD_PATH << File.expand_path('../../lib', __FILE__)

if ENV['RUBYLIB']
  ENV['RUBYLIB'] += ":"
else
  ENV['RUBYLIB'] = ""
end
ENV['RUBYLIB'] += File.expand_path('../../lib', __FILE__)

TEST_DATA_DIR = File.expand_path('../data', __FILE__)

require 'perfmonger'
require 'tempfile'
require 'pathname'
Dir.glob(::File.expand_path('../support/*.rb', __FILE__)).each { |f| require_relative f }

def data_file(rel_path)
  File.expand_path(rel_path, TEST_DATA_DIR)
end

def perfmonger_bin
  File.expand_path('../../bin/perfmonger', __FILE__)
end

RSpec.configure do |config|
  # RSpec config here
end

def skip_if_proc_is_not_available
  if ! File.exists?("/proc/diskstats")
    skip "/proc/diskstats is not available."
  end
end
