require 'aruba/api'
require 'aruba/reporting'

RSpec.configure do |config|
  config.include Aruba::Api

  config.before(:each) do
    setup_aruba
  end
end
