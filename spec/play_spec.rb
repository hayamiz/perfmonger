
require 'spec_helper'

describe '[play] subcommand' do
  it 'should print 3 JSON records for busy100.pgr' do
    busy100 = data_file "busy100.pgr"
    cmd = "#{perfmonger_bin} play #{busy100}"
    run(cmd)
    expect(last_command_started).to be_successfully_executed
    expect(last_command_started.stdout.lines.to_a.size).to eq 3

    run(cmd)
    last_command_started.stdout.each_line do |line|
      expect do
        JSON.parse(line)
      end.not_to raise_error

      json = JSON.parse(line)
      expect(json.keys.sort).to eq %w{time cpu disk net}.sort
    end
  end
end
