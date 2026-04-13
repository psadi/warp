package main

import "fmt"

func printConnectHelp() {
	fmt.Println("Connect to an SSH host using FZF selection")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp connect [options]    (alias: c)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --help, -h              Show this help")
	fmt.Println("  --ssh-debug             Enable SSH debug mode")
	fmt.Println("  --ssh-extra-args <args> Pass extra arguments to SSH")
	fmt.Println("  --select                Print selected host only")
}

func printListHelp() {
	fmt.Println("List all SSH hosts")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp list                 (alias: ls)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --help, -h              Show this help")
}

func printAddHelp() {
	fmt.Println("Add a new SSH host configuration")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp add [options]        (alias: a)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --help, -h              Show this help")
	fmt.Println("  --file, -f <csv>        Import hosts from CSV file")
	fmt.Println()
	fmt.Println("CSV format:")
	fmt.Println("  host,hostname,user,port,identity_file")
}

func printRemoveHelp() {
	fmt.Println("Remove SSH hosts from config (multi-select with FZF)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp remove               (alias: rm)")
}

func printExportHelp() {
	fmt.Println("Export SSH hosts to CSV file")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp export [options]    (alias: e)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --file, -f <path>       Output file path (default: ./warp_export.csv)")
}

func printEditHelp() {
	fmt.Println("Edit an existing SSH host configuration")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp edit [options]      (alias: ed)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --host, -n <name>       Host name to edit (uses FZF if not specified)")
}

func printUsage() {
	fmt.Println("Warp! - SSH Fuzzy launcher")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp connect [options]    Connect to a host (alias: c)")
	fmt.Println("  warp list                 List all hosts (alias: ls)")
	fmt.Println("  warp add [options]        Add a new host (alias: a)")
	fmt.Println("  warp edit [options]       Edit a host (alias: ed)")
	fmt.Println("  warp remove               Remove hosts (alias: rm)")
	fmt.Println("  warp export [options]     Export hosts to CSV (alias: e)")
	fmt.Println()
	fmt.Println("Shell Integration:")
	fmt.Println("  warp --zsh [-i] [--wrap-ssh]   Show/install zsh config")
	fmt.Println("  warp --bash [-i] [--wrap-ssh]  Show/install bash config")
	fmt.Println("  warp --fish [-i] [--wrap-ssh]  Show/install fish config")
}
