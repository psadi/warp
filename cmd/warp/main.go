package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/psadi/warp/internal/config"
	"github.com/psadi/warp/internal/fzf"
	"github.com/psadi/warp/internal/ssh"
)

type connectFlags struct {
	sshDebug     bool
	sshExtraArgs string
	selectOnly   bool
}

func validatePort(port string) error {
	if port == "" {
		return nil
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port must be a number")
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func sanitizeSSHConfigValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", "")
	if strings.Contains(value, " ") || strings.Contains(value, "\"") {
		value = "\"" + strings.ReplaceAll(value, "\"", "\\\"") + "\""
	}
	return value
}

func validateHostName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("host name cannot be empty")
	}
	if strings.Contains(name, "\n") || strings.Contains(name, "\r") {
		return fmt.Errorf("host name cannot contain newlines")
	}
	if matched, _ := regexp.MatchString(`^\s|\s$`, name); matched {
		return fmt.Errorf("host name cannot have leading/trailing whitespace")
	}
	return nil
}

func validateHostNameForSSH(name string) error {
	if strings.Contains(name, " ") || strings.Contains(name, "\t") {
		return fmt.Errorf("host name cannot contain whitespace")
	}
	return nil
}

func getSSHKeys() []string {
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")

	var keys []string
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return keys
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".pub") || name == "known_hosts" || name == "config" || name == "authorized_keys" {
			continue
		}
		privKey := filepath.Join(sshDir, name)
		pubKey := privKey + ".pub"
		if _, err := os.Stat(pubKey); err == nil {
			keys = append(keys, name)
		}
	}
	return keys
}

func promptIdentityFile(reader *bufio.Reader, currentValue string) string {
	keys := getSSHKeys()

	fmt.Println("\nAvailable SSH keys in ~/.ssh/:")

	var options []string
	if currentValue != "" {
		options = append(options, "Keep current: "+currentValue)
	}
	options = append(options, "(none)")
	for _, key := range keys {
		displayPath := "~/.ssh/" + key
		if currentValue == displayPath || currentValue == key || currentValue == "~/.ssh/"+key {
			options = append(options, displayPath+" (current)")
		} else {
			options = append(options, displayPath)
		}
	}
	options = append(options, "Enter custom path")

	fmt.Println()
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Println()

	fmt.Print("Select option (1-" + strconv.Itoa(len(options)) + "): ")
	choice := readInput(reader, "")
	num, err := strconv.Atoi(choice)
	if err != nil || num < 1 || num > len(options) {
		fmt.Println("Invalid selection, using (none)")
		return ""
	}

	selected := options[num-1]

	if strings.HasPrefix(selected, "Keep current:") {
		return currentValue
	}
	if selected == "(none)" {
		return ""
	}
	if selected == "Enter custom path" {
		fmt.Print("Enter custom identity file path: ")
		return readInput(reader, "")
	}

	return selected
}

func readInput(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
	return strings.TrimSpace(input)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if err := fzf.EnsureFZFInstalled(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if os.Args[1] == "--preview" && len(os.Args) == 3 {
		hosts, err := config.ParseSSHConfig()
		if err != nil {
			fmt.Println("Error reading config:", err)
			os.Exit(1)
		}
		showPreview(hosts, os.Args[2])
		os.Exit(0)
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

	cmd := os.Args[1]

	switch cmd {
	case "connect", "c":
		if hasHelpFlag(os.Args[2:]) {
			printConnectHelp()
			return
		}
		flags := parseConnectFlags()
		connectToHost(hosts, flags)
	case "remove", "rm":
		if hasHelpFlag(os.Args[2:]) {
			printRemoveHelp()
			return
		}
		removeHost(hosts)
	case "list", "ls":
		if hasHelpFlag(os.Args[2:]) {
			printListHelp()
			return
		}
		listHosts(hosts)
	case "add", "a":
		if hasHelpFlag(os.Args[2:]) {
			printAddHelp()
			return
		}
		csvFile := parseAddFlags()
		addHost(hosts, csvFile)
	case "export", "e":
		if hasHelpFlag(os.Args[2:]) {
			printExportHelp()
			return
		}
		exportFile := parseExportFlags()
		exportHosts(hosts, exportFile)
	case "edit", "ed":
		if hasHelpFlag(os.Args[2:]) {
			printEditHelp()
			return
		}
		hostName := parseEditFlags()
		editHost(hosts, hostName)
	case "shell-config", "completion":
		printShellConfig(os.Args[2:])
	case "--bash", "--zsh", "--fish":
		shell := strings.TrimPrefix(cmd, "--")
		printShellConfig(append([]string{"--" + shell}, os.Args[2:]...))
	case "help", "h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func parseConnectFlags() connectFlags {
	flags := connectFlags{}
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ssh-debug":
			flags.sshDebug = true
		case "--ssh-extra-args":
			if i+1 < len(args) {
				flags.sshExtraArgs = args[i+1]
				i++
			}
		case "--select":
			flags.selectOnly = true
		}
	}
	return flags
}

func parseAddFlags() string {
	var csvFile string
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--file" || args[i] == "-f" {
			if i+1 < len(args) {
				csvFile = args[i+1]
				i++
			}
		}
	}
	return csvFile
}

func parseExportFlags() string {
	var exportFile string
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--file" || args[i] == "-f" {
			if i+1 < len(args) {
				exportFile = args[i+1]
				i++
			}
		}
	}
	return exportFile
}

func parseEditFlags() string {
	var hostName string
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--host" || args[i] == "-n" {
			if i+1 < len(args) {
				hostName = args[i+1]
				i++
			}
		}
	}
	return hostName
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

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
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  warp connect")
	fmt.Println("  warp c --ssh-debug")
	fmt.Println("  warp connect --ssh-extra-args \"-o StrictHostKeyChecking=no\"")
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
	fmt.Println("Examples:")
	fmt.Println("  warp add                  # Interactive prompts")
	fmt.Println("  warp a --file hosts.csv")
	fmt.Println()
	fmt.Println("CSV format:")
	fmt.Println("  host,hostname,user,port,identity_file")
}

func printRemoveHelp() {
	fmt.Println("Remove SSH hosts from config (multi-select with FZF)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp remove               (alias: rm)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --help, -h              Show this help")
}

func printExportHelp() {
	fmt.Println("Export SSH hosts to CSV file")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp export [options]    (alias: e)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --help, -h              Show this help")
	fmt.Println("  --file, -f <path>       Output file path (default: ./warp_export.csv)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  warp export")
	fmt.Println("  warp e --file ~/hosts.csv")
}

func printEditHelp() {
	fmt.Println("Edit an existing SSH host configuration")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp edit [options]      (alias: ed)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --help, -h              Show this help")
	fmt.Println("  --host, -n <name>       Host name to edit (uses FZF if not specified)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  warp edit")
	fmt.Println("  warp edit -n myserver")
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
	fmt.Println("  warp --zsh [-i]         Show/install zsh config")
	fmt.Println("  warp --bash [-i]        Show/install bash config")
	fmt.Println("  warp --fish [-i]        Show/install fish config")
	fmt.Println()
	fmt.Println("Run 'warp <command> --help' for more information on a command.")
}

func installCompletions(shell, exePath string) {
	switch shell {
	case "bash":
		home, _ := os.UserHomeDir()
		bashrc := filepath.Join(home, ".bashrc")
		content := fmt.Sprintf(`# Warp shell integration for bash
# Added by warp
alias c='%s connect'
alias cl='%s list'
alias ca='%s add'
alias ce='%s edit'
alias cr='%s remove'

ssh() {
    local host=$(%s connect 2>/dev/null)
    if [ -n "$host" ]; then
        command ssh "$host" "$@"
    fi
}

# Warp completions
source ~/.local/share/warp/completions/warp.bash
`, exePath, exePath, exePath, exePath, exePath, exePath)

		installPath := filepath.Join(home, ".local", "share", "warp", "completions")
		os.MkdirAll(installPath, 0755)
		os.WriteFile(filepath.Join(installPath, "warp.bash"), []byte(`_warp_hosts() {
    local cur=${COMP_WORDS[COMP_CWORD]}
    local hosts=$(warp list 2>/dev/null | tail -n +3 | awk '{print $1}')
    COMPREPLY=( $(compgen -W "$hosts" -- "$cur") )
}

_warp() {
    local commands="connect c list ls add a edit ed remove rm export e shell-config"
    local cur=${COMP_WORDS[COMP_CWORD]}
    COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
}

complete -F _warp warp
complete -F _warp_hosts c
`), 0644)

		if _, err := os.Stat(bashrc); os.IsNotExist(err) {
			os.WriteFile(bashrc, []byte(content), 0644)
		} else {
			data, _ := os.ReadFile(bashrc)
			if !strings.Contains(string(data), "# Warp shell integration") {
				f, _ := os.OpenFile(bashrc, os.O_APPEND|os.O_WRONLY, 0644)
				f.WriteString("\n" + content)
				f.Close()
			}
		}
		fmt.Printf("✓ Installed bash completions\n")
		fmt.Printf("  Config: %s\n", bashrc)

	case "zsh":
		home, _ := os.UserHomeDir()
		zshrc := filepath.Join(home, ".zshrc")
		compDir := filepath.Join(home, ".local", "share", "zsh", "site-functions")
		os.MkdirAll(compDir, 0755)

		content := fmt.Sprintf(`# Warp shell integration for zsh
# Added by warp
alias c='%s connect'
alias cl='%s list'
alias ca='%s add'
alias ce='%s edit'
alias cr='%s remove'

ssh() {
    local host=$(%s connect 2>/dev/null)
    if [ -n "$host" ]; then
        command ssh "$host" "$@"
    fi
}

# Warp completions
fpath=(~/.local/share/warp/completions $fpath)
autoload -Uz _warp
`, exePath, exePath, exePath, exePath, exePath, exePath)

		installPath := filepath.Join(home, ".local", "share", "warp", "completions")
		os.MkdirAll(installPath, 0755)
		os.WriteFile(filepath.Join(installPath, "_warp"), []byte(`#compdef _warp warp

_warp_hosts() {
    local -a hosts
    hosts=($(warp list 2>/dev/null | tail -n +3 | awk '{print $1}'))
    _describe 'hosts' hosts
}

_warp_commands() {
    local -a commands
    commands=(
        'connect:Connect to a host'
        'list:List all hosts'
        'add:Add a host'
        'edit:Edit a host'
        'remove:Remove hosts'
        'export:Export to CSV'
        'shell-config:Shell integration'
    )
    _describe 'commands' commands
}

_warp() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '-h[Show help]' \
        '--help[Show help]' \
        '*::command:_warp_commands'
}

_warp
`), 0644)

		if _, err := os.Stat(zshrc); os.IsNotExist(err) {
			os.WriteFile(zshrc, []byte(content), 0644)
		} else {
			data, _ := os.ReadFile(zshrc)
			if !strings.Contains(string(data), "# Warp shell integration") {
				f, _ := os.OpenFile(zshrc, os.O_APPEND|os.O_WRONLY, 0644)
				f.WriteString("\n" + content)
				f.Close()
			}
		}

		os.WriteFile(filepath.Join(compDir, "_warp"), []byte(`#compdef _warp warp
_warp_hosts() {
    local -a hosts
    hosts=($(warp list 2>/dev/null | tail -n +3 | awk '{print $1}'))
    _describe 'hosts' hosts
}
_warp_commands() {
    local -a commands
    commands=(
        'connect:Connect to a host'
        'list:List all hosts'
        'add:Add a host'
        'edit:Edit a host'
        'remove:Remove hosts'
        'export:Export to CSV'
        'shell-config:Shell integration'
    )
    _describe 'commands' commands
}
_warp() {
    local curcontext="$curcontext" state line
    typeset -A opt_args
    _arguments -C '-h[Show help]' '--help[Show help]' '*::command:_warp_commands'
}
_warp
`), 0644)

		fmt.Printf("✓ Installed zsh completions\n")
		fmt.Printf("  Config: %s\n", zshrc)
		fmt.Printf("  Completions: %s/_warp\n", compDir)

	case "fish":
		home, _ := os.UserHomeDir()
		fishConfig := filepath.Join(home, ".config", "fish", "config.fish")
		compDir := filepath.Join(home, ".config", "fish", "completions")
		os.MkdirAll(compDir, 0755)

		content := fmt.Sprintf(`# Warp shell integration for fish
# Added by warp
alias c='%s connect'
alias cl='%s list'
alias ca='%s add'
alias ce='%s edit'
alias cr='%s remove'

function ssh
    set -l host (%s connect 2>/dev/null)
    if test -n "$host"
        command ssh $host $argv
    end
end
`, exePath, exePath, exePath, exePath, exePath, exePath)

		if _, err := os.Stat(fishConfig); os.IsNotExist(err) {
			os.MkdirAll(filepath.Join(home, ".config", "fish"), 0755)
			os.WriteFile(fishConfig, []byte(content), 0644)
		} else {
			data, _ := os.ReadFile(fishConfig)
			if !strings.Contains(string(data), "# Warp shell integration") {
				f, _ := os.OpenFile(fishConfig, os.O_APPEND|os.O_WRONLY, 0644)
				f.WriteString("\n" + content)
				f.Close()
			}
		}

		os.WriteFile(filepath.Join(compDir, "warp.fish"), []byte(`complete -c warp -f -a "connect c list ls add a edit ed remove rm export e shell-config"

function __warp_hosts
    warp list 2>/dev/null | tail -n +3 | awk '{print $1}' | while read -l host
        echo $host
    end
end

complete -c warp -f -n "__fish_seen_subcommand_from connect" -a "(__warp_hosts)"
complete -c warp -f -n "__fish_seen_subcommand_from c" -a "(__warp_hosts)"
`), 0644)

		fmt.Printf("✓ Installed fish completions\n")
		fmt.Printf("  Config: %s\n", fishConfig)
		fmt.Printf("  Completions: %s/warp.fish\n", compDir)
	}

	fmt.Println()
	fmt.Println("Restart your shell or run:")
	fmt.Println("  source ~/.zshrc   # for zsh")
	fmt.Println("  source ~/.bashrc   # for bash")
}

func detectShell() string {
	sh := os.Getenv("SHELL")
	if sh != "" {
		if strings.Contains(sh, "zsh") {
			return "zsh"
		}
		if strings.Contains(sh, "bash") {
			return "bash"
		}
		if strings.Contains(sh, "fish") {
			return "fish"
		}
	}
	return "bash"
}

func printShellConfig(args []string) {
	shell := ""
	install := false

	for i, arg := range args {
		lowered := strings.ToLower(arg)

		if lowered == "--help" || lowered == "-h" {
			fmt.Println("Usage: warp --<shell> [-i]")
			fmt.Println()
			fmt.Println("Show or install shell integration config.")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  -i, --install   Install completions to system directories")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  warp --zsh               # Show zsh config")
			fmt.Println("  warp --zsh -i            # Install zsh completions")
			fmt.Println("  warp --bash -i           # Install bash completions")
			fmt.Println()
			fmt.Println("Shell completion paths:")
			fmt.Println("  bash: ~/.bashrc")
			fmt.Println("  zsh:  ~/.zshrc + ~/.local/share/zsh/site-functions/_warp")
			fmt.Println("  fish: ~/.config/fish/config.fish")
			os.Exit(0)
		}

		if lowered == "--install" || lowered == "-i" || lowered == "install" {
			install = true
			continue
		}

		if lowered == "--bash" || lowered == "--zsh" || lowered == "--fish" {
			shell = lowered[2:]
			continue
		}

		if lowered == "bash" || lowered == "zsh" || lowered == "fish" {
			if shell == "" {
				shell = lowered
			}
			continue
		}

		if i == 0 && shell == "" && install == false {
			shell = detectShell()
		}
	}

	if shell == "" {
		shell = detectShell()
	}

	valid := false
	for _, s := range []string{"bash", "zsh", "fish"} {
		if shell == s {
			valid = true
			break
		}
	}
	if !valid {
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", shell)
		fmt.Printf("Supported shells: %s\n", "bash, zsh, fish")
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		exePath = "warp"
	}

	if install {
		installCompletions(shell, exePath)
		return
	}

	switch shell {
	case "bash":
		output := strings.Replace(`# Warp shell integration for bash
# Auto-install completions if not present
if [ ! -f ~/.local/share/warp/completions/warp.bash ]; then
    mkdir -p ~/.local/share/warp/completions
    cat > ~/.local/share/warp/completions/warp.bash << 'EOFBASH'
_warp_hosts() {
    local cur=${COMP_WORDS[COMP_CWORD]}
    local hosts=$(warp list 2>/dev/null | tail -n +3 | awk '{print $1}')
    COMPREPLY=( $(compgen -W "$hosts" -- "$cur") )
}
_warp() {
    local commands="connect c list ls add a edit ed remove rm export e shell-config"
    local cur=${COMP_WORDS[COMP_CWORD]}
    COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
}
complete -F _warp warp
complete -F _warp_hosts c
EOFBASH
fi
source ~/.local/share/warp/completions/warp.bash 2>/dev/null

# Aliases
alias c='WARP_EXE connect'
alias cl='WARP_EXE list'
alias ca='WARP_EXE add'
alias ce='WARP_EXE edit'
alias cr='WARP_EXE remove'

# SSH wrapper
ssh() {
    local host=$(__SELECT_HOST__ 2>/dev/null)
    if [ -n "$host" ]; then
        command ssh "$host" "$@"
    fi
}
`, "WARP_EXE", exePath, -1)
		output = strings.Replace(output, "__SELECT_HOST__", exePath+" connect --select", -1)
		fmt.Print(output)
	case "zsh":
		output := strings.Replace(`# Warp shell integration for zsh
# Auto-install completions if not present
if [[ ! -f ~/.local/share/zsh/site-functions/_warp ]]; then
    mkdir -p ~/.local/share/zsh/site-functions
    cat > ~/.local/share/zsh/site-functions/_warp << 'EOFZSH'
#compdef _warp warp
_warp_hosts() {
    local -a hosts
    hosts=($(warp list 2>/dev/null | tail -n +3 | awk '{print $1}'))
    _describe 'hosts' hosts
}
_warp_commands() {
    local -a commands
    commands=(
        'connect:Connect to a host'
        'list:List all hosts'
        'add:Add a host'
        'edit:Edit a host'
        'remove:Remove hosts'
        'export:Export to CSV'
        'shell-config:Shell integration'
    )
    _describe 'commands' commands
}
_warp() {
    local curcontext="$curcontext" state line
    typeset -A opt_args
    _arguments -C '-h[Show help]' '--help[Show help]' '*::command:_warp_commands'
}
_warp
EOFZSH
fi

# Aliases
alias c='WARP_EXE connect'
alias cl='WARP_EXE list'
alias ca='WARP_EXE add'
alias ce='WARP_EXE edit'
alias cr='WARP_EXE remove'

# SSH wrapper
ssh() {
    local host=$(__SELECT_HOST__ 2>/dev/null)
    if [ -n "$host" ]; then
        command ssh "$host" "$@"
    fi
}
`, "WARP_EXE", exePath, -1)
		output = strings.Replace(output, "__SELECT_HOST__", exePath+" connect --select", -1)
		fmt.Print(output)
	case "fish":
		output := strings.Replace(`# Warp shell integration for fish
# Auto-install completions if not present
if not set -q WARP_INIT
    set -gx WARP_INIT true
    mkdir -p ~/.config/fish/completions
    cat > ~/.config/fish/completions/warp.fish << 'EOFFISH'
complete -c warp -f -a "connect c list ls add a edit ed remove rm export e shell-config"
function __warp_hosts
    warp list 2>/dev/null | tail -n +3 | awk '{print $1}' | while read -l host
        echo $host
    end
end
complete -c warp -f -n "__fish_seen_subcommand_from connect" -a "(__warp_hosts)"
complete -c warp -f -n "__fish_seen_subcommand_from c" -a "(__warp_hosts)"
EOFFISH
end

# Aliases
alias c='WARP_EXE connect'
alias cl='WARP_EXE list'
alias ca='WARP_EXE add'
alias ce='WARP_EXE edit'
alias cr='WARP_EXE remove'

# SSH wrapper
function ssh
    set -l host (__SELECT_HOST__ 2>/dev/null)
    if test -n "$host"
        command ssh $host $argv
    end
end
`, "WARP_EXE", exePath, -1)
		output = strings.Replace(output, "__SELECT_HOST__", exePath+" connect --select", -1)
		fmt.Print(output)
	}
}

func connectToHost(hosts []config.Host, flags connectFlags) {
	hostNames := config.GetHostNames(hosts)

	previewScript := generatePreviewScript()

	selected, err := fzf.Select(hostNames, fzf.NewOptions().
		WithPrompt("Select host> ").
		WithHeight("60%").
		WithPreviewWindow("right:50%:wrap").
		WithPreview(previewScript))

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if selected == "" {
		if flags.selectOnly {
			os.Exit(1)
		}
		fmt.Println("No host selected")
		os.Exit(0)
	}

	if flags.selectOnly {
		fmt.Print(selected)
		return
	}

	fmt.Println("\nConnecting to:", selected)
	if err := ssh.Connect(selected, flags.sshDebug, flags.sshExtraArgs); err != nil {
		fmt.Fprintln(os.Stderr, "Connection failed:", err)
		os.Exit(1)
	}
}

func listHosts(hosts []config.Host) {
	fmt.Println()
	fmt.Printf("%-20s %-25s %-15s %-6s %s\n", "HOST", "HOSTNAME", "USER", "PORT", "IDENTITY FILE")
	fmt.Println(strings.Repeat("-", 90))
	for _, h := range hosts {
		port := h.Port
		if port == "" {
			port = "22"
		}
		identity := h.IdentityFile
		if identity == "" {
			identity = "-"
		}
		fmt.Printf("%-20s %-25s %-15s %-6s %s\n", h.Name, h.HostName, h.User, port, identity)
	}
	fmt.Println()
}

func removeHost(hosts []config.Host) {
	hostNames := config.GetHostNames(hosts)

	selected, err := fzf.SelectMulti(hostNames, fzf.NewOptions().
		WithPrompt("Select hosts to remove> ").
		WithHeight("60%").
		WithPreviewWindow("right:50%:wrap").
		WithPreview(generatePreviewScript()))

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if len(selected) == 0 {
		fmt.Println("No hosts selected for removal")
		os.Exit(0)
	}

	fmt.Printf("\nRemoving %d host(s): %s\n", len(selected), strings.Join(selected, ", "))

	if err := config.RemoveHosts(selected); err != nil {
		fmt.Fprintln(os.Stderr, "Error removing hosts:", err)
		os.Exit(1)
	}

	fmt.Println("Hosts removed successfully!")
}

func addHost(hosts []config.Host, csvFile string) {
	if csvFile != "" {
		importFromCSV(hosts, csvFile)
		return
	}

	fmt.Println("Add a new SSH host configuration")
	fmt.Println("================================")
	fmt.Println()

	var name, hostname, user, port, identityFile string

	reader := bufio.NewReader(os.Stdin)

	name = readInput(reader, "Host name: ")

	if err := validateHostName(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := validateHostNameForSSH(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, h := range hosts {
		if strings.EqualFold(h.Name, name) {
			fmt.Printf("\nError: Host '%s' already exists in config.\n", name)
			os.Exit(1)
		}
	}

	hostname = readInput(reader, "Hostname (IP or domain): ")

	if hostname == "" {
		fmt.Fprintln(os.Stderr, "Error: hostname cannot be empty")
		os.Exit(1)
	}

	user = readInput(reader, "User: ")

	port = readInput(reader, "Port (default: 22): ")
	if port == "" {
		port = "22"
	}

	if err := validatePort(port); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	identityFile = promptIdentityFile(reader, "")

	newHost := config.Host{
		Name:         name,
		HostName:     hostname,
		User:         user,
		Port:         port,
		IdentityFile: identityFile,
	}

	if err := config.AddHost(newHost); err != nil {
		fmt.Fprintln(os.Stderr, "Error adding host:", err)
		os.Exit(1)
	}

	fmt.Println("\nHost added successfully!")
}

func importFromCSV(existingHosts []config.Host, csvFile string) {
	file, err := os.Open(csvFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening CSV file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading CSV file: %v\n", err)
		os.Exit(1)
	}

	if len(records) < 2 {
		fmt.Println("CSV file is empty or has no data rows")
		os.Exit(1)
	}

	header := records[0]
	if len(header) < 3 || header[0] != "host" {
		fmt.Println("Invalid CSV format. Expected header: host,hostname,user,port,identity_file")
		os.Exit(1)
	}

	existingSet := make(map[string]bool)
	for _, h := range existingHosts {
		existingSet[h.Name] = true
	}

	var newHosts []config.Host
	var skipped []string

	for i, record := range records[1:] {
		if len(record) < 3 {
			fmt.Printf("Skipping row %d: insufficient columns\n", i+2)
			continue
		}

		name := strings.TrimSpace(record[0])
		if err := validateHostName(name); err != nil {
			fmt.Printf("Skipping row %d: invalid host name - %v\n", i+2, err)
			continue
		}

		if existingSet[name] {
			skipped = append(skipped, name)
			continue
		}

		hostname := ""
		if len(record) > 1 {
			hostname = strings.TrimSpace(record[1])
		}

		if hostname == "" {
			fmt.Printf("Skipping row %d: empty hostname\n", i+2)
			continue
		}

		user := ""
		if len(record) > 2 {
			user = strings.TrimSpace(record[2])
		}

		port := "22"
		if len(record) > 3 && strings.TrimSpace(record[3]) != "" {
			port = strings.TrimSpace(record[3])
			if err := validatePort(port); err != nil {
				fmt.Printf("Skipping row %d: invalid port - %v\n", i+2, err)
				continue
			}
		}

		identityFile := ""
		if len(record) > 4 {
			identityFile = sanitizeSSHConfigValue(record[4])
		}

		newHosts = append(newHosts, config.Host{
			Name:         sanitizeSSHConfigValue(name),
			HostName:     sanitizeSSHConfigValue(hostname),
			User:         sanitizeSSHConfigValue(user),
			Port:         port,
			IdentityFile: identityFile,
		})
		existingSet[name] = true
	}

	if len(newHosts) == 0 {
		fmt.Println("No new hosts to add.")
		if len(skipped) > 0 {
			fmt.Printf("Skipped %d duplicate(s): %s\n", len(skipped), strings.Join(skipped, ", "))
		}
		return
	}

	for _, host := range newHosts {
		if err := config.AddHost(host); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding host %s: %v\n", host.Name, err)
			continue
		}
	}

	fmt.Printf("Successfully added %d host(s)\n", len(newHosts))
	if len(skipped) > 0 {
		fmt.Printf("Skipped %d duplicate(s): %s\n", len(skipped), strings.Join(skipped, ", "))
	}
}

func exportHosts(hosts []config.Host, exportFile string) {
	var exportPath string

	if exportFile != "" {
		exportPath = exportFile
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error getting current directory:", err)
			os.Exit(1)
		}
		exportPath = filepath.Join(cwd, "warp_export.csv")
	}

	file, err := os.Create(exportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating export file:", err)
		os.Exit(1)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"host", "hostname", "user", "port", "identity_file"})

	for _, h := range hosts {
		writer.Write([]string{h.Name, h.HostName, h.User, h.Port, h.IdentityFile})
	}

	fmt.Printf("Exported %d hosts to: %s\n", len(hosts), exportPath)
}

func editHost(hosts []config.Host, hostName string) {
	var targetHost *config.Host

	if hostName != "" {
		targetHost = config.GetHostByName(hosts, hostName)
		if targetHost == nil {
			fmt.Fprintf(os.Stderr, "Host '%s' not found in config\n", hostName)
			os.Exit(1)
		}
	} else {
		hostNames := config.GetHostNames(hosts)
		selected, err := fzf.Select(hostNames, fzf.NewOptions().
			WithPrompt("Select host to edit> ").
			WithHeight("60%").
			WithPreviewWindow("right:50%:wrap").
			WithPreview(generatePreviewScript()))

		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		if selected == "" {
			fmt.Println("No host selected")
			os.Exit(0)
		}

		targetHost = config.GetHostByName(hosts, selected)
	}

	fmt.Println("Edit SSH host configuration")
	fmt.Println("===========================")
	fmt.Printf("Editing: %s\n\n", targetHost.Name)

	reader := bufio.NewReader(os.Stdin)

	newName := readInput(reader, fmt.Sprintf("Host name (current: %s): ", targetHost.Name))
	if newName == "" {
		newName = targetHost.Name
	}

	if newName != targetHost.Name {
		if err := validateHostName(newName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := validateHostNameForSSH(newName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, h := range hosts {
			if strings.EqualFold(h.Name, newName) && h.Name != targetHost.Name {
				fmt.Fprintf(os.Stderr, "Error: Host '%s' already exists in config.\n", newName)
				os.Exit(1)
			}
		}
	}

	newHostname := readInput(reader, fmt.Sprintf("Hostname (current: %s): ", targetHost.HostName))
	if newHostname == "" {
		newHostname = targetHost.HostName
	}

	currentUser := targetHost.User
	newUser := readInput(reader, fmt.Sprintf("User (current: %s): ", currentUser))
	if newUser == "" {
		newUser = currentUser
	}

	currentPort := targetHost.Port
	if currentPort == "" {
		currentPort = "22"
	}
	newPort := readInput(reader, fmt.Sprintf("Port (current: %s): ", currentPort))
	if newPort == "" {
		newPort = currentPort
	}

	if newPort != "22" {
		if err := validatePort(newPort); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	currentIdentity := targetHost.IdentityFile
	newIdentity := promptIdentityFile(reader, currentIdentity)
	if newIdentity == "" {
		newIdentity = currentIdentity
	}

	newHost := config.Host{
		Name:         newName,
		HostName:     newHostname,
		User:         newUser,
		Port:         newPort,
		IdentityFile: newIdentity,
	}

	if err := config.UpdateHost(targetHost.Name, newHost); err != nil {
		fmt.Fprintln(os.Stderr, "Error updating host:", err)
		os.Exit(1)
	}

	fmt.Println("\nHost updated successfully!")
}

func generatePreviewScript() string {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "warp"
	}
	var sb strings.Builder
	sb.WriteString("'")
	sb.WriteString(execPath)
	sb.WriteString("' --preview {}")
	return sb.String()
}

func showPreview(hosts []config.Host, hostName string) {
	hostName = strings.TrimSpace(hostName)

	host := config.GetHostByName(hosts, hostName)
	if host == nil {
		fmt.Println("Host not found:", hostName)
		return
	}

	fmt.Println("Host:", host.Name)
	fmt.Println()
	if host.HostName != "" {
		fmt.Println("  HostName:", host.HostName)
	}
	if host.User != "" {
		fmt.Println("  User:", host.User)
	}
	port := host.Port
	if port == "" {
		port = "22"
	}
	if port != "22" {
		fmt.Println("  Port:", port)
	}
	if host.IdentityFile != "" {
		fmt.Println("  IdentityFile:", host.IdentityFile)
	}

	fmt.Println()
	fmt.Println("Testing connection...")

	if ssh.TestConnection(host.Name) {
		fmt.Println("  Status: Reachable")
	} else {
		fmt.Println("  Status: Unreachable or key not configured")
	}
}
