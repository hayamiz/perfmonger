
require 'optparse'
require 'json'
require 'tempfile'
require 'tmpdir'

module PerfMonger
module Command

class PlotCommand < BaseCommand
  register_command 'plot', "Plot system performance graphs collected by 'record'"

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger plot [options] LOG_FILE

Options:
EOS

    @data_file = nil
    @offset_time = 0.0
    @output_dir = Dir.pwd
    @output_type = 'pdf'
    @output_prefix = ''
    @save_gpfiles = false
    @disk_only_regex = nil
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

    @parser.on('-T', '--output-type TYPE', 'Available: pdf, png') do |typ|
      unless ['pdf', 'png'].include?(typ)
        puts("ERROR: non supported image type: #{typ}")
        puts(@parser.help)
        exit(false)
      end

      if typ != 'pdf' && ! system('which convert >/dev/null 2>&1')
        puts("ERROR: convert(1) not found.")
        puts("ERROR: ImageMagick is required for #{typ}")
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

    @parser.on('--disk-only REGEX', "Select disk devices that matches REGEX") do |regex|
      @disk_only_regex = Regexp.compile(regex)
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
    unless system('which gnuplot >/dev/null 2>&1')
      puts("ERROR: gnuplot not found")
      puts(@parser.help)
      exit(false)
    end

    unless system('gnuplot -e "set terminal" < /dev/null 2>&1 | grep pdfcairo >/dev/null 2>&1')
      puts("ERROR: pdfcairo is not supported by installed gnuplot")
      puts("ERROR: PerfMonger requires pdfcairo-supported gnuplot")
      puts(@parser.help)
      exit(false)
    end

    player_bin = ::PerfMonger::Command::CoreFinder.player()

    tmpfile = Tempfile.new("jsondata")
    IO.popen([player_bin, @data_file], "r").each_line do |line|
      tmpfile.print(line)
    end
    tmpfile.flush()

    plot_ioinfo(tmpfile.path)
    plot_cpuinfo(tmpfile.path)
  end

  private
  def plot_ioinfo(json_file)
    iops_pdf_filename = @output_prefix + 'iops.pdf'
    transfer_pdf_filename = @output_prefix + 'transfer.pdf'
    gp_filename  = @output_prefix + 'io.gp'
    dat_filename = @output_prefix + 'io.dat'
    if @output_type != 'pdf'
      iops_img_filename = @output_prefix + 'iops.' + @output_type
      transfer_img_filename = @output_prefix + 'transfer.' + @output_type
    else
      iops_img_filename = nil
      transfer_img_filename = nil
    end

    Dir.mktmpdir do |working_dir|
      Dir.chdir(working_dir) do
        datafile = File.open(dat_filename, 'w')
        gpfile = File.new(gp_filename, 'w')

        start_time = nil
        devices = nil

        File.open(json_file).each_line do |line|
          record = JSON.parse(line)
          time = record["time"]
          diskinfo = record["disk"]
          return unless diskinfo

          start_time ||= time
          devices ||= diskinfo["devices"]

          if @disk_only_regex
            devices = devices.select do |devname|
              devname =~ @disk_only_regex
            end
          end

          datafile.puts([time - start_time,
                         devices.map{|device|
                           [diskinfo[device]["riops"], diskinfo[device]["wiops"],
                            diskinfo[device]["rkbyteps"] * 512 / 1024 / 1024, # in MB/s
                            diskinfo[device]["wkbyteps"] * 512 / 1024 / 1024, # in MB/s
                           ]
                         }].flatten.map(&:to_s).join("\t"))
        end

        datafile.close

        col_idx = 2
        iops_plot_stmt_list = devices.map do |device|
          plot_stmt = []
          plot_stmt.push("\"#{dat_filename}\" usi 1:#{col_idx} with lines lw 2 title \"#{device} read\"")
          plot_stmt.push("\"#{dat_filename}\" usi 1:#{col_idx + 1} with lines lw 2 title \"#{device} write\"")
          col_idx += 4
          plot_stmt
        end.flatten

        col_idx = 4
        transfer_plot_stmt_list = devices.map do |device|
          plot_stmt = []
          plot_stmt.push("\"#{dat_filename}\" usi 1:#{col_idx} with lines lw 2 title \"#{device} read\"")
          plot_stmt.push("\"#{dat_filename}\" usi 1:#{col_idx + 1} with lines lw 2 title \"#{device} write\"")
          col_idx += 4
          plot_stmt
        end.flatten

        gpfile.puts <<EOS
set term pdfcairo enhanced color
set title "IOPS: #{File.basename(@data_file)}"
set size 1.0, 1.0
set output "#{iops_pdf_filename}"

set xlabel "elapsed time [sec]"
set ylabel "IOPS"

set grid
set xrange [#{@offset_time}:*]
set yrange [0:*]

set key below center

plot #{iops_plot_stmt_list.join(",\\\n     ")}


set title "Transfer rate: #{File.basename(@data_file)}"
set output "#{transfer_pdf_filename}"
set ylabel "transfer rate [MB/s]"
plot #{transfer_plot_stmt_list.join(",\\\n     ")}
EOS

        gpfile.close

        system("gnuplot #{gpfile.path}")

        if @output_type != 'pdf'
          system("convert -density 150 -background white #{iops_pdf_filename} #{iops_img_filename}")
          system("convert -density 150 -background white #{transfer_pdf_filename} #{transfer_img_filename}")
        end

      end # chdir

      copy_targets = [iops_pdf_filename, transfer_pdf_filename]
      copy_targets.push(iops_img_filename) if iops_img_filename
      copy_targets.push(transfer_img_filename) if transfer_img_filename

      if @save_gpfiles
        copy_targets.push(dat_filename)
        copy_targets.push(gp_filename)
      end

      copy_targets.each do |target|
        FileUtils.copy(File.join(working_dir, target), @output_dir)
      end
    end # mktempdir
  end # def

  def plot_cpuinfo(json_file)
    pdf_filename = @output_prefix + 'cpu.pdf'
    gp_filename  = @output_prefix + 'cpu.gp'
    dat_filename = @output_prefix + 'cpu.dat'

    all_pdf_filename = @output_prefix + 'allcpu.pdf'
    all_gp_filename  = @output_prefix + 'allcpu.gp'
    all_dat_filename = @output_prefix + 'allcpu.dat'

    if @output_type != 'pdf'
      img_filename = @output_prefix + 'cpu.' + @output_type
      all_img_filename = @output_prefix + 'allcpu.' + @output_type
    else
      img_filename = nil
      all_img_filename = nil
    end

    Dir.mktmpdir do |working_dir|
      Dir.chdir(working_dir) do
        datafile = File.open(dat_filename, 'w')
        gpfile = File.open(gp_filename, 'w')
        all_datafile = File.open(all_dat_filename, 'w')
        all_gpfile = File.open(all_gp_filename, 'w')

        start_time = nil
        end_time = 0
        devices = nil
        nr_cpu = nil

        File.open(json_file).each_line do |line|
          record = JSON.parse(line)

          time = record["time"]
          cpuinfo = record["cpu"]
          return unless cpuinfo
          nr_cpu = cpuinfo['num_core']

          cores = cpuinfo['cores']

          start_time ||= time
          end_time = [end_time, time].max

          datafile.puts([time - start_time,
                         %w|usr nice sys iowait hardirq softirq steal guest idle|.map do |key|
                           cores.map{|core| core[key]}.inject(&:+)
                         end].flatten.map(&:to_s).join("\t"))
        end
        datafile.close

        col_idx = 2
        columns = []
        plot_stmt_list = []
        %w|%usr %nice %sys %iowait %hardirq %softirq %steal %guest|.each do |key|
          columns << col_idx
          plot_stmt = "\"#{datafile.path}\" usi 1:(#{columns.map{|i| "$#{i}"}.join("+")}) with filledcurve x1 lw 0 lc #{col_idx - 1} title \"#{key}\""
          plot_stmt_list << plot_stmt
          col_idx += 1
        end

        pdf_file = File.join(@output_dir, "cpu.pdf")
        gpfile.puts <<EOS
set term pdfcairo enhanced color
set title "CPU usage: #{File.basename(@data_file)} (max: #{nr_cpu*100}%)"
set output "#{pdf_filename}"
set key outside center bottom horizontal
set size 1.0, 1.0

set xlabel "elapsed time [sec]"
set ylabel "CPU usage"

set grid
set xrange [#{@offset_time}:#{end_time - start_time}]
set yrange [0:*]

plot #{plot_stmt_list.reverse.join(",\\\n     ")}
EOS

        gpfile.close
        system("gnuplot #{gpfile.path}")

        if @output_type != 'pdf'
          system("convert -density 150 -background white #{pdf_filename} #{img_filename}")
        end

        ## Plot all CPUs in a single file

        nr_cpu_factors = factors(nr_cpu)
        nr_cols = nr_cpu_factors.select do |x|
          x <= Math.sqrt(nr_cpu)
        end.max
        nr_cols ||= Math.sqrt(nr_cpu).ceil
        nr_rows = nr_cpu / nr_cols

        all_gpfile.puts <<EOS
set term pdfcairo color enhanced size 8.5inch, 11inch
set output "#{all_pdf_filename}"
set size 1.0, 1.0
set multiplot
set grid
set xrange [#{@offset_time}:#{end_time - start_time}]
set yrange [0:101]

EOS

        legend_height = 0.04
        nr_cpu.times do |cpu_idx|
          all_datafile.puts("# cpu #{cpu_idx}")
          File.open(json_file).each_line do |line|
            record = JSON.parse(line)
            time = record["time"]
            cpurec = record["cpu"]["cores"][cpu_idx]
            all_datafile.puts([time - start_time,
                               cpurec["usr"] + cpurec["nice"],
                               cpurec["sys"],
                               cpurec["hardirq"],
                               cpurec["softirq"],
                               cpurec["steal"] + cpurec["guest"],
                               cpurec["iowait"]].map(&:to_s).join("\t"))
          end
          all_datafile.puts("")
          all_datafile.puts("")

          xpos = (1.0 / nr_cols) * (cpu_idx % nr_cols)
          ypos = ((1.0 - legend_height) / nr_rows) * (nr_rows - 1 - (cpu_idx / nr_cols).to_i) + legend_height

          all_gpfile.puts <<EOS
set title 'cpu #{cpu_idx}' offset 0.0,-0.7 font 'Arial,16'
unset key
set origin #{xpos}, #{ypos}
set size #{1.0/nr_cols}, #{(1.0 - legend_height)/nr_rows}
set rmargin 0.5
set lmargin 3.5
set tmargin 1.3
set bmargin 1.3
set xtics offset 0.0,0.5
set ytics offset 0.5,0
set style fill noborder
plot '#{all_datafile.path}' index #{cpu_idx} using 1:($2+$3+$4+$5+$6+$7) with filledcurve x1 lw 0 lc 6 title '%iowait', \\
     '#{all_datafile.path}' index #{cpu_idx} using 1:($2+$3+$4+$5+$6) with filledcurve x1 lw 0 lc 5 title '%other', \\
     '#{all_datafile.path}' index #{cpu_idx} using 1:($2+$3+$4+$5) with filledcurve x1 lw 0 lc 4 title '%soft', \\
     '#{all_datafile.path}' index #{cpu_idx} using 1:($2+$3+$4) with filledcurve x1 lw 0 lc 3 title '%irq', \\
     '#{all_datafile.path}' index #{cpu_idx} using 1:($2+$3) with filledcurve x1 lw 0 lc 2 title '%sys', \\
     '#{all_datafile.path}' index #{cpu_idx} using 1:2 with filledcurve x1 lw 0 lc 1 title '%usr'

EOS

        end

        all_gpfile.puts <<EOS
unset title
set key center center horizontal font "Arial,16"
set origin 0.0, 0.0
set size 1.0, #{legend_height}
set rmargin 0
set lmargin 0
set tmargin 0
set bmargin 0
unset tics
set border 0
set yrange [0:1]
# plot -1 with filledcurve x1 title '%usr'

plot -1 with filledcurve x1 lw 0 lc 1 title '%usr', \\
     -1 with filledcurve x1 lw 0 lc 2 title '%sys', \\
     -1 with filledcurve x1 lw 0 lc 3 title '%irq', \\
     -1 with filledcurve x1 lw 0 lc 4 title '%soft', \\
     -1 with filledcurve x1 lw 0 lc 5 title '%other', \\
     -1 with filledcurve x1 lw 0 lc 6 title '%iowait'
EOS

        all_datafile.fsync
        all_gpfile.fsync
        all_datafile.close
        all_gpfile.close

        system("gnuplot #{all_gpfile.path}")

        if @output_type != 'pdf'
          system("convert -density 150 -background white #{all_pdf_filename} #{all_img_filename}")
        end

      end # chdir

      copy_targets = []

      copy_targets << pdf_filename
      copy_targets << img_filename if img_filename
      copy_targets << all_pdf_filename
      copy_targets << all_img_filename if all_img_filename

      if @save_gpfiles
        copy_targets << gp_filename
        copy_targets << dat_filename
        copy_targets << all_gp_filename
        copy_targets << all_dat_filename
      end

      copy_targets.each do |target|
        FileUtils.copy(File.join(working_dir, target), @output_dir)
      end
    end # mktempdir
  end # def

  private
  def factors(n)
    (2..([n, n / 2].max).to_i).select do |x|
      n % x == 0
    end.sort
  end
end

end # module Command
end # module PerfMonger
