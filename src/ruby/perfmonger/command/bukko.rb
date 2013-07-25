
require 'fileutils'

module PerfMonger
module Command

class BukkoCommand < BaseCommand
  register_command 'bukko'

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

    $stderr.puts("System information is gathered into #{@output_dir}")

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
  end

  private
  def do_with_message(message)
    $stderr.print(message + " ... ")
    $stderr.flush

    begin
      yield
    rescue StandardError => err
      $stderr.puts("failed")
      raise err
    end

    $stderr.puts("done")
  end

  def save_proc_file(src, dest)
    if File.exists?(src)
      File.open(dest, "w") do |f|
        f.print(File.read(src))
      end
    end
  end

  def save_proc_info()
    save_proc_file('/proc/cpuinfo', "#{@output_dir}/proc-cpuinfo.log")
    save_proc_file('/proc/meminfo', "#{@output_dir}/proc-meminfo.log")
    save_proc_file('/proc/mdstat',  "#{@output_dir}/proc-mdstat.log")
    save_proc_file('/proc/mounts',  "#{@output_dir}/proc-mounts.log")
    save_proc_file('/proc/interrupts',  "#{@output_dir}/proc-interrupts.log")
    save_proc_file('/proc/scsi/scsi',  "#{@output_dir}/proc-scsi.log")
  end

  def save_irq_info()
    Dir.glob('/proc/irq/*/smp_affinity').each do |path|
      irqno = File.basename(File.dirname(path))
      save_proc_file(path,  "#{@output_dir}/irq-#{irqno}-smp-affinity.log")
    end
  end

  def save_device_info()
    Dir.glob('/sys/block/sd*').each do |sd_dev|
      File.open("#{@output_dir}/block-#{File.basename(sd_dev)}.log", "w") do |f|
        f.puts("## ls -l #{sd_dev}")
        f.puts(`ls -l #{sd_dev}`)
        f.puts("")
        ['device/queue_depth', 'queue/scheduler', 'queue/nr_requests'].each do |entity|
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

    File.open("#{@output_dir}/disk-multipath.log", "w") do |f|
      f.puts(`/sbin/multipath -ll 2>&1`)
    end
  end

  def save_pci_info()
    File.open("#{@output_dir}/lspci.log", "w") do |f|
      f.puts(`lspci -D -vvv`)
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
        ["irq", "local_cpulist", "local_cpus", "pools", "numa_node", "vendor", "device"].each do |entry|
          path = "#{pcidir}/#{entry}"
          if File.exists?(path)
            f.puts("## #{path}")
            f.puts(`cat #{path}`)
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
end

end # module Command
end # module PerfMonger

