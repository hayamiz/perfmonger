
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
    header_record = records.shift # removed first all-zero line

    puts("")
    puts("== Performance summary of '#{@argv.join(" ")}' ==")
    printf("Execution time: %.4f\n", @end_time - @start_time)

    if header_record && header_record['cpuinfo']
      usr, sys, iowait, irq, soft, other = records.map do |record|
        [record['cpuinfo']['all']['%usr'] + record['cpuinfo']['all']['%nice'],
         record['cpuinfo']['all']['%sys'],
         record['cpuinfo']['all']['%iowait'],
         record['cpuinfo']['all']['%irq'],
         record['cpuinfo']['all']['%soft'],
         [100.0 - record['cpuinfo']['all']['%idle'], 0.0].max]
      end.inject do |a, b|
        (0..5).map do |idx|
          a[idx] + b[idx]
        end
      end.map do |sum|
        sum / records.size.to_f
      end.map do |avg|
        sprintf("%.4f", avg)
      end

      puts("")
      puts("* CPU USAGE")
      puts("     %usr: #{usr || 'N/A'}")
      puts("     %sys: #{sys || 'N/A'}")
      puts("  %iowait: #{iowait || 'N/A'}")
      puts("     %irq: #{irq || 'N/A'}")
      puts("    %soft: #{soft || 'N/A'}")
      puts("   %other: #{other || 'N/A'}")
    end

    if header_record && header_record['ioinfo']
      records.first['ioinfo']['devices'].each do |device|
        r_iops, w_iops, r_sec, w_sec = records.map do |record|
          [record['ioinfo'][device]['r/s'],
           record['ioinfo'][device]['w/s'],
           record['ioinfo'][device]['rsec/s'] * 512 / 1024.0 / 1024.0,
           record['ioinfo'][device]['wsec/s'] * 512 / 1024.0 / 1024.0]
        end.inject do |a, b|
          (0..3).map do |idx|
            a[idx] + b[idx]
          end
        end.map do |sum|
          sum / records.size.to_f
        end.map do |avg|
          sprintf("%.4f", avg)
        end

        puts("")
        puts("* DEVICE: #{device}")
        puts("        avg. read IOPS: #{r_iops || 'N/A'}")
        puts("       avg. write IOPS: #{w_iops || 'N/A'}")
        puts("  avg. read throughput: #{r_sec || 'N/A'} MB/s")
        puts(" avg. write throughput: #{w_sec || 'N/A'} MB/s")
      end
    end
  end

  private
end

end # module Command
end # module PerfMonger
