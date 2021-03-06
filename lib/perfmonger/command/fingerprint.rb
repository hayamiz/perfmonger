
require 'fileutils'
require 'tmpdir'
require 'tempfile'

module PerfMonger
module Command

class FingerprintCommand < BaseCommand
  register_command 'fingerprint', 'Gather all possible system config information'
  register_alias   'bukko'
  register_alias   'fp'

  def initialize
    @parser = OptionParser.new
    @parser.banner = <<EOS
Usage: perfmonger fingerprint [options] OUTPUT_TARBALL

Options:
EOS

    hostname = `hostname`.strip
    @output_tarball = "./fingerprint.#{hostname}.tar.gz"
  end

  def parse_args(argv)
    @parser.parse!(argv)

    if argv.size == 0
      puts("ERROR: output directory is required.")
      puts(@parser.help)
      exit(false)
    end

    @output_tarball = argv.shift

    if ! @output_tarball =~ /\.(tar\.gz|tgz)$/
      @output_tarball += ".tar.gz"
    end
  end

  def run(argv)
    parse_args(argv)

    ENV['LANG'] = 'C'

    $stderr.puts("System information is gathered into #{@output_tarball}")

    Dir.mktmpdir do |tmpdir|
      output_basename = File.basename(@output_tarball.gsub(/\.(tar\.gz|tgz)$/, ''))

      @output_dir = File.join(tmpdir, output_basename)
      FileUtils.mkdir(@output_dir)

      ## Collect generic info.
      do_with_message("Saving /proc info") do
        save_proc_info()
      end

      do_with_message("Saving numactl info") do
        save_numactl_info()
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

      do_with_message("Saving fdisk info") do
        save_fdisk_info()
      end

      do_with_message("Saving lsblk info") do
        save_lsblk_info()
      end

      do_with_message("Saving LVM info") do
        save_lvm_info()
      end

      do_with_message("Saving PCI/PCIe info") do
        save_pci_info()
      end

      do_with_message("Saving kernel module info") do
        save_module_info()
      end

      do_with_message("Saving distro info") do
        save_distro_info()
      end

      do_with_message("Saving sysctl info") do
        save_sysctl_info()
      end

      do_with_message("Saving dmidecode info") do
        save_dmidecode_info()
      end

      do_with_message("Saving biosdecode info") do
        save_biosdecode_info()
      end

      do_with_message("Saving nvme info") do
        save_nvme_info()
      end

      ## Collect vendor specific info

      # https://aws.amazon.com/jp/code/ec2-instance-metadata-query-tool/
      if find_executable("ec2-metadata")
        do_with_message("Saving EC2 metadata info") do
          save_ec2metadata_info()
        end
      end

      # LSI MegaRAID
      megacli_bin = "/opt/MegaRAID/MegaCli/MegaCli64"
      if File.executable?(megacli_bin) && Process::UID.rid == 0
        do_with_message("Saving MegaRAID settings") do
          save_megaraid_info(megacli_bin)
        end
      end

      tmptar_path = Tempfile.new("fingerprint").path

      Dir.chdir(tmpdir) do
        if ! system("tar czf '#{tmptar_path}' #{output_basename}")
          raise RuntimeError.new("Failed to execute tar(1)")
        end
      end

      FileUtils.mv(tmptar_path, @output_tarball)
    end

    true
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

  def find_executable(command_name)
    # try to find lspci
    dirs = ["/sbin", "/usr/sbin", "/usr/local/sbin", "/usr/bin", "/usr/local/bin"]
    dirs += ENV['PATH'].split(":")

    bindir = dirs.find do |dir|
      File.executable?(File.expand_path(command_name, dir))
    end

    if bindir
      File.expand_path(command_name, bindir)
    else
      @errors << RuntimeError.new("#{command_name}(1) not found")
      nil
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

  def save_fdisk_info()
    fdisk_bin = find_executable("fdisk")

    File.open("#{@output_dir}/fdisk.log", "w") do |f|
      f.puts(`#{fdisk_bin} -l`)
    end
  end

  def save_numactl_info()
    numactl_bin = find_executable("numactl")

    File.open("#{@output_dir}/numactl.log", "w") do |f|
      f.puts(`#{numactl_bin} --hardware`)
    end
  end

  def save_ec2metadata_info()
    ec2metadata_bin = find_executable("ec2-metadata")

    File.open("#{@output_dir}/ec2-metadata.log", "w") do |f|
      f.puts(`#{ec2metadata_bin} --all`)
    end
  end

  def save_lsblk_info()
    lsblk_bin = find_executable("lsblk")

    File.open("#{@output_dir}/lsblk.log", "w") do |f|
      f.puts(`#{lsblk_bin} -t`)
    end
  end

  def save_lvm_info()
    vgdisplay_bin = find_executable("vgdisplay")
    lvdisplay_bin = find_executable("lvdisplay")
    pvdisplay_bin = find_executable("pvdisplay")

    File.open("#{@output_dir}/lvm-vgdisplay.log", "w") do |f|
      f.puts(`#{vgdisplay_bin}`)
    end

    File.open("#{@output_dir}/lvm-lvdisplay.log", "w") do |f|
      f.puts(`#{lvdisplay_bin}`)
    end

    File.open("#{@output_dir}/lvm-pvdisplay.log", "w") do |f|
      f.puts(`#{pvdisplay_bin}`)
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
    # list of file pattern regexps which cannot be read
    blacklists = [
                  %r|device/sw_activity$|,
                  %r|trace/pid$|,
                  %r|trace/end_lba$|,
                  %r|trace/enable$|,
                  %r|trace/act_mask$|,
                  %r|trace/start_lba$|,
                  %r|trace/pid$|,
                  %r|trace/end_lba$|,
                  %r|trace/act_mask$|,
                  %r|power/autosuspend_delay_ms$|,
                  %r|device/unload_heads$|,
                  %r|device/vpd_pg80$|,
                  %r|device/vpd_pg83$|,
                 ]

    Dir.glob('/sys/block/*').each do |blockdev|
      if File.basename(blockdev) =~ /^(loop|ram)/
        next
      end

      devname = File.basename(blockdev)

      File.open("#{@output_dir}/block-#{File.basename(blockdev)}.log", "w") do |f|
        f.puts("## ls -l #{blockdev}")
        f.puts(`ls -l #{blockdev}`)
        f.puts("")

        dirs = []
        Dir.glob(blockdev + "/*").each do |entry|
          next if blacklists.any?{|r| r =~ entry}

          st = File::Stat.new(entry)

          if st.ftype == "file" && st.readable? && st.mode & 0444 > 0
            f.puts("## #{entry}")
            f.puts(`cat #{entry}`)
            f.puts("")
          elsif st.ftype == "link" && st.readable? && st.mode & 0444 > 0
            f.puts("## #{entry}")
            f.puts(`ls -l #{entry}`)
            f.puts("")
          elsif st.ftype == "directory" && st.readable? && st.mode & 0444 > 0
            dirs.push(entry)
          end
        end

        while (dir = dirs.shift) != nil
          Dir.glob(dir + "/*").each do |entry|
            next if blacklists.any?{|r| r =~ entry}

            st = File::Stat.new(entry)

            if st.ftype == "file" && st.readable? && st.mode & 0444 > 0
              f.puts("## #{entry}")
              f.puts(`cat #{entry}`)
              f.puts("")
            elsif st.ftype == "link" && st.readable? && st.mode & 0444 > 0
              f.puts("## #{entry}")
              f.puts(`ls -l #{entry}`)
              f.puts("")
            elsif st.ftype == "directory" && st.readable? && st.mode & 0444 > 0
              f.puts("## #{entry}")
              f.puts(`ls -l #{entry}/`)
              f.puts("")
            end

            if devname =~ /^nvme/
              if entry =~ /device\/device$/
                dirs.push(entry)
              end
            end
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
    lspci_bin = find_executable("lspci")

    if lspci_bin
      File.open("#{@output_dir}/lspci.log", "w") do |f|
        f.puts(`#{lspci_bin} -D -vvv`)
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
    modules = []

    if lsmod_bin = find_executable("lsmod")
      File.open("#{@output_dir}/lsmod.log", "w") do |f|
        content = `#{lsmod_bin}`
        f.puts(content)

        lines = content.split("\n")
        lines.shift # omit 1st line (label)
        modules = lines.map do |line|
          line.split[0]
        end
      end
    else
      return
    end

    modinfo_bin = find_executable("modinfo")

    Dir.glob("/sys/module/*/parameters") do |params_dir|
      module_name = File.basename(File.dirname(params_dir))
      next unless modules.include?(module_name)

      File.open("#{@output_dir}/module-#{module_name}.log", "w") do |f|
        Dir.glob("#{params_dir}/*").each do |param_file|
          param_name = File.basename(param_file)
          # blacklisting
          next if module_name == "apparmor" && param_name == "audit"
          next if module_name == "apparmor" && param_name == "mode"

          content = read_file(param_file)
          f.puts("## #{param_file}")
          f.puts(content)
          f.puts("")
        end

        if modinfo_bin
          content = `#{modinfo_bin} #{module_name}`
          f.puts("## modinfo #{module_name}")
          f.puts(content)
          f.puts("")
        end
      end
    end
  end

  def save_distro_info()
    File.open("#{@output_dir}/distro.log", "w") do |f|
      if system("which uname >/dev/null 2>&1")
        content = `uname -a`
        f.puts("## uname -a")
        f.puts(content)
        f.puts("")
      end

      if system("which lsb_release >/dev/null 2>&1")
        content = `lsb_release -a 2>/dev/null`
        f.puts("## lsb_release -a")
        f.puts(content)
        f.puts("")
      end

      if File.exists?("/etc/debian_version")
        content = read_file("/etc/debian_version")
        f.puts("## /etc/debian_version")
        f.puts(content)
        f.puts("")
      end

      if File.exists?("/etc/redhat-release")
        content = read_file("/etc/redhat-release")
        f.puts("## /etc/redhat-release")
        f.puts(content)
        f.puts("")
      end
    end
  end

  def save_sysctl_info()
    sysctl_bin = find_executable("sysctl")

    if sysctl_bin
      File.open("#{@output_dir}/sysctl.log", "w") do |f|
        content = `#{sysctl_bin} -a`
        f.puts("## sysctl -a")
        f.puts(content)
        f.puts("")
      end
    end
  end

  def save_dmidecode_info()
    dmidecode_bin = find_executable("dmidecode")

    if dmidecode_bin
      File.open("#{@output_dir}/dmidecode.log", "w") do |f|
        content = `#{dmidecode_bin} 2>&1`
        f.puts("## dmidecode")
        f.puts(content)
        f.puts("")
      end
    end
  end

  def save_biosdecode_info()
    biosdecode_bin = find_executable("biosdecode")

    if biosdecode_bin
      File.open("#{@output_dir}/biosdecode.log", "w") do |f|
        content = `#{biosdecode_bin} 2>&1`
        f.puts("## biosdecode")
        f.puts(content)
        f.puts("")
      end
    end
  end

  def save_nvme_info()
    # https://github.com/linux-nvme/nvme-cli
    nvme_bin = find_executable("nvme")

    if nvme_bin
      File.open("#{@output_dir}/nvme-cli-list.log", "w") do |f|
        content = `#{nvme_bin} list 2>&1`
        f.puts("## nvme list")
        f.puts(content)
        f.puts("")
      end
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

