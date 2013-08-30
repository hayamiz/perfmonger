
require 'fileutils'

module PerfMonger
module Command

class BukkoCommand < BaseCommand
  register_command 'bukko', 'Gather all possible system config information'

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger bukko [options] OUTPUT_DIR

Options:
EOS

    @output_dir = './bukko'
  end

  def parse_args(argv)
    @parser.parse!(argv)

    if argv.size == 0
      puts("ERROR: output directory is required.")
      puts(@parser.help)
      exit(false)
    end

    @output_dir = argv.shift

    if ! File.directory?(@output_dir)
      FileUtils.mkdir_p(@output_dir)
    end
  end

  def run(argv)
    parse_args(argv)

    ENV['LANG'] = 'C'

    $stderr.puts("System information is gathered into #{@output_dir}")

    ## Collect generic info.

    do_with_message("Saving /proc info") do
      save_proc_info()
    end

    do_with_message("Saving IRQ info") do
      save_irq_info()
    end

    do_with_message("Saving block device info") do
      save_device_info()
    end

    do_with_message("Saving /dev/disk info") do
      save_disk_info()
    end

    do_with_message("Saving PCI/PCIe info") do
      save_pci_info()
    end

    do_with_message("Saving kernel module info") do
      save_module_info()
    end


    ## Collect vendor specific info

    # LSI MegaRAID
    megacli_bin = "/opt/MegaRAID/MegaCli/MegaCli64"
    if File.executable?(megacli_bin) && Process::UID.rid == 0
      do_with_message("Saving MegaRAID settings") do
        save_megaraid_info(megacli_bin)
      end
    end
  end

  private
  def do_with_message(message)
    $stderr.print(message + " ... ")
    $stderr.flush

    @errors = []

    begin
      yield
    rescue StandardError => err
      $stderr.puts("failed")
      @errors.push(err)
    end

    if @errors.empty?
      $stderr.puts("done")
      $stderr.puts("")
    else
      $stderr.puts("failed")
      $stderr.puts("")
      @errors.each do |error|
        $stderr.puts(" ERROR: #{error.message}")
      end
      $stderr.puts("")
    end
  end

  def read_file(src)
    if File.exists?(src)
      begin
        return File.read(src)
      rescue StandardError => err
        @errors.push(err)
      end
    end

    nil
  end

  def copy_file(src, dest)
    content = read_file(src)

    if content
      File.open(dest, "w") do |f|
        f.print(content)
      end
    end
  end

  def save_proc_info()
    ["cpuinfo", "meminfo", "mdstat", "mounts", "interrupts",
     "diskstats", "partitions", "ioports",
    ].each do |entry|
      copy_file("/proc/#{entry}", "#{@output_dir}/proc-#{entry}.log")
    end

    copy_file('/proc/scsi/scsi',  "#{@output_dir}/proc-scsi.log")

    File.open("#{@output_dir}/proc-sys-fs.log", "w") do |f|
      Dir.glob("/proc/sys/fs/*").each do |path|
        next unless File.file?(path)
        begin
          content = File.read(path)
        rescue Errno::EACCES => err
          @errors.push(err)
          f.puts("## #{path}")
          f.puts("permission denied")
          f.puts("")
          next
        rescue StandardError => err
          @errors.push(err)
          next
        end
        f.puts("## #{path}")
        f.puts(content)
        f.puts("")
      end
    end
  end

  def save_irq_info()
    File.open("#{@output_dir}/irq-smp-affinity.log", "w") do |f|
      Dir.glob('/proc/irq/*/smp_affinity').sort_by do |path|
        irqno = File.basename(File.dirname(path)).to_i
      end.each do |path|
        f.puts("## cat #{path}")
        f.puts(`cat #{path}`)
        f.puts("")
      end
    end
  end

  def save_device_info()
    Dir.glob('/sys/block/sd*').each do |sd_dev|
      File.open("#{@output_dir}/block-#{File.basename(sd_dev)}.log", "w") do |f|
        f.puts("## ls -l #{sd_dev}")
        f.puts(`ls -l #{sd_dev}`)
        f.puts("")
        ['device/queue_depth',
         'device/queue_type',
         'device/iorequest_cnt',
         'device/vendor',
         'queue/scheduler',
         'queue/nr_requests',
         'queue/rq_affinity',
         'queue/nomerges',
         'queue/add_random',
         'queue/rotational',
         'queue/max_hw_sectors_kb',
         'queue/physical_block_size',
         'queue/optimal_io_size',
        ].each do |entity|
          path = "#{sd_dev}/#{entity}"
          if File.exists?(path)
            f.puts("## #{path}")
            f.puts(`cat #{path}`)
            f.puts("")
          end
        end
      end
    end
  end

  def save_disk_info()
    File.open("#{@output_dir}/disk-by-path.log", "w") do |f|
      f.puts(`ls -l /dev/disk/by-path/`)
    end

    File.open("#{@output_dir}/disk-by-uuid.log", "w") do |f|
      f.puts(`ls -l /dev/disk/by-uuid/`)
    end

    File.open("#{@output_dir}/disk-by-id.log", "w") do |f|
      f.puts(`ls -l /dev/disk/by-id/`)
    end

    File.open("#{@output_dir}/disk-multipath.log", "w") do |f|
      f.puts(`/sbin/multipath -ll 2>&1`)
    end
  end

  def save_pci_info()
    # try to find lspci
    dirs = ["/sbin", "/usr/sbin", "/usr/local/sbin", "/usr/bin", "/usr/local/bin"]
    dirs += ENV['PATH'].split(":")

    lspci_bindir = dirs.find do |dir|
      File.executable?(File.expand_path("lspci", dir))
    end

    unless lspci_bindir
      @errors << RuntimeError.new("lspci(1) not found")
    else
      File.open("#{@output_dir}/lspci.log", "w") do |f|
        f.puts(`#{lspci_bindir}/lspci -D -vvv`)
      end
    end

    Dir.glob("/sys/devices/pci*/*/*/vendor") do |vendor|
      pcidir = File.dirname(vendor)

      prefix = [File.basename(File.dirname(File.dirname(pcidir))),
                File.basename(File.dirname(pcidir)),
                File.basename(pcidir)].join("-")

      File.open("#{@output_dir}/#{prefix}.log", "w") do |f|
        f.puts("## ls -l #{pcidir}")
        f.puts(`ls -l #{pcidir}`)
        f.puts("")
        Dir.entries(pcidir).select do |filename|
          ! (["remove", "reset", "rescan", "rom", "uevent", "config",
              "vpd"
             ].include?(filename) ||
             filename =~ /\Aresource\d+\Z/ ||
             filename =~ /\Aresource\d+_wc\Z/ # DDN device specific node (?)
             )
        end.each do |filename|
          path = File.expand_path(filename, pcidir)
          next unless File.file?(path)
          content = read_file(path)
          if content
            f.puts("## #{path}")
            f.puts(content)
            f.puts("")
          end
        end

        msi_irqs_dir = File.expand_path("msi_irqs", pcidir)
        if File.directory?(msi_irqs_dir)
          f.puts("## ls -l #{msi_irqs_dir}")
          f.puts(`ls -l #{msi_irqs_dir}`)
          f.puts("")

          Dir.glob("#{msi_irqs_dir}/*/mode").each do |mode_path|
            content = read_file(mode_path)
            f.puts("## #{mode_path}")
            f.puts(content)
            f.puts("")
          end
        end
      end
    end
  end

  def save_module_info()
    File.open("#{@output_dir}/lsmod.log", "w") do |f|
      f.puts(`/sbin/lsmod`)
    end
  end

  def save_megaraid_info(megacli_bin)
    File.open("#{@output_dir}/megaraid.log", "w") do |f|
      params_list = ["-AdpCount",
                     "-AdpAllinfo -aALL",
                     "-AdpBbuCmd -aALL",
                     "-LDInfo -Lall -aALL",
                     "-PDList -aALL"
                    ].each do |params|
        f.puts("## #{megacli_bin} #{params}")
        f.puts(`#{megacli_bin} #{params}`.gsub(/\r/, ""))
        f.puts("")
      end
    end
  end
end

end # module Command
end # module PerfMonger

