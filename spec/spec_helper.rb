# encoding: utf-8

$LOAD_PATH << File.expand_path('../../src/ruby', __FILE__)

TEST_DATA_DIR = File.expand_path('../data', __FILE__)

require 'perfmonger'

def data_file(rel_path)
  File.expand_path(rel_path, TEST_DATA_DIR)
end

RSpec.configure do |config|
  # RSpec config here
end
