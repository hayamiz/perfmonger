package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const bashCompletion = `# bash completion support for PerfMonger

_perfmonger() {
    local cmd cur prev subcmd
    cmd=$1
    cur=$2
    prev=$3

    subcmds="live record play stat plot summary fingerprint init-shell"

    # contextual completion
    case $prev in
        perfmonger)
            COMPREPLY=( $(compgen -W "$subcmds" $cur) )
            return 0
            ;;
        -d|--disks)
            COMPREPLY=( $(tail -n +3 /proc/partitions 2>/dev/null | awk '{print $4}') )
            return 0
            ;;
        -o|--output)
            COMPREPLY=( $(compgen -o default) )
            return 0
            ;;
        -i|--interval|-s|--start-delay|-t|--timeout)
            COMPREPLY=()
            return 0
            ;;
    esac

    # complete options
    subcmd=${COMP_WORDS[1]}

    case "$cur" in
        # complete options
        -*)
            COMPREPLY=( $(compgen -W "$(perfmonger $subcmd --help 2>&1 | egrep -o ' ([-][[:alnum:]]|-{2}[[:alnum:]-]+)\b')" -- "$cur") )
            return 0
            ;;
        *)
            COMPREPLY=( $(compgen -o default "$cur") )
            return 0
            ;;
    esac

    return 0
}

complete -F _perfmonger perfmonger
`

const zshCompletion = `# zsh completion support for PerfMonger

_perfmonger() {
    local subcmds
    subcmds=(
        'live:Monitor live system performance'
        'record:Record system performance'
        'play:Play a recorded perfmonger session'
        'stat:Run a command and show performance summary'
        'plot:Plot system performance graphs'
        'summary:Summarize system performance data'
        'fingerprint:Gather device information'
        'init-shell:Initialize shell integration'
    )

    if (( CURRENT == 2 )); then
        _describe 'perfmonger subcommands' subcmds
    else
        case "$words[2]" in
            record)
                _arguments \
                    '-d[Disk devices to monitor]:disk device:_files' \
                    '--disks[Disk devices to monitor]:disk device:_files' \
                    '-i[Measurement interval]:interval:' \
                    '--interval[Measurement interval]:interval:' \
                    '-o[Output file]:output file:_files' \
                    '--output[Output file]:output file:_files' \
                    '-s[Start delay]:delay:' \
                    '--start-delay[Start delay]:delay:' \
                    '-t[Timeout]:timeout:' \
                    '--timeout[Timeout]:timeout:' \
                    '--debug[Enable debug mode]' \
                    '-z[Gzip output]' \
                    '--gzip[Gzip output]' \
                    '--no-cpu[Do not record CPU]' \
                    '--no-disk[Do not record disk]' \
                    '--no-net[Do not record network]' \
                    '--no-mem[Do not record memory]'
                ;;
            *)
                _files
                ;;
        esac
    fi
}

compdef _perfmonger perfmonger
`

func getParentShell() string {
	// Try to get parent process shell
	ppid := os.Getppid()
	
	// Try ps command to get parent shell
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", ppid), "-o", "comm=")
	output, err := cmd.Output()
	if err == nil {
		shell := strings.TrimSpace(string(output))
		return filepath.Base(shell)
	}
	
	// Fallback: check SHELL environment variable
	if shell := os.Getenv("SHELL"); shell != "" {
		return filepath.Base(shell)
	}
	
	return ""
}

func runInitShell(args []string) {
	shell := getParentShell()
	
	// Check if user wants the actual completion script
	generateScript := len(args) > 0 && args[0] == "-"
	
	switch shell {
	case "bash":
		if generateScript {
			fmt.Print(bashCompletion)
		} else {
			fmt.Println("# Add the following line to ~/.bashrc")
			fmt.Println()
			fmt.Println("eval \"$(perfmonger init-shell -)\"")
		}
		
	case "zsh":
		if generateScript {
			fmt.Print(zshCompletion)
		} else {
			fmt.Println("# Add the following line to ~/.zshrc")
			fmt.Println()
			fmt.Println("eval \"$(perfmonger init-shell -)\"")
		}
		
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", shell)
		fmt.Fprintln(os.Stderr, "Only bash and zsh are supported")
		os.Exit(1)
	}
}

// initShellOptions represents all options for the init-shell command
type initShellOptions struct {
	GenerateScript bool
}

// newInitShellOptions creates initShellOptions with defaults
func newInitShellOptions() *initShellOptions {
	return &initShellOptions{
		GenerateScript: false,
	}
}

// parseArgs validates and processes the parsed arguments
func (opts *initShellOptions) parseArgs(args []string, cmd *cobra.Command) error {
	// Check if user wants the actual completion script (Ruby compatibility)
	if len(args) > 0 && args[0] == "-" {
		opts.GenerateScript = true
	}
	
	return nil
}

// run executes the init-shell command logic
func (opts *initShellOptions) run() error {
	shell := getParentShell()
	
	switch shell {
	case "bash":
		if opts.GenerateScript {
			fmt.Print(bashCompletion)
		} else {
			fmt.Println("# Add the following line to ~/.bashrc")
			fmt.Println()
			fmt.Println("eval \"$(perfmonger init-shell -)\"")
		}
		
	case "zsh":
		if opts.GenerateScript {
			fmt.Print(zshCompletion)
		} else {
			fmt.Println("# Add the following line to ~/.zshrc")
			fmt.Println()
			fmt.Println("eval \"$(perfmonger init-shell -)\"")
		}
		
	default:
		return fmt.Errorf("unsupported shell: %s. Only bash and zsh are supported", shell)
	}
	
	return nil
}

// newInitShellCommand creates the init-shell subcommand
func newInitShellCommand() *cobra.Command {
	opts := newInitShellOptions()
	
	cmd := &cobra.Command{
		Use:   "init-shell",
		Short: "Initialize shell integration",
		Long:  `Generate shell script to init shell completion for bash and zsh.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.parseArgs(args, cmd); err != nil {
				return err
			}
			return opts.run()
		},
	}
	
	cmd.SetUsageTemplate(subCommandUsageTemplate)
	
	return cmd
}