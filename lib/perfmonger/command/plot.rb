
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
    @disk_plot_read = true
    @disk_plot_write = true
    @disk_numkey_threshold = 10
    @plot_iops_max = nil
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

    @parser.on('--plot-read-only', "Plot only READ performance for disks") do
      @disk_plot_read = true
      @disk_plot_write = false
    end

    @parser.on('--plot-write-only', "Plot only WRITE performance for disks") do
      @disk_plot_read = false
      @disk_plot_write = true
    end

    @parser.on('--plot-read-write', "Plot READ and WRITE performance for disks") do
      @disk_plot_read = true
      @disk_plot_write = true
    end

    @parser.on('--plot-numkey-threshold NUM', "Legends of per-disk plots are turned off if the number of disks is larger than this value.") do |num|
      @disk_numkey_threshold = num.to_i
    end

    @parser.on('--plot-iops-max IOPS', "Maximum of IOPS plot range (default: auto)") do |iops|
      @plot_iops_max = iops.to_f
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

    formatter_bin = ::PerfMonger::Command::CoreFinder.plot_formatter()

    @tmpdir = Dir.mktmpdir

    @disk_dat = File.expand_path("disk.dat", @tmpdir)
    @cpu_dat = File.expand_path("cpu.dat", @tmpdir)

    meta_json = nil
    IO.popen([formatter_bin, "-perfmonger", @data_file, "-cpufile", @cpu_dat, "-diskfile", @disk_dat], "r") do |io|
      meta_json = io.read
    end
    if $?.exitstatus != 0
      puts("ERROR: failed to run perfmonger-plot-formatter")
      exit(false)
    end
    meta = JSON.parse(meta_json)

    plot_disk(meta)
    plot_cpu(meta)

    true
  end

  private
  def plot_disk(meta)
    iops_pdf_filename = @output_prefix + 'disk-iops.pdf'
    transfer_pdf_filename = @output_prefix + 'disk-transfer.pdf'
    total_iops_pdf_filename = @output_prefix + 'disk-total-iops.pdf'
    total_transfer_pdf_filename = @output_prefix + 'disk-total-transfer.pdf'
    gp_filename  = @output_prefix + 'disk.gp'
    dat_filename = @output_prefix + 'disk.dat'
    if @output_type != 'pdf'
      iops_img_filename = @output_prefix + 'disk-iops.' + @output_type
      transfer_img_filename = @output_prefix + 'disk-transfer.' + @output_type
      total_iops_img_filename = @output_prefix + 'disk-total-iops.' + @output_type
      total_transfer_img_filename = @output_prefix + 'disk-total-transfer.' + @output_type
    else
      iops_img_filename = nil
      transfer_img_filename = nil
      total_iops_img_filename = nil
      total_transfer_img_filename = nil
    end

    start_time = meta["start_time"]
    end_time = meta["end_time"]

    Dir.chdir(@tmpdir) do
      gpfile = File.new(gp_filename, 'w')

      total_iops_plot_stmt_list = []
      iops_plot_stmt_list = meta["disk"]["devices"].map do |dev_entry|
        devname = dev_entry["name"]
        idx = dev_entry["idx"]

        if devname == "total"
          if @disk_plot_read
            total_iops_plot_stmt_list.push("\"disk.dat\" ind #{idx} usi 1:2 with lines lw 2 title \"#{devname} read\"")
          end
          if @disk_plot_write
            total_iops_plot_stmt_list.push("\"disk.dat\" ind #{idx} usi 1:3 with lines lw 2 title \"#{devname} write\"")
          end

          []
        elsif @disk_only_regex && !(devname =~ @disk_only_regex)
          []
        else
          plot_stmt = []

          if @disk_plot_read
            plot_stmt.push("\"disk.dat\" ind #{idx} usi 1:2 with lines lw 2 title \"#{devname} read\"")
          end
          if @disk_plot_write
            plot_stmt.push("\"disk.dat\" ind #{idx} usi 1:3 with lines lw 2 title \"#{devname} write\"")
          end

          plot_stmt
        end
      end.flatten

      total_transfer_plot_stmt_list = []
      transfer_plot_stmt_list = meta["disk"]["devices"].map do |dev_entry|
        devname = dev_entry["name"]
        idx = dev_entry["idx"]

        if devname == "total"
          if @disk_plot_read
            total_transfer_plot_stmt_list.push("\"disk.dat\" ind #{idx} usi 1:4 with lines lw 2 title \"#{devname} read\"")
          end
          if @disk_plot_write
            total_transfer_plot_stmt_list.push("\"disk.dat\" ind #{idx} usi 1:5 with lines lw 2 title \"#{devname} write\"")
          end

          []
        elsif @disk_only_regex && !(devname =~ @disk_only_regex)
          []
        else
          plot_stmt = []

          if @disk_plot_read
            plot_stmt.push("\"disk.dat\" ind #{idx} usi 1:4 with lines lw 2 title \"#{devname} read\"")
          end
          if @disk_plot_write
            plot_stmt.push("\"disk.dat\" ind #{idx} usi 1:5 with lines lw 2 title \"#{devname} write\"")
          end

          plot_stmt
        end
      end.flatten

      if iops_plot_stmt_list.size == 0
        puts("No plot target disk devices.")
        return
      end

      num_dev = meta["disk"]["devices"].select do |dev_entry|
        dev_entry["name"] != "total"
      end.size

      if num_dev > @disk_numkey_threshold
        set_key_stmt = "unset key"
      else
        set_key_stmt = "set key below center"
      end

      iops_yrange = "set yrange [0:*]"
      if @plot_iops_max
        iops_yrange = "set yrange [0:#{@plot_iops_max}]"
      end

      gpfile.puts <<EOS
set term pdfcairo enhanced color size 6in,2.5in
set title "IOPS"
set size 1.0, 1.0
set output "#{iops_pdf_filename}"

set xlabel "elapsed time [sec]"
set ylabel "IOPS"

set grid
set xrange [#{@offset_time}:#{end_time - start_time}]
#{iops_yrange}

#{set_key_stmt}
plot #{iops_plot_stmt_list.join(",\\\n     ")}

set title "Total IOPS"
unset key
set output "#{total_iops_pdf_filename}"
plot #{total_iops_plot_stmt_list.join(",\\\n     ")}


set title "Transfer rate"
set output "#{transfer_pdf_filename}"
set ylabel "transfer rate [MB/s]"
set yrange [0:*]
#{set_key_stmt}
plot #{transfer_plot_stmt_list.join(",\\\n     ")}

set title "Total transfer rate"
set output "#{total_transfer_pdf_filename}"
unset key
plot #{total_transfer_plot_stmt_list.join(",\\\n     ")}
EOS
      gpfile.close

      system("gnuplot #{gpfile.path}")

      if @output_type != 'pdf'
        system("convert -density 150 -background white #{iops_pdf_filename} #{iops_img_filename}")
        system("convert -density 150 -background white #{transfer_pdf_filename} #{transfer_img_filename}")
        system("convert -density 150 -background white #{total_iops_pdf_filename} #{total_iops_img_filename}")
        system("convert -density 150 -background white #{total_transfer_pdf_filename} #{total_transfer_img_filename}")
      end

    end # chdir

    copy_targets = []
    copy_targets += [iops_pdf_filename, transfer_pdf_filename]
    copy_targets += [total_iops_pdf_filename, total_transfer_pdf_filename]
    copy_targets.push(iops_img_filename) if iops_img_filename
    copy_targets.push(transfer_img_filename) if transfer_img_filename
    copy_targets.push(total_iops_img_filename) if total_iops_img_filename
    copy_targets.push(total_transfer_img_filename) if total_transfer_img_filename

    if @save_gpfiles
      copy_targets.push(dat_filename)
      copy_targets.push(gp_filename)
    end

    copy_targets.each do |target|
      FileUtils.copy(File.join(@tmpdir, target), @output_dir)
    end
  end # def

  def plot_cpu(meta)
    pdf_filename = @output_prefix + 'cpu.pdf'
    gp_filename  = @output_prefix + 'cpu.gp'
    dat_filename = @output_prefix + 'cpu.dat'

    all_pdf_filename = @output_prefix + 'allcpu.pdf'
    all_gp_filename  = @output_prefix + 'allcpu.gp'

    if @output_type != 'pdf'
      img_filename = @output_prefix + 'cpu.' + @output_type
      all_img_filename = @output_prefix + 'allcpu.' + @output_type
    else
      img_filename = nil
      all_img_filename = nil
    end

    start_time = meta["start_time"]
    end_time = meta["end_time"]

    Dir.chdir(@tmpdir) do
      gpfile = File.open(gp_filename, 'w')
      all_gpfile = File.open(all_gp_filename, 'w')

      devices = nil
      nr_cpu = meta["cpu"]["num_core"]

      plot_stmt_list = []
      %w|%usr %nice %sys %iowait %hardirq %softirq %steal %guest|.each_with_index do |key, idx|
        stack_columns = (0..idx).to_a.map{|x| x + 2}
        plot_stmt = "\"cpu.dat\" ind 0 usi 1:(#{stack_columns.map{|i| "$#{i}"}.join("+")}) with filledcurve x1 lw 0 lc #{idx+1} title \"#{key}\""
        plot_stmt_list << plot_stmt
      end

      pdf_file = File.join(@output_dir, "cpu.pdf")
      gpfile.puts <<EOS
set term pdfcairo enhanced color size 6in,2.5in
set title "CPU usage (max: #{nr_cpu*100}%)"
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
      nr_cols = 1
      nr_rows = nr_cpu / nr_cols

      plot_height = ([nr_cpu, 8].min) + ([nr_cpu - 8, 0].max) * 0.5

      if nr_rows == 1
        plot_height /= 2.0
      end

      all_gpfile.puts <<EOS
set term pdfcairo color enhanced size 8.5inch, #{plot_height}inch
set output "#{all_pdf_filename}"
set size 1.0, 1.0
set multiplot
set grid
set xrange [#{@offset_time}:#{end_time - start_time}]
set yrange [0:101]

EOS

      legend_height = 0.04
      nr_cpu.times do |cpu_idx|
        xpos = (1.0 / nr_cols) * (cpu_idx % nr_cols)
        ypos = ((1.0 - legend_height) / nr_rows) * (nr_rows - 1 - (cpu_idx / nr_cols).to_i) + legend_height

        plot_stmt_list = []
        %w|%usr %nice %sys %iowait %hardirq %softirq %steal %guest|.each_with_index do |key, idx|
          stack_columns = (0..idx).to_a.map{|x| x + 2}
          plot_stmt = "\"cpu.dat\" ind #{cpu_idx+1} usi 1:(#{stack_columns.map{|i| "$#{i}"}.join("+")}) with filledcurve x1 lw 0 lc #{idx+1} title \"#{key}\""
          plot_stmt_list << plot_stmt
        end

        all_gpfile.puts <<EOS
set title 'cpu #{cpu_idx}' offset -61,-3 font 'Arial,16'
unset key
set origin #{xpos}, #{ypos}
set size #{1.0/nr_cols}, #{(1.0 - legend_height)/nr_rows}
set rmargin 2
set lmargin 12
set tmargin 0.5
set bmargin 0.5
set xtics offset 0.0,0.5
set ytics offset 0.5,0
set style fill noborder
plot #{plot_stmt_list.reverse.join(",\\\n     ")}

EOS
      end # times

      # plot legends
      plot_stmt_list = []
      %w|%usr %nice %sys %iowait %hardirq %softirq %steal %guest|.each_with_index do |key, idx|
        plot_stmt = "-1 with filledcurve x1 lw 0 lc #{idx+1} title \"#{key}\""
        plot_stmt_list << plot_stmt
      end
      all_gpfile.puts <<EOS
unset title
set key center center horizontal font "Arial,14"
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

set xlabel "elapsed time [sec]"
plot #{plot_stmt_list.reverse.join(",\\\n     ")}

EOS

      all_gpfile.fsync
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
    end

    copy_targets.each do |target|
      FileUtils.copy(File.join(@tmpdir, target), @output_dir)
    end
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
