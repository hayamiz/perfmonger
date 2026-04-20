package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type fingerprintCommand struct {
	outputTarball string
	outputDir     string
	errors        []error
}

func (f *fingerprintCommand) parseArgs(args []string) error {
	fs := flag.NewFlagSet("fingerprint", flag.ContinueOnError)
	
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: perfmonger fingerprint [options] OUTPUT_TARBALL\n\n")
		fmt.Fprintf(os.Stderr, "Gather all possible system config information\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}
	
	if err := fs.Parse(args); err != nil {
		return err
	}
	
	remaining := fs.Args()
	if len(remaining) == 0 {
		// Default output name
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown"
		}
		f.outputTarball = fmt.Sprintf("./fingerprint.%s.tar.gz", hostname)
	} else {
		f.outputTarball = remaining[0]
		// Ensure .tar.gz extension
		if !strings.HasSuffix(f.outputTarball, ".tar.gz") && !strings.HasSuffix(f.outputTarball, ".tgz") {
			f.outputTarball += ".tar.gz"
		}
	}
	
	return nil
}

func (f *fingerprintCommand) run() error {
	// Set LANG=C for consistent output
	os.Setenv("LANG", "C")
	
	fmt.Fprintf(os.Stderr, "System information is gathered into %s\n", f.outputTarball)
	
	// Create temporary directory
	tmpdir, err := ioutil.TempDir("", "perfmonger-fingerprint-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpdir)
	
	// Create output directory within tmpdir
	baseName := strings.TrimSuffix(filepath.Base(f.outputTarball), ".tar.gz")
	baseName = strings.TrimSuffix(baseName, ".tgz")
	f.outputDir = filepath.Join(tmpdir, baseName)
	
	if err := os.MkdirAll(f.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %v", err)
	}
	
	// Initialize errors slice
	f.errors = []error{}
	
	// Collect system information
	f.doWithMessage("Saving /proc info", f.saveProcInfo)
	f.doWithMessage("Saving CPU info", f.saveCPUInfo)
	f.doWithMessage("Saving memory info", f.saveMemoryInfo)
	f.doWithMessage("Saving disk info", f.saveDiskInfo)
	f.doWithMessage("Saving block device info", f.saveBlockDeviceInfo)
	f.doWithMessage("Saving PCI/PCIe info", f.savePCIInfo)
	f.doWithMessage("Saving kernel module info", f.saveModuleInfo)
	f.doWithMessage("Saving distro info", f.saveDistroInfo)
	f.doWithMessage("Saving sysctl info", f.saveSysctlInfo)
	f.doWithMessage("Saving network info", f.saveNetworkInfo)
	
	// Create tarball
	if err := f.createTarball(tmpdir, baseName); err != nil {
		return fmt.Errorf("failed to create tarball: %v", err)
	}
	
	return nil
}

func (f *fingerprintCommand) doWithMessage(message string, fn func()) {
	fmt.Fprintf(os.Stderr, "%s ... ", message)
	
	errCountBefore := len(f.errors)
	fn()
	errCountAfter := len(f.errors)
	
	if errCountBefore == errCountAfter {
		fmt.Fprintln(os.Stderr, "done")
	} else {
		fmt.Fprintln(os.Stderr, "failed")
		for i := errCountBefore; i < errCountAfter; i++ {
			fmt.Fprintf(os.Stderr, " ERROR: %v\n", f.errors[i])
		}
	}
	fmt.Fprintln(os.Stderr)
}

func (f *fingerprintCommand) readFile(src string) ([]byte, error) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		f.errors = append(f.errors, err)
		return nil, err
	}
	return content, nil
}

func (f *fingerprintCommand) saveFile(filename string, content []byte) {
	path := filepath.Join(f.outputDir, filename)
	if err := ioutil.WriteFile(path, content, 0644); err != nil {
		f.errors = append(f.errors, err)
	}
}

func (f *fingerprintCommand) runCommand(cmd string, args ...string) ([]byte, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		// Don't treat command errors as fatal, just log them
		return output, nil
	}
	return output, nil
}

func (f *fingerprintCommand) saveProcInfo() {
	procFiles := []string{
		"cpuinfo", "meminfo", "mdstat", "mounts", "interrupts",
		"diskstats", "partitions", "ioports", "version", "cmdline",
		"filesystems", "swaps",
	}
	
	for _, file := range procFiles {
		if content, err := f.readFile("/proc/" + file); err == nil {
			f.saveFile("proc-"+file+".log", content)
		}
	}
	
	// Save /proc/sys/fs info
	var fsContent strings.Builder
	entries, _ := ioutil.ReadDir("/proc/sys/fs")
	for _, entry := range entries {
		if entry.Mode().IsRegular() {
			path := filepath.Join("/proc/sys/fs", entry.Name())
			fsContent.WriteString(fmt.Sprintf("## %s\n", path))
			if content, err := f.readFile(path); err == nil {
				fsContent.Write(content)
			} else {
				fsContent.WriteString("permission denied\n")
			}
			fsContent.WriteString("\n")
		}
	}
	f.saveFile("proc-sys-fs.log", []byte(fsContent.String()))
}

func (f *fingerprintCommand) saveCPUInfo() {
	// lscpu output
	if output, _ := f.runCommand("lscpu"); len(output) > 0 {
		f.saveFile("lscpu.log", output)
	}
	
	// CPU frequency info
	var freqContent strings.Builder
	cpuFreqDir := "/sys/devices/system/cpu"
	entries, _ := ioutil.ReadDir(cpuFreqDir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "cpu") && entry.IsDir() {
			cpuPath := filepath.Join(cpuFreqDir, entry.Name(), "cpufreq")
			if _, err := os.Stat(cpuPath); err == nil {
				freqContent.WriteString(fmt.Sprintf("## %s\n", entry.Name()))
				files := []string{"scaling_cur_freq", "scaling_min_freq", "scaling_max_freq", "scaling_governor"}
				for _, file := range files {
					path := filepath.Join(cpuPath, file)
					if content, err := f.readFile(path); err == nil {
						freqContent.WriteString(fmt.Sprintf("%s: %s", file, content))
					}
				}
				freqContent.WriteString("\n")
			}
		}
	}
	if freqContent.Len() > 0 {
		f.saveFile("cpu-frequency.log", []byte(freqContent.String()))
	}
}

func (f *fingerprintCommand) saveMemoryInfo() {
	// free output
	if output, _ := f.runCommand("free", "-h"); len(output) > 0 {
		f.saveFile("free.log", output)
	}
	
	// numactl output
	if output, _ := f.runCommand("numactl", "--hardware"); len(output) > 0 {
		f.saveFile("numactl.log", output)
	}
}

func (f *fingerprintCommand) saveDiskInfo() {
	// fdisk output
	if output, _ := f.runCommand("fdisk", "-l"); len(output) > 0 {
		f.saveFile("fdisk.log", output)
	}
	
	// lsblk output
	if output, _ := f.runCommand("lsblk", "-t"); len(output) > 0 {
		f.saveFile("lsblk.log", output)
	}
	
	// df output
	if output, _ := f.runCommand("df", "-h"); len(output) > 0 {
		f.saveFile("df.log", output)
	}
	
	// mount output
	if output, _ := f.runCommand("mount"); len(output) > 0 {
		f.saveFile("mount.log", output)
	}
}

func (f *fingerprintCommand) saveBlockDeviceInfo() {
	var blockContent strings.Builder
	
	blockDevs, _ := ioutil.ReadDir("/sys/block")
	for _, dev := range blockDevs {
		devName := dev.Name()
		// Skip loop and ram devices
		if strings.HasPrefix(devName, "loop") || strings.HasPrefix(devName, "ram") {
			continue
		}
		
		blockContent.WriteString(fmt.Sprintf("## /sys/block/%s\n", devName))
		
		// Read common attributes
		attrs := []string{"size", "queue/scheduler", "queue/rotational", "queue/hw_sector_size"}
		for _, attr := range attrs {
			path := filepath.Join("/sys/block", devName, attr)
			if content, err := f.readFile(path); err == nil {
				blockContent.WriteString(fmt.Sprintf("%s: %s", attr, content))
			}
		}
		blockContent.WriteString("\n")
	}
	
	if blockContent.Len() > 0 {
		f.saveFile("block-devices.log", []byte(blockContent.String()))
	}
}

func (f *fingerprintCommand) savePCIInfo() {
	// lspci output
	if output, _ := f.runCommand("lspci", "-vvv"); len(output) > 0 {
		f.saveFile("lspci.log", output)
	}
}

func (f *fingerprintCommand) saveModuleInfo() {
	// lsmod output
	if output, _ := f.runCommand("lsmod"); len(output) > 0 {
		f.saveFile("lsmod.log", output)
	}
}

func (f *fingerprintCommand) saveDistroInfo() {
	var distroContent strings.Builder
	
	// uname output
	if output, _ := f.runCommand("uname", "-a"); len(output) > 0 {
		distroContent.WriteString("## uname -a\n")
		distroContent.Write(output)
		distroContent.WriteString("\n")
	}
	
	// lsb_release output
	if output, _ := f.runCommand("lsb_release", "-a"); len(output) > 0 {
		distroContent.WriteString("## lsb_release -a\n")
		distroContent.Write(output)
		distroContent.WriteString("\n")
	}
	
	// Distribution files
	distFiles := []string{"/etc/debian_version", "/etc/redhat-release", "/etc/os-release"}
	for _, file := range distFiles {
		if content, err := f.readFile(file); err == nil {
			distroContent.WriteString(fmt.Sprintf("## %s\n", file))
			distroContent.Write(content)
			distroContent.WriteString("\n")
		}
	}
	
	if distroContent.Len() > 0 {
		f.saveFile("distro.log", []byte(distroContent.String()))
	}
}

func (f *fingerprintCommand) saveSysctlInfo() {
	// sysctl output
	if output, _ := f.runCommand("sysctl", "-a"); len(output) > 0 {
		f.saveFile("sysctl.log", output)
	}
}

func (f *fingerprintCommand) saveNetworkInfo() {
	var netContent strings.Builder
	
	// ip addr output
	if output, _ := f.runCommand("ip", "addr"); len(output) > 0 {
		netContent.WriteString("## ip addr\n")
		netContent.Write(output)
		netContent.WriteString("\n")
	}
	
	// ip route output
	if output, _ := f.runCommand("ip", "route"); len(output) > 0 {
		netContent.WriteString("## ip route\n")
		netContent.Write(output)
		netContent.WriteString("\n")
	}
	
	// netstat output
	if output, _ := f.runCommand("netstat", "-i"); len(output) > 0 {
		netContent.WriteString("## netstat -i\n")
		netContent.Write(output)
		netContent.WriteString("\n")
	}
	
	if netContent.Len() > 0 {
		f.saveFile("network.log", []byte(netContent.String()))
	}
}

func (f *fingerprintCommand) createTarball(tmpdir, baseName string) error {
	// Create the tar.gz file
	tarFile, err := os.Create(f.outputTarball)
	if err != nil {
		return err
	}
	defer tarFile.Close()
	
	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()
	
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	
	// Walk the output directory and add files to the tarball
	baseDir := filepath.Join(tmpdir, baseName)
	return filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Create tar header
		relPath, err := filepath.Rel(tmpdir, path)
		if err != nil {
			return err
		}
		
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath
		
		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		// If it's a file, write its contents
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			
			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}
		
		return nil
	})
}

func runFingerprint(args []string) {
	cmd := &fingerprintCommand{}
	if err := cmd.parseArgs(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	if err := cmd.run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// fingerprintOptions represents all options for the fingerprint command
type fingerprintOptions struct {
	OutputTarball string
}

// newFingerprintOptions creates fingerprintOptions with Ruby-compatible defaults
func newFingerprintOptions() *fingerprintOptions {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	
	return &fingerprintOptions{
		OutputTarball: fmt.Sprintf("./fingerprint.%s.tar.gz", hostname),
	}
}

// parseArgs validates and processes the parsed arguments
func (opts *fingerprintOptions) parseArgs(args []string, cmd *cobra.Command) error {
	// Handle output tarball argument
	if len(args) > 0 {
		opts.OutputTarball = args[0]
	}
	
	// Ensure .tar.gz extension (Ruby compatibility)
	if !strings.HasSuffix(opts.OutputTarball, ".tar.gz") && !strings.HasSuffix(opts.OutputTarball, ".tgz") {
		opts.OutputTarball += ".tar.gz"
	}
	
	return nil
}

// run executes the fingerprint command logic
func (opts *fingerprintOptions) run() error {
	// Use the existing implementation
	cmd := &fingerprintCommand{outputTarball: opts.OutputTarball}
	return cmd.run()
}

// newFingerprintCommand creates the fingerprint subcommand with Ruby-compatible options
func newFingerprintCommand() *cobra.Command {
	opts := newFingerprintOptions()
	
	cmd := &cobra.Command{
		Use:   "fingerprint [options] OUTPUT_TARBALL",
		Short: "Gather all possible system config information",
		Long:  `Gather all possible system config information and create a tarball.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validation moved to PreRunE for consistency
			return opts.parseArgs(args, cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Direct execution - no additional parsing needed
			return opts.run()
		},
	}
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	
	// Add Ruby-compatible aliases
	cmd.Aliases = []string{"bukko", "fp"}
	
	return cmd
}