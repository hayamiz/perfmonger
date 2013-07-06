
require 'optparse'
require 'json'

module PerfMonger
module Command

class StatCommand < BaseCommand
  register_command 'stat'

  def initialize
    super
  end

  def run(argv)
    @argv, @option = PerfMonger::Command::StatOption.parse(argv)

    if @argv.size == 0
      puts("ERROR: No command given.")
      exit(false)
    end

    # Search perfmonger-record binary: First, search perfmonger-record
    # in build environment, then in installed directory
    record_bin = [File.expand_path("../../../../perfmonger-record", __FILE__),
                  File.expand_path("perfmonger-record", PerfMonger::BINDIR)].find do |bin|
      File.executable?(bin)
    end

    if record_bin.nil?
      puts("ERROR: perfmonger-record not found!")
      exit(false)
    end

    record_cmd = @option.make_command

    begin
      record_pid = Process.spawn(*record_cmd)

      @start_time = Time.now
      command_pid = Process.spawn(*@argv)
      Process.wait(command_pid)
    ensure
      @end_time = Time.now
      Process.kill(:INT, record_pid)
    end

    show_summary
  end

  def show_summary
    sleep(1)
    records = File.read(@option.logfile).lines.map do |line|
      begin
        JSON.parse(line)
      rescue JSON::ParserError => err
        nil
      end
    end.compact
    records.shift # removed first all-zero line

    puts("== Performance summary of '#{@argv.join(" ")}' ==")
    printf("Execution time: %.4f\n", @end_time - @start_time)

    if records.first['ioinfo']
      records.first['ioinfo']['devices'].each do |device|
        r_iops, w_iops, r_sec, w_sec = records.map do |record|
          [record['ioinfo'][device]['r/s'],
           record['ioinfo'][device]['w/s'],
           record['ioinfo'][device]['rsec/s'],
           record['ioinfo'][device]['wsec/s']]
        end.inject do |a, b|
          (0..3).map do |idx|
            a[idx] + b[idx]
          end
        end.map do |sum|
          sum / records.size.to_f
        end

        puts("")
        puts("* DEVICE: #{device}")
        printf("        avg. read IOPS: %.4f\n", r_iops)
        printf("       avg. write IOPS: %.4f\n", w_iops)
        printf("  avg. read throuhgput: %.4f MB/s\n", r_sec * 512 / 1024.0 / 1024.0)
        printf(" avg. write throughput: %.4f MB/s\n", w_sec * 512 / 1024.0 / 1024.0)
      end
    end
  end

  private
end

end # module Command
end # module PerfMonger
