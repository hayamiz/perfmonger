require 'spec_helper'

RSpec.describe '[live] subcommand' do
  before(:each) do
    skip_if_proc_is_not_available
  end

  it 'should print JSON records for 3 seconds and exit successfully.' do
    cmd = "#{perfmonger_bin} live --timeout 3"
    run(cmd, 5)
    expect(last_command_started).to be_successfully_executed

    run(cmd)
    last_command_started.stdout.each_line do |line|
      expect do
        JSON.parse(line)
      end.not_to raise_error

      json = JSON.parse(line)
      expect(json.keys.sort).to eq %w{time cpu disk net}.sort
    end

    expect("perfmonger.pgr.gz").to be_an_existing_file
  end
end
