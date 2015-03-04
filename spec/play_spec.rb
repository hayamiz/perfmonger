
require 'spec_helper'

describe '[play] subcommand' do
  it 'should print 3 JSON records for busy100.pgr' do
    busy100 = data_file "busy100.pgr"
    cmd = "#{perfmonger_bin} play #{busy100}"
    run(cmd)
    assert_success(true)
    expect(stdout_from(cmd).lines.to_a.size).to eq 3

    stdout_from(cmd).each_line do |line|
      expect do
        JSON.parse(line)
      end.not_to raise_error

      json = JSON.parse(line)
      expect(json.keys.sort).to eq %w{time cpu disk net}.sort
    end
  end
end
