require 'spec_helper'

describe '[live] subcommand' do
  before(:each) do
    skip_if_proc_is_not_available
  end

  it 'should print JSON records for 3 seconds and exit successfully.' do
    cmd = "#{perfmonger_bin} live --timeout 3"
    run(cmd, 5)
    assert_success(true)
    expect(stdout_from(cmd).lines.to_a.size).to eq 3

    stdout_from(cmd).each_line do |line|
      expect do
        JSON.parse(line)
      end.not_to raise_error

      json = JSON.parse(line)
      expect(json.keys.sort).to eq %w{time cpu disk net}.sort
    end

    check_file_presence("perfmonger.pgr")
  end
end
