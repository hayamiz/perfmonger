
require 'optparse'
require 'json'
require 'tempfile'
require 'tmpdir'

module PerfMonger
module Command

class PlotCommand < BaseCommand
  register_command 'plot'

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger plot [options] LOG_FILE

Options:
EOS

    @data_file = nil
    @offset_time = 0.0
    @output_dir = Dir.pwd
    @output_type = 'png'
    @output_prefix = ''
    @save_gpfiles = true
  end

  def parse_args(argv)
    @parser.on('--offset-time TIME') do |time|
      @offset_time = Float(time)
    end

    @parser.on('-o', '--output-dir DIR') do |dir|
      unless File.directory?(dir)
        puts("ERROR: no such directory: #{dir}")
        puts(@parser.help)
        exit(false)
      end

      @output_dir = dir
    end

    @parser.on('-T', '--output-type TYPE', 'Available: eps, png') do |typ|
      unless ['eps', 'png'].include?(typ)
        puts("ERROR: non supported image type: #{typ}")
        puts(@parser.help)
        exit(false)
      end

      @output_type = typ
    end

    @parser.on('-p', '--prefix PREFIX',
               'Output file name prefix.') do |prefix|
      if ! (prefix =~ /-\Z/)
        prefix += '-'
      end

      @output_prefix = prefix
    end

    @parser.on('-s', '--save',
               'Save GNUPLOT and data files.') do
      @save_gpfiles = true
    end


    @parser.parse!(argv)

    if argv.size == 0
      puts("ERROR: PerfMonger log file is required")
      puts(@parser.help)
      exit(false)
    end


    @data_file = File.expand_path(argv.shift)
  end

  def run(argv)
    parse_args(argv)

    plot_ioinfo()
    plot_cpuinfo()
  end

  private
  def plot_ioinfo()
    eps_filename = @output_prefix + 'read-iops.eps'
    gp_filename  = @output_prefix + 'read-iops.gp'
    dat_filename = @output_prefix + 'read-iops.dat'
    if @output_type != 'eps'
      img_filename = @output_prefix + 'read-iops.' + @output_type
    else
      img_filename = nil
    end

    Dir.mktmpdir do |working_dir|
      Dir.chdir(working_dir) do
        datafile = File.open(dat_filename, 'w')
        gpfile = File.new(gp_filename, 'w')

        start_time = nil
        devices = nil

        File.open(@data_file).each_line do |line|
          record = JSON.parse(line)
          time = record["time"]
          ioinfo = record["ioinfo"]
          return unless ioinfo

          start_time ||= time
          devices ||= ioinfo["devices"]

          datafile.puts([time - start_time,
                         devices.map{|device|
                           ioinfo[device]["r/s"]
                         }].flatten.map(&:to_s).join("\t"))
        end

        datafile.close

        col_idx = 2
        plot_stmt_list = devices.map do |device|
          plot_stmt = "\"#{dat_filename}\" usi 1:#{col_idx} with lines title \"#{device}\""
          col_idx += 1
          plot_stmt
        end

        gpfile.puts <<EOS
set term postscript enhanced color
set title "Read IOPS: #{@data_file}"
set size 1.0, 1.0
set output "#{eps_filename}"

set xlabel "elapsed time [sec]"
set ylabel "IOPS"

set grid
set xrange [#{@offset_time}:*]
set yrange [0:*]

plot #{plot_stmt_list.join(",\\\n     ")}
EOS

        gpfile.close

        system("gnuplot #{gpfile.path}")

        if @output_type != 'eps'
          system("convert -rotate 90 -background white #{eps_filename} #{img_filename}")
        end

        FileUtils.copy(eps_filename, @output_dir)
        FileUtils.copy(img_filename, @output_dir) if img_filename
        if @save_gpfiles
          FileUtils.copy(gp_filename , @output_dir)
          FileUtils.copy(dat_filename, @output_dir)
        end
      end
    end
  end

  def plot_cpuinfo()
    eps_filename = @output_prefix + 'cpu.eps'
    gp_filename  = @output_prefix + 'cpu.gp'
    dat_filename = @output_prefix + 'cpu.dat'
    if @output_type != 'eps'
      img_filename = @output_prefix + 'cpu.' + @output_type
    else
      img_filename = nil
    end

    Dir.mktmpdir do |working_dir|
      Dir.chdir(working_dir) do
        datafile = File.open(dat_filename, 'w')
        gpfile = File.open(gp_filename, 'w')

        start_time = nil
        devices = nil
        nr_cpu = nil

        File.open(@data_file).each_line do |line|
          record = JSON.parse(line)
          time = record["time"]
          cpuinfo = record["cpuinfo"]
          return unless cpuinfo
          nr_cpu = cpuinfo['nr_cpu']

          cores = cpuinfo['cpus']

          start_time ||= time

          datafile.puts([time - start_time,
                         %w|%usr %nice %sys %iowait %irq %soft %steal %guest %idle|.map do |key|
                           cores.map{|core| core[key]}.inject(&:+)
                         end].flatten.map(&:to_s).join("\t"))
        end

        datafile.close

        col_idx = 2
        columns = []
        plot_stmt_list = []
        %w|%usr %nice %sys %iowait %irq %soft %steal %guest|.each do |key|
          columns << col_idx
          plot_stmt = "\"#{datafile.path}\" usi 1:(#{columns.map{|i| "$#{i}"}.join("+")}) with filledcurve x1 lw 0 lc #{col_idx - 1} title \"#{key}\""
          plot_stmt_list << plot_stmt
          col_idx += 1
        end

        eps_file = File.join(@output_dir, "cpu.eps")
        gpfile.puts <<EOS
set term postscript enhanced color
set title "CPU usage: #{@data_file} (max: #{nr_cpu*100}%)"
set output "#{eps_filename}"
set key outside center bottom horizontal
set size 1.0, 1.0

set xlabel "elapsed time [sec]"
set ylabel "CPU usage"

set grid
set xrange [#{@offset_time}:*]
set yrange [0:*]

plot #{plot_stmt_list.reverse.join(",\\\n     ")}
EOS

        gpfile.close
        system("gnuplot #{gpfile.path}")

        if @output_type != 'eps'
          system("convert -rotate 90 -background white #{eps_filename} #{img_filename}")
        end

        FileUtils.copy(eps_filename, @output_dir)
        FileUtils.copy(img_filename, @output_dir) if img_filename
        if @save_gpfiles
          FileUtils.copy(gp_filename , @output_dir)
          FileUtils.copy(dat_filename, @output_dir)
        end
      end
    end
  end
end

end # module Command
end # module PerfMonger
