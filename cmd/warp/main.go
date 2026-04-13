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

	cmd := os.Args[1]
	args := os.Args[2:]

	if cmd == "self-upgrade" || cmd == "upgrade" {
		if err := selfUpgrade(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}

	if cmd == "--version" || (len(args) > 0 && args[0] == "--version") {
		fmt.Println("warp version", getCurrentVersion())
		return
	}

	if cmd == "version" {
		fmt.Println("warp version", getCurrentVersion())
		return
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
			fmt.Println("Please create your SSH config first.")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Error reading SSH config:", err)
		os.Exit(1)
	}

	dispatch(cmd, args, hosts)
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
