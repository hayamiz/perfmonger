# encoding: utf-8

$LOAD_PATH << File.expand_path('../../src/ruby', __FILE__)

TEST_DATA_DIR = File.expand_path('../data', __FILE__)

require 'perfmonger'
require 'tempfile'
require 'pathname'

def data_file(rel_path)
  from = Pathname.new(Dir.pwd)
  path = Pathname.new(File.expand_path(rel_path, TEST_DATA_DIR))

  path.relative_path_from(from).to_s
end

RSpec.configure do |config|
  # RSpec config here
end
