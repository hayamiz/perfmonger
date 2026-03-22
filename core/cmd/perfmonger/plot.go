package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// plotCommand represents the plot command with direct field setting
type plotCommand struct {
	// Input data file
	DataFile string

	// Output options
	OffsetTime    float64
	OutputDir     string
	OutputType    string
	OutputPrefix  string
	SaveGpfiles   bool

	// Disk filtering options
	DiskOnly      string
	DiskOnlyRegex *regexp.Regexp

	// Plot mode options
	PlotReadOnly   bool
	PlotWriteOnly  bool
	PlotReadWrite  bool
	DiskPlotRead   bool
	DiskPlotWrite  bool

	// Advanced options
	DiskNumkeyThreshold int
	PlotIOPSMax        float64
	GnuplotBin         string
}

// newPlotCommandStruct creates plotCommand with Ruby-compatible defaults
func newPlotCommandStruct() *plotCommand {
	return &plotCommand{
		DataFile:            "",
		OffsetTime:          0.0,
		OutputDir:           ".",
		OutputType:          "pdf",
		OutputPrefix:        "",
		SaveGpfiles:         false,
		DiskOnly:           "",
		DiskOnlyRegex:      nil,
		PlotReadOnly:       false,
		PlotWriteOnly:      false,
		PlotReadWrite:      false,
		DiskPlotRead:       true,  // Default: plot read
		DiskPlotWrite:      true,  // Default: plot write
		DiskNumkeyThreshold: 10,
		PlotIOPSMax:        0.0,   // 0 = auto
		GnuplotBin:         "gnuplot",
	}
}

// validateAndSetDataFile validates the data file argument using cobra's PreRunE approach
func (cmd *plotCommand) validateAndSetDataFile(args []string) error {
	// Validate data file argument
	if len(args) == 0 {
		return fmt.Errorf("LOG_FILE argument is required")
	}

	cmd.DataFile = args[0]

	// Check if data file exists
	if _, err := os.Stat(cmd.DataFile); os.IsNotExist(err) {
		return fmt.Errorf("data file %q does not exist", cmd.DataFile)
	}

	return nil
}

// validateOptions performs validation using cobra's PreRunE approach
func (cmd *plotCommand) validateOptions() error {
	// Validate output type
	if cmd.OutputType != "pdf" && cmd.OutputType != "png" {
		return fmt.Errorf("output-type must be 'pdf' or 'png', got %q", cmd.OutputType)
	}

	// Compile disk-only regex if provided
	if cmd.DiskOnly != "" {
		regex, err := regexp.Compile(cmd.DiskOnly)
		if err != nil {
			return fmt.Errorf("invalid disk-only regex: %v", err)
		}
		cmd.DiskOnlyRegex = regex
	}

	// Handle plot mode flags
	if cmd.PlotReadOnly {
		cmd.DiskPlotRead = true
		cmd.DiskPlotWrite = false
	} else if cmd.PlotWriteOnly {
		cmd.DiskPlotRead = false
		cmd.DiskPlotWrite = true
	} else if cmd.PlotReadWrite {
		cmd.DiskPlotRead = true
		cmd.DiskPlotWrite = true
	}

	// Validate plot options
	if !cmd.DiskPlotRead && !cmd.DiskPlotWrite {
		return fmt.Errorf("at least one of read or write plotting must be enabled")
	}

	// Validate IOPS max
	if cmd.PlotIOPSMax < 0 {
		return fmt.Errorf("plot-iops-max cannot be negative")
	}

	// Validate disk numkey threshold
	if cmd.DiskNumkeyThreshold < 0 {
		return fmt.Errorf("plot-numkey-threshold cannot be negative")
	}

	return nil
}

// run executes the plot command with direct field access
func (cmd *plotCommand) run() error {
	if os.Getenv("PERFMONGER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] running plot command with file: %s\n", cmd.DataFile)
	}

	return runPlotCommand(cmd)
}

// newPlotCommand creates the plot subcommand with direct cobra setting
func newPlotCommand() *cobra.Command {
	plotCmd := newPlotCommandStruct()

	cmd := &cobra.Command{
		Use:   "plot [options] LOG_FILE",
		Short: "Plot system performance graphs",
		Long:  `Plot system performance graphs from recorded performance data.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Validation moved to PreRunE for cobra integration
			if err := plotCmd.validateAndSetDataFile(args); err != nil {
				return err
			}
			return plotCmd.validateOptions()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Direct execution - no additional validation needed
			return plotCmd.run()
		},
	}

	// Direct cobra flag setting to plotCommand fields (no conversion needed)
	cmd.Flags().Float64Var(&plotCmd.OffsetTime, "offset-time", plotCmd.OffsetTime,
		"Offset time in seconds")
	cmd.Flags().StringVarP(&plotCmd.OutputDir, "output-dir", "o", plotCmd.OutputDir,
		"Output directory")
	cmd.Flags().StringVarP(&plotCmd.OutputType, "output-type", "T", plotCmd.OutputType,
		"Output type: pdf, png")
	cmd.Flags().StringVarP(&plotCmd.OutputPrefix, "prefix", "p", plotCmd.OutputPrefix,
		"Output file name prefix")
	cmd.Flags().BoolVarP(&plotCmd.SaveGpfiles, "save", "s", plotCmd.SaveGpfiles,
		"Save GNUPLOT and data files")
	cmd.Flags().StringVar(&plotCmd.DiskOnly, "disk-only", plotCmd.DiskOnly,
		"Select disk devices that match REGEX")

	// Plot mode flags
	cmd.Flags().BoolVar(&plotCmd.PlotReadOnly, "plot-read-only", plotCmd.PlotReadOnly,
		"Plot only READ performance for disks")
	cmd.Flags().BoolVar(&plotCmd.PlotWriteOnly, "plot-write-only", plotCmd.PlotWriteOnly,
		"Plot only WRITE performance for disks")
	cmd.Flags().BoolVar(&plotCmd.PlotReadWrite, "plot-read-write", plotCmd.PlotReadWrite,
		"Plot READ and WRITE performance for disks")

	// Advanced options
	cmd.Flags().IntVar(&plotCmd.DiskNumkeyThreshold, "plot-numkey-threshold", plotCmd.DiskNumkeyThreshold,
		"Turn off legends if disk count exceeds this")
	cmd.Flags().Float64Var(&plotCmd.PlotIOPSMax, "plot-iops-max", plotCmd.PlotIOPSMax,
		"Maximum of IOPS plot range (0=auto)")
	cmd.Flags().StringVar(&plotCmd.GnuplotBin, "with-gnuplot", plotCmd.GnuplotBin,
		"Path to gnuplot binary")

	cmd.SetUsageTemplate(subCommandUsageTemplate)
	return cmd
}

// runPlotCommand implements the actual plot functionality
func runPlotCommand(cmd *plotCommand) error {
	// Check if gnuplot is available
	if err := checkGnuplotAvailable(cmd.GnuplotBin); err != nil {
		return err
	}

	// Check if pdfcairo terminal is supported
	if cmd.OutputType == "pdf" {
		if err := checkPdfCairoSupported(cmd.GnuplotBin); err != nil {
			return err
		}
	}

	// Check if ImageMagick convert is available for non-PDF output
	if cmd.OutputType != "pdf" {
		if err := checkConvertAvailable(); err != nil {
			return err
		}
	}

	// Create temporary directory for data files
	tmpDir, err := os.MkdirTemp("", "perfmonger-plot-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Run plot-formatter to generate data files
	diskDat := filepath.Join(tmpDir, "disk.dat")
	cpuDat := filepath.Join(tmpDir, "cpu.dat") 
	memDat := filepath.Join(tmpDir, "mem.dat")

	meta, err := runPlotFormatter(cmd.DataFile, diskDat, cpuDat, memDat, cmd.DiskOnly)
	if err != nil {
		return err
	}

	// Generate plots
	if err := generatePlots(cmd, tmpDir, meta); err != nil {
		return err
	}

	return nil
}

// checkGnuplotAvailable checks if gnuplot binary is available
func checkGnuplotAvailable(gnuplotBin string) error {
	cmd := exec.Command("which", gnuplotBin)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gnuplot not found")
	}
	return nil
}

// checkPdfCairoSupported checks if pdfcairo terminal is supported
func checkPdfCairoSupported(gnuplotBin string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`%s -e "set terminal" < /dev/null 2>&1 | grep pdfcairo >/dev/null 2>&1`, gnuplotBin))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pdfcairo is not supported by installed gnuplot")
	}
	return nil
}

// checkConvertAvailable checks if ImageMagick convert is available
func checkConvertAvailable() error {
	cmd := exec.Command("which", "convert")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("convert(1) not found. ImageMagick is required for non-PDF output")
	}
	return nil
}

// PlotMeta represents metadata returned by plot-formatter
type PlotMeta struct {
	Disk struct {
		Devices []struct {
			Name string `json:"name"`
			Idx  int    `json:"idx"`
		} `json:"devices"`
	} `json:"disk"`
	Cpu struct {
		NumCore int `json:"num_core"`
	} `json:"cpu"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

// runPlotFormatter runs the plot-formatter component to generate data files
func runPlotFormatter(dataFile, diskDat, cpuDat, memDat, diskOnly string) (*PlotMeta, error) {
	// Find the plot-formatter binary using the same logic as the Ruby implementation
	formatterBin, err := findPlotFormatterBinary()
	if err != nil {
		return nil, err
	}

	// Build command arguments: "plot-format" subcommand + formatter flags
	args := []string{
		"plot-format",
		"-perfmonger", dataFile,
		"-cpufile", cpuDat,
		"-diskfile", diskDat,
		"-memfile", memDat,
	}

	if diskOnly != "" {
		args = append(args, "-disk-only", diskOnly)
	}

	// Run the command and capture JSON output
	cmd := exec.Command(formatterBin, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run perfmonger-plot-formatter: %v", err)
	}

	// Parse the JSON metadata
	var meta PlotMeta
	if err := json.Unmarshal(output, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse plot-formatter output: %v", err)
	}

	return &meta, nil
}

// findPlotFormatterBinary returns the path to use as the plot-formatter binary.
// In the unified binary architecture, this is our own executable with the
// hidden "plot-format" subcommand.
func findPlotFormatterBinary() (string, error) {
	selfBin, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to find own executable: %v", err)
	}
	return selfBin, nil
}

// generatePlots generates the actual plot files using gnuplot
func generatePlots(cmd *plotCommand, tmpDir string, meta *PlotMeta) error {
	duration := meta.EndTime - meta.StartTime
	diskDat := filepath.Join(tmpDir, "disk.dat")
	cpuDat := filepath.Join(tmpDir, "cpu.dat")

	// Disk IOPS plot
	if err := generateDiskIOPSPlot(cmd, tmpDir, diskDat, meta, duration); err != nil {
		return err
	}
	// Disk transfer plot
	if err := generateDiskTransferPlot(cmd, tmpDir, diskDat, meta, duration); err != nil {
		return err
	}
	// CPU plot
	if err := generateCPUPlot(cmd, tmpDir, cpuDat, meta, duration); err != nil {
		return err
	}
	// AllCPU plot
	if err := generateAllCPUPlot(cmd, tmpDir, cpuDat, meta, duration); err != nil {
		return err
	}

	// Copy data/gp files if --save
	if cmd.SaveGpfiles {
		for _, name := range []string{"disk.dat", "cpu.dat", "mem.dat", "disk-iops.gp", "disk-transfer.gp", "cpu.gp", "allcpu.gp"} {
			src := filepath.Join(tmpDir, name)
			if _, err := os.Stat(src); err == nil {
				dst := filepath.Join(cmd.OutputDir, name)
				data, _ := os.ReadFile(src)
				os.WriteFile(dst, data, 0644)
			}
		}
	}

	return nil
}

func buildDiskPlotLines(diskDat string, meta *PlotMeta, cmd *plotCommand, yColRead, yColWrite int) string {
	var lines []string
	for _, dev := range meta.Disk.Devices {
		if cmd.DiskOnlyRegex != nil && !cmd.DiskOnlyRegex.MatchString(dev.Name) {
			continue
		}
		if cmd.DiskPlotRead {
			lines = append(lines, fmt.Sprintf(
				`"%s" ind %d usi 1:%d with lines lw 2 title "%s read"`,
				diskDat, dev.Idx, yColRead, dev.Name))
		}
		if cmd.DiskPlotWrite {
			lines = append(lines, fmt.Sprintf(
				`"%s" ind %d usi 1:%d with lines lw 2 title "%s write"`,
				diskDat, dev.Idx, yColWrite, dev.Name))
		}
	}
	return strings.Join(lines, ", \\\n     ")
}

func setKeyStmt(meta *PlotMeta, threshold int) string {
	if len(meta.Disk.Devices) > threshold {
		return "unset key"
	}
	return "set key below center"
}

func runGnuplot(cmd *plotCommand, gpFile string) error {
	c := exec.Command(cmd.GnuplotBin, gpFile)
	c.Stderr = os.Stderr
	return c.Run()
}

func generateDiskIOPSPlot(cmd *plotCommand, tmpDir, diskDat string, meta *PlotMeta, duration float64) error {
	gpFile := filepath.Join(tmpDir, "disk-iops.gp")
	outFile := filepath.Join(cmd.OutputDir, "disk-iops."+cmd.OutputType)

	plotLines := buildDiskPlotLines(diskDat, meta, cmd, 2, 3)
	if plotLines == "" {
		return nil
	}

	iopsMax := "*"
	if cmd.PlotIOPSMax > 0 {
		iopsMax = fmt.Sprintf("%g", cmd.PlotIOPSMax)
	}

	script := fmt.Sprintf(`set term pdfcairo enhanced color size 6in,2.5in
set title "IOPS"
set output "%s"
set xlabel "elapsed time [sec]"
set ylabel "IOPS"
set grid
set xrange [%g:%g]
set yrange [0:%s]
%s

plot %s
`, outFile, cmd.OffsetTime, duration, iopsMax, setKeyStmt(meta, cmd.DiskNumkeyThreshold), plotLines)

	if err := os.WriteFile(gpFile, []byte(script), 0644); err != nil {
		return err
	}
	return runGnuplot(cmd, gpFile)
}

func generateDiskTransferPlot(cmd *plotCommand, tmpDir, diskDat string, meta *PlotMeta, duration float64) error {
	gpFile := filepath.Join(tmpDir, "disk-transfer.gp")
	outFile := filepath.Join(cmd.OutputDir, "disk-transfer."+cmd.OutputType)

	plotLines := buildDiskPlotLines(diskDat, meta, cmd, 4, 5)
	if plotLines == "" {
		return nil
	}

	script := fmt.Sprintf(`set term pdfcairo enhanced color size 6in,2.5in
set title "Transfer rate"
set output "%s"
set xlabel "elapsed time [sec]"
set ylabel "transfer rate [MB/s]"
set grid
set xrange [%g:%g]
set yrange [0:*]
%s

plot %s
`, outFile, cmd.OffsetTime, duration, setKeyStmt(meta, cmd.DiskNumkeyThreshold), plotLines)

	if err := os.WriteFile(gpFile, []byte(script), 0644); err != nil {
		return err
	}
	return runGnuplot(cmd, gpFile)
}

func generateCPUPlot(cmd *plotCommand, tmpDir, cpuDat string, meta *PlotMeta, duration float64) error {
	gpFile := filepath.Join(tmpDir, "cpu.gp")
	outFile := filepath.Join(cmd.OutputDir, "cpu."+cmd.OutputType)

	script := fmt.Sprintf(`set term pdfcairo enhanced color size 6in,2.5in
set title "CPU usage (max: %d%%)"
set output "%s"
set key outside center bottom horizontal
set xlabel "elapsed time [sec]"
set ylabel "CPU usage"
set grid
set xrange [%g:%g]
set yrange [0:*]

plot "%s" ind 0 usi 1:($2+$3+$4+$5+$6+$7+$8+$9) with filledcurve x1 lw 0 lc 1 title "%%usr", \
     "%s" ind 0 usi 1:($3+$4+$5+$6+$7+$8+$9) with filledcurve x1 lw 0 lc 2 title "%%nice", \
     "%s" ind 0 usi 1:($4+$5+$6+$7+$8+$9) with filledcurve x1 lw 0 lc 3 title "%%sys", \
     "%s" ind 0 usi 1:($5+$6+$7+$8+$9) with filledcurve x1 lw 0 lc 4 title "%%iowait", \
     "%s" ind 0 usi 1:($6+$7+$8+$9) with filledcurve x1 lw 0 lc 5 title "%%hardirq", \
     "%s" ind 0 usi 1:($7+$8+$9) with filledcurve x1 lw 0 lc 6 title "%%softirq", \
     "%s" ind 0 usi 1:($8+$9) with filledcurve x1 lw 0 lc 7 title "%%steal", \
     "%s" ind 0 usi 1:($9) with filledcurve x1 lw 0 lc 8 title "%%guest"
`, meta.Cpu.NumCore*100, outFile, cmd.OffsetTime, duration,
		cpuDat, cpuDat, cpuDat, cpuDat, cpuDat, cpuDat, cpuDat, cpuDat)

	if err := os.WriteFile(gpFile, []byte(script), 0644); err != nil {
		return err
	}
	return runGnuplot(cmd, gpFile)
}

func generateAllCPUPlot(cmd *plotCommand, tmpDir, cpuDat string, meta *PlotMeta, duration float64) error {
	gpFile := filepath.Join(tmpDir, "allcpu.gp")
	outFile := filepath.Join(cmd.OutputDir, "allcpu."+cmd.OutputType)
	nrCPU := meta.Cpu.NumCore

	plotHeight := 8.0
	if nrCPU > 8 {
		plotHeight += float64(nrCPU-8) * 0.5
	}
	legendHeight := 0.06
	cellHeight := (1.0 - legendHeight) / float64(nrCPU)

	var sb strings.Builder
	fmt.Fprintf(&sb, "set term pdfcairo color enhanced size 8.5inch, %.1finch\n", plotHeight)
	fmt.Fprintf(&sb, "set output \"%s\"\n", outFile)
	sb.WriteString("set size 1.0, 1.0\nset multiplot\nset grid\n")
	fmt.Fprintf(&sb, "set xrange [%g:%g]\nset yrange [0:101]\n", cmd.OffsetTime, duration)

	for i := 0; i < nrCPU; i++ {
		ypos := legendHeight + float64(nrCPU-1-i)*cellHeight
		fmt.Fprintf(&sb, "\nset title 'cpu %d' offset -61,-3 font 'Arial,16'\n", i)
		sb.WriteString("unset key\n")
		fmt.Fprintf(&sb, "set origin 0.0, %f\nset size 1.0, %f\n", ypos, cellHeight)
		sb.WriteString("set rmargin 2\nset lmargin 12\nset tmargin 0.5\nset bmargin 0.5\n")
		sb.WriteString("set xtics offset 0.0,0.5\nset ytics offset 0.5,0\nset style fill noborder\n")

		ind := i + 1 // ind 0 = aggregate, ind 1..N = per-core
		fmt.Fprintf(&sb, `plot "%s" ind %d usi 1:($2+$3+$4+$5+$6+$7+$8+$9) with filledcurve x1 lw 0 lc 1 title "%%usr", \
     "%s" ind %d usi 1:($3+$4+$5+$6+$7+$8+$9) with filledcurve x1 lw 0 lc 2 title "%%nice", \
     "%s" ind %d usi 1:($4+$5+$6+$7+$8+$9) with filledcurve x1 lw 0 lc 3 title "%%sys", \
     "%s" ind %d usi 1:($5+$6+$7+$8+$9) with filledcurve x1 lw 0 lc 4 title "%%iowait", \
     "%s" ind %d usi 1:($6+$7+$8+$9) with filledcurve x1 lw 0 lc 5 title "%%hardirq", \
     "%s" ind %d usi 1:($7+$8+$9) with filledcurve x1 lw 0 lc 6 title "%%softirq", \
     "%s" ind %d usi 1:($8+$9) with filledcurve x1 lw 0 lc 7 title "%%steal", \
     "%s" ind %d usi 1:($9) with filledcurve x1 lw 0 lc 8 title "%%guest"
`,
			cpuDat, ind, cpuDat, ind, cpuDat, ind, cpuDat, ind,
			cpuDat, ind, cpuDat, ind, cpuDat, ind, cpuDat, ind)
	}

	// Legend at bottom
	sb.WriteString("\nunset title\nset key center center horizontal font \"Arial,14\"\n")
	fmt.Fprintf(&sb, "set origin 0.0, 0.0\nset size 1.0, %f\n", legendHeight)
	sb.WriteString("set rmargin 0\nset lmargin 0\nset tmargin 0\nset bmargin 0\n")
	sb.WriteString("unset tics\nset border 0\nset yrange [0:1]\n")
	sb.WriteString("set xlabel \"elapsed time [sec]\"\n")
	sb.WriteString(`plot -1 with filledcurve x1 lw 0 lc 1 title "%usr", \
     -1 with filledcurve x1 lw 0 lc 2 title "%nice", \
     -1 with filledcurve x1 lw 0 lc 3 title "%sys", \
     -1 with filledcurve x1 lw 0 lc 4 title "%iowait", \
     -1 with filledcurve x1 lw 0 lc 5 title "%hardirq", \
     -1 with filledcurve x1 lw 0 lc 6 title "%softirq", \
     -1 with filledcurve x1 lw 0 lc 7 title "%steal", \
     -1 with filledcurve x1 lw 0 lc 8 title "%guest"
`)
	sb.WriteString("\nunset multiplot\n")

	if err := os.WriteFile(gpFile, []byte(sb.String()), 0644); err != nil {
		return err
	}
	return runGnuplot(cmd, gpFile)
}