
require 'optparse'
require 'json'

module PerfMonger
module Command

class SummaryCommand < BaseCommand
  register_command 'summary', "Show a summary of a perfmonger log file"

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger summary [options] LOG_FILE

Options:
EOS

    @json = false
    @pager = nil

    @parser.on('--json', "Output summary in JSON") do
      @json = true
    end

    @parser.on('-p', '--pager [PAGER]', "Use pager to see summary output.") do |pager|
      if pager.nil?
        if ENV['PAGER'].nil?
          puts("ERROR: No pager is available.")
          puts("ERROR: Please set PAGER or give pager name to --pager option.")
          puts(@parser.help)
          exit(false)
        else
          @pager = ENV['PAGER']
        end
      else
        @pager = pager
      end
    end
  end

  def parse_args(argv)
    @parser.parse!(argv)

    if argv.size == 0
      puts("ERROR: PerfMonger log file is required")
      puts(@parser.help)
      exit(false)
    end

    @logfile = argv.shift
    if ! File.exists?(@logfile)
      puts("ERROR: No such file: #{@logfile}")
      puts(@parser.help)
      exit(false)
    end
  end

  def run(argv, summary_title = nil)
    parse_args(argv)

    summary_title ||= @logfile

    show_summary(@logfile, summary_title)
  end


  def read_logfile(logfile)
    File.read(logfile).lines.map do |line|
      begin
        JSON.parse(line)
      rescue JSON::ParserError => err
        nil
      end
    end.compact
  end

  def make_accumulation(records)
    unless records.all?{|record| record.has_key?("ioinfo")}
      return nil
    end
    unless records.size > 1
      return nil
    end

    accum = Hash.new

    devices = records.first["ioinfo"]["devices"]
    accum["ioinfo"] = Hash.new

    devices.each do |device|
      read_requests = 0
      read_bytes = 0
      write_requests = 0
      write_bytes = 0

      (1..(records.size - 1)).each do |idx|
        last_record = records[idx - 1]
        record = records[idx]
        dt = record["time"] - last_record["time"]

        read_requests += record["ioinfo"][device]["riops"] * dt
        write_requests += record["ioinfo"][device]["wiops"] * dt
        read_bytes += record["ioinfo"][device]["rsecps"] * 512 * dt
        write_bytes += record["ioinfo"][device]["wsecps"] * 512 * dt
      end

      accum["ioinfo"][device] = Hash.new
      accum["ioinfo"][device]["read_requests"] = read_requests
      accum["ioinfo"][device]["read_bytes"] = read_bytes
      accum["ioinfo"][device]["write_requests"] = write_requests
      accum["ioinfo"][device]["write_bytes"] = write_bytes
    end

    accum
  end

  def make_summary(records)
    if records.empty?
      return nil
    elsif records.size == 1
      return records.first
    end

    # setup getters and setters all attributes for avg. calculation
    # getter.call(record) returns value
    # setter.call(record, value) set value
    # getters and setters include attribute info as a closure
    getters = []
    setters = []

    if records.first.include?("ioinfo")
      records.first["ioinfo"]["devices"].each do |device|
        records.first["ioinfo"][device].keys.each do |attr|
          getters << lambda do |record|
            record["ioinfo"][device][attr]
          end
          setters << lambda do |record, value|
            record["ioinfo"][device][attr] = value
          end
        end
      end

      records.first["ioinfo"]["total"].keys.each do |attr|
        getters << lambda do |record|
          record["ioinfo"]["total"][attr]
        end
        setters << lambda do |record, value|
          record["ioinfo"]["total"][attr] = value
        end
      end
    end

    if records.first.include?("cpuinfo")
      records.first["cpuinfo"]["all"].keys.each do |attr|
        getters << lambda do |record|
          record["cpuinfo"]["all"][attr]
        end
        setters << lambda do |record, value|
          record["cpuinfo"]["all"][attr] = value
        end
      end

      records.first["cpuinfo"]["nr_cpu"].times do |cpu_idx|
        records.first["cpuinfo"]["cpus"][cpu_idx].keys.each do |attr|
          getters << lambda do |record|
            record["cpuinfo"]["cpus"][cpu_idx][attr]
          end
          setters << lambda do |record, value|
            record["cpuinfo"]["cpus"][cpu_idx][attr] = value
          end
        end
      end
    end

    avg_record = Marshal.load(Marshal.dump(records.first)); # deep copy

    setters.each do |setter|
      setter.call(avg_record, 0.0)
    end

    (1..(records.size - 1)).each do |idx|
      record = records[idx]

      last_t = records[idx - 1]["time"]
      t      = record["time"]

      getters.size.times do |_etters_idx|
        getter = getters[_etters_idx]
        setter = setters[_etters_idx]

        setter.call(avg_record,
                    getter.call(avg_record) + getter.call(record) * (t - last_t))
      end
    end

    getters.size.times do |_etters_idx|
      getter = getters[_etters_idx]
      setter = setters[_etters_idx]

      setter.call(avg_record,
                  getter.call(avg_record) / (records[-1]["time"] - records[0]["time"]))
    end


    # r_await/w_await need special handling
    if records.first.include?("ioinfo")
      records.first["ioinfo"]["devices"].each do |device|
        accum_r_io_time = 0.0
        accum_w_io_time = 0.0
        r_io_count = 0
        w_io_count = 0

        (records.size - 1).times do |idx|
          rec0 = records[idx]
          rec1 = records[idx + 1]
          dev_ioinfo = rec1["ioinfo"][device]
          dt = rec1["time"] - rec0["time"]

          accum_r_io_time += dev_ioinfo["r_await"] * dev_ioinfo["riops"] * dt
          accum_w_io_time += dev_ioinfo["w_await"] * dev_ioinfo["wiops"] * dt
          r_io_count += dev_ioinfo["riops"] * dt
          w_io_count += dev_ioinfo["wiops"] * dt
        end

        if r_io_count > 0
          avg_record["ioinfo"][device]["r_await"] = accum_r_io_time / r_io_count
        else
          avg_record["ioinfo"][device]["r_await"] = 0.0
        end

        if w_io_count > 0
          avg_record["ioinfo"][device]["w_await"] = accum_w_io_time / w_io_count
        else
          avg_record["ioinfo"][device]["w_await"] = 0.0
        end
      end
    end

    avg_record
  end

  def show_summary(logfile, summary_title)
    records = read_logfile(logfile)
    summary = make_summary(records)
    accum = make_accumulation(records)

    duration = records.last["time"] - records.first["time"]

    if @json
      output = Hash.new

      output["duration"] = duration
      if summary
        if summary["cpuinfo"]
          output["cpuinfo"] = summary["cpuinfo"]
        end

        if summary['ioinfo']
          output["ioinfo"] = summary["ioinfo"]
        end
      end

      puts output.to_json
    else
      if $stdout.tty? && @pager
        output_file = IO.popen(@pager, "w")
      else
        output_file = $stdout
      end

      output_file.puts("")
      output_file.puts("== Performance summary of '#{summary_title}' ==")
      output_file.puts("")
      duration_str = sprintf("%.2f", duration)
      output_file.puts("record duration: #{duration_str} sec")

      if summary.nil?
        output_file.puts("")
        output_file.puts("No performance info was collected.")
        output_file.puts("This is because command execution time was too short, or something went wrong.")
      end

      if summary && summary["cpuinfo"]
        nr_cpu = records.first["cpuinfo"]["nr_cpu"]

        usr, sys, iowait, irq, soft, idle =
          [summary['cpuinfo']['all']['usr'] + summary['cpuinfo']['all']['nice'],
           summary['cpuinfo']['all']['sys'],
           summary['cpuinfo']['all']['iowait'],
           summary['cpuinfo']['all']['irq'],
           summary['cpuinfo']['all']['soft'],
           summary['cpuinfo']['all']['idle']].map do |val|
          val * nr_cpu
        end

        other = [100.0 - (usr + sys + iowait + irq + soft + idle), 0.0].max * nr_cpu

        usr_str, sys_str, iowait_str, irq_str, soft_str, other_str, idle_str =
          [usr, sys, iowait, irq, soft, other, idle].map do |value|
          sprintf("%.2f", value)
        end

        total_non_idle_str = sprintf("%.2f", usr + sys + irq + soft + other)
        total_idle_str = sprintf("%.2f", iowait + idle)

        output_file.puts("")
        output_file.puts <<EOS
* Average CPU usage (MAX: #{100 * nr_cpu} %)
  * Non idle portion: #{total_non_idle_str}
       %usr: #{usr_str}
       %sys: #{sys_str}
       %irq: #{irq_str}
      %soft: #{soft_str}
     %other: #{other_str}
  * Idle portion: #{total_idle_str}
    %iowait: #{iowait_str}
      %idle: #{idle_str}
EOS
      end

      if summary && summary['ioinfo']
        total_r_iops, total_w_iops, total_r_sec, total_w_sec = [0.0] * 4

        summary['ioinfo']['devices'].each do |device|
          r_iops, w_iops, r_sec, w_sec, r_await, w_await =
            [summary['ioinfo'][device]['riops'],
             summary['ioinfo'][device]['wiops'],
             summary['ioinfo'][device]['rsecps'] * 512 / 1024.0 / 1024.0,
             summary['ioinfo'][device]['wsecps'] * 512 / 1024.0 / 1024.0,
             summary['ioinfo'][device]['r_await'],
             summary['ioinfo'][device]['w_await']]

          total_r_iops += r_iops
          total_w_iops += w_iops
          total_r_sec  += r_sec
          total_w_sec  += w_sec

          r_iops_str, w_iops_str, r_sec_str, w_sec_str = [r_iops, w_iops, r_sec, w_sec].map do |value|
            sprintf("%.2f", value)
          end

          r_await_str, w_await_str = [r_await, w_await].map do |await|
            if await < 1.0
              sprintf("%.1f usec", await * 1000)
            else
              sprintf("%.2f msec", await)
            end
          end

          total_r_bytes_str, total_w_bytes_str = ["read_bytes", "write_bytes"].map do |key|
            bytes = accum["ioinfo"][device][key]
            if bytes > 2**30
              sprintf("%.2f GB", bytes / 2**30)
            elsif bytes > 2**20
              sprintf("%.2f MB", bytes / 2**20)
            elsif bytes > 2**10
              sprintf("%.2f KB", bytes / 2**10)
            else
              sprintf("%.2f bytes", bytes)
            end
          end


          output_file.puts("")
          output_file.puts("* Average DEVICE usage: #{device}")
          output_file.puts("        read IOPS: #{r_iops_str}")
          output_file.puts("       write IOPS: #{w_iops_str}")
          output_file.puts("  read throughput: #{r_sec_str} MB/s")
          output_file.puts(" write throughput: #{w_sec_str} MB/s")
          output_file.puts("     read latency: #{r_await_str}")
          output_file.puts("    write latency: #{w_await_str}")
          output_file.puts("      read amount: #{total_r_bytes_str}")
          output_file.puts("     write amount: #{total_w_bytes_str}")
        end

        if summary['ioinfo']['devices'].size > 1
          total_r_iops_str, total_w_iops_str, total_r_sec_str, total_w_sec_str =
            [total_r_iops, total_w_iops, total_r_sec, total_w_sec].map do |value|
            sprintf("%.2f", value)
          end

          output_file.puts("")
          output_file.puts("* TOTAL DEVICE usage: #{summary['ioinfo']['devices'].join(', ')}")
          output_file.puts("        read IOPS: #{total_r_iops_str}")
          output_file.puts("       write IOPS: #{total_w_iops_str}")
          output_file.puts("  read throughput: #{total_r_sec_str} MB/s")
          output_file.puts(" write throughput: #{total_w_sec_str} MB/s")
        end

        output_file.puts("")
      end

      if output_file != $stdout
        output_file.close
      end
    end
  end
end

end # module Command
end # module PerfMonger
