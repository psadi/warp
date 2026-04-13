package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/psadi/warp/internal/config"
	"github.com/psadi/warp/internal/fzf"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if err := fzf.EnsureFZFInstalled(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if handled := handlePreviewCommand(os.Args[1:]); handled {
		return
	}

	hosts, err := config.ParseSSHConfig()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("SSH config file not found at:", config.GetSSHConfigPath())
			fmt.Println("Please create your SSH config file first.")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Error reading SSH config:", err)
		os.Exit(1)
	}

	dispatch(os.Args[1], os.Args[2:], hosts)
}

func dispatch(cmd string, args []string, hosts []config.Host) {
	switch cmd {
	case "connect", "c":
		if hasHelpFlag(args) {
			printConnectHelp()
			return
		}
		connectToHost(hosts, parseConnectFlags(args))
	case "remove", "rm":
		if hasHelpFlag(args) {
			printRemoveHelp()
			return
		}
		removeHost(hosts)
	case "list", "ls":
		if hasHelpFlag(args) {
			printListHelp()
			return
		}
		listHosts(hosts)
	case "add", "a":
		if hasHelpFlag(args) {
			printAddHelp()
			return
		}
		addHost(hosts, parseFileFlag(args))
	case "export", "e":
		if hasHelpFlag(args) {
			printExportHelp()
			return
		}
		exportHosts(hosts, parseFileFlag(args))
	case "edit", "ed":
		if hasHelpFlag(args) {
			printEditHelp()
			return
		}
		editHost(hosts, parseHostFlag(args))
	case "shell-config", "completion":
		printShellConfig(args)
	case "--bash", "--zsh", "--fish":
		printShellConfig(append([]string{"--" + strings.TrimPrefix(cmd, "--")}, args...))
	case "help", "h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}
