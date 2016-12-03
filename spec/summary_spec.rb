
require 'spec_helper'

describe '[summary] subcommand' do
  it 'should print valid output' do
    busy100 = data_file "busy100.pgr"
    cmd = "#{perfmonger_bin} summary #{busy100}"
    run(cmd)
    expect(last_command_started).to be_successfully_executed
    output = last_command_started.stdout

    # measurement duration
    expect(output).to match(/^Duration: (\d+\.\d+) sec$/)

    # CPU usage
    non_idle_regex = /Non-idle usage: (\d+\.\d+) %$/
    usr_regex = /%usr: (\d+\.\d+) %$/
    idle_regex = /Idle usage: (\d+\.\d+) %$/
    expect(output).to match(non_idle_regex)
    expect(output).to match(usr_regex)
    expect(output).to match(idle_regex)

    non_idle_regex =~ output; non_idle_usage = Float($~[1])
    idle_regex =~ output; idle_usage = Float($~[1])
    usr_regex =~ output; usr_usage = Float($~[1])

    expect(non_idle_usage).to be_within(1.0).of(100.0)
    expect(usr_usage).to be_within(1.0).of(100.0)
    expect(idle_usage).to be_within(1.0).of(99.0)
    expect(non_idle_usage + idle_usage).to be_within(0.1).of(200.0)

    # disk usage
    expect(output).to match(/^\* Average DEVICE usage: .+$/)
  end

  it 'should print valid JSON if --json option is given' do
    busy100 = data_file "busy100.pgr"
    cmd = "#{perfmonger_bin} summary --json #{busy100}"
    run(cmd)
    expect(last_command_started).to be_successfully_executed
    output = last_command_started.stdout

    expect do
      JSON.parse(output)
    end.not_to raise_error

    json = JSON.parse(output)

    expect(json.keys.sort).to eq %w{cpu disk net exectime}.sort
  end
end
