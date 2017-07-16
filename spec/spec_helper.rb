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
  File.expand_path('../../exe/perfmonger', __FILE__)
end

RSpec.configure do |config|
  config.expect_with :rspec do |expectations|
    expectations.include_chain_clauses_in_custom_matcher_descriptions = true
  end

  config.mock_with :rspec do |mocks|
    mocks.verify_partial_doubles = true
  end

  config.shared_context_metadata_behavior = :apply_to_host_groups

  config.filter_run_when_matching :focus

  config.example_status_persistence_file_path = "spec/examples.txt"

  config.disable_monkey_patching!

  config.warnings = true

  if config.files_to_run.one?
    config.default_formatter = "doc"
  end

  config.profile_examples = 10

  config.order = :random
  # Kernel.srand config.seed
end

def skip_if_proc_is_not_available
  if ! File.exist?("/proc/diskstats")
    skip "/proc/diskstats is not available."
  end
end
