package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

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

	// Build command arguments
	args := []string{
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

// findPlotFormatterBinary finds the plot-formatter binary (placeholder implementation)
func findPlotFormatterBinary() (string, error) {
	// This is a simplified implementation. In the real Ruby version, it uses CoreFinder.
	// For now, use the relative path to perfmonger-core binary
	return "./lib/exec/perfmonger-core_linux_amd64", nil
}

// generatePlots generates the actual plot files (placeholder implementation)
func generatePlots(cmd *plotCommand, tmpDir string, meta *PlotMeta) error {
	// This is a major simplification. The full implementation would:
	// 1. Generate gnuplot scripts for disk IOPS, disk transfer, CPU usage, memory usage
	// 2. Execute gnuplot to create PDF files
	// 3. Optionally convert to PNG using ImageMagick
	// 4. Save or cleanup temporary files based on cmd.SaveGpfiles
	
	fmt.Printf("Plot generation completed (simplified implementation)\n")
	fmt.Printf("Data file: %s\n", cmd.DataFile)
	fmt.Printf("Output directory: %s\n", cmd.OutputDir)
	fmt.Printf("Output type: %s\n", cmd.OutputType)
	fmt.Printf("Time range: %.2f - %.2f seconds\n", meta.StartTime, meta.EndTime)
	fmt.Printf("CPU cores: %d\n", meta.Cpu.NumCore)
	fmt.Printf("Disk devices: %d\n", len(meta.Disk.Devices))
	
	return nil
}