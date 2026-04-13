package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func detectShell() string {
	sh := os.Getenv("SHELL")
	switch {
	case strings.Contains(sh, "zsh"):
		return "zsh"
	case strings.Contains(sh, "fish"):
		return "fish"
	default:
		return "bash"
	}
}

func printShellConfig(args []string) {
	if hasHelpFlag(args) {
		fmt.Println("Usage: warp --<shell> [-i] [--wrap-ssh]")
		fmt.Println()
		fmt.Println("Show or install shell integration config.")
		fmt.Println("Use --wrap-ssh to override ssh command.")
		return
	}

	flags := parseShellFlags(args)
	if flags.shell == "" {
		flags.shell = detectShell()
	}
	if flags.shell != "bash" && flags.shell != "zsh" && flags.shell != "fish" {
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", flags.shell)
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		exePath = "warp"
	}

	if flags.install {
		installCompletions(flags, exePath)
		return
	}

	fmt.Print(renderShellConfig(flags, exePath))
}

func installCompletions(flags shellConfigFlags, exePath string) {
	home, _ := os.UserHomeDir()
	content := renderShellConfig(flags, exePath)

	switch flags.shell {
	case "bash":
		bashrc := filepath.Join(home, ".bashrc")
		writeIntegrationFile(filepath.Join(home, ".local", "share", "warp", "completions", "warp.bash"), []byte(bashCompletionScript()))
		appendIfMissing(bashrc, content)
		fmt.Printf("✓ Installed bash integration\n  Config: %s\n", bashrc)
	case "zsh":
		zshrc, completionPath := zshPaths(home)
		writeIntegrationFile(completionPath, []byte(zshCompletionScript()))
		appendIfMissing(zshrc, content)
		fmt.Printf("✓ Installed zsh integration\n  Config: %s\n  Completions: %s\n", zshrc, completionPath)
	case "fish":
		fishConfig := filepath.Join(home, ".config", "fish", "config.fish")
		writeIntegrationFile(filepath.Join(home, ".config", "fish", "completions", "warp.fish"), []byte(fishCompletionScript()))
		appendIfMissing(fishConfig, content)
		fmt.Printf("✓ Installed fish integration\n  Config: %s\n", fishConfig)
	}
}

func renderShellConfig(flags shellConfigFlags, exePath string) string {
	wrapperName := "wssh"
	if flags.wrapSSH {
		wrapperName = "ssh"
	}
	selectCmd := exePath + " connect --select"

	switch flags.shell {
	case "zsh":
		return fmt.Sprintf(`# Warp shell integration for zsh
fpath=(${ZDOTDIR:-$HOME}/completions $fpath)
autoload -Uz _warp 2>/dev/null

alias c='%s connect'
alias cl='%s list'
alias ca='%s add'
alias ce='%s edit'
alias cr='%s remove'

function %s {
    local host=$(%s 2>/dev/null)
    if [ -n "$host" ]; then
        command ssh "$host" "$@"
    fi
}
`, exePath, exePath, exePath, exePath, exePath, wrapperName, selectCmd)
	case "fish":
		return fmt.Sprintf(`# Warp shell integration for fish
alias c='%s connect'
alias cl='%s list'
alias ca='%s add'
alias ce='%s edit'
alias cr='%s remove'

function %s
    set -l host (%s 2>/dev/null)
    if test -n "$host"
        command ssh $host $argv
    end
end
`, exePath, exePath, exePath, exePath, exePath, wrapperName, selectCmd)
	default:
		return fmt.Sprintf(`# Warp shell integration for bash
source ~/.local/share/warp/completions/warp.bash 2>/dev/null

alias c='%s connect'
alias cl='%s list'
alias ca='%s add'
alias ce='%s edit'
alias cr='%s remove'

%s() {
    local host=$(%s 2>/dev/null)
    if [ -n "$host" ]; then
        command ssh "$host" "$@"
    fi
}
`, exePath, exePath, exePath, exePath, exePath, wrapperName, selectCmd)
	}
}

func appendIfMissing(path, content string) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.WriteFile(path, []byte(content), 0644)
		return
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "# Warp shell integration") {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString("\n" + content)
}

func writeIntegrationFile(path string, content []byte) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, content, 0644)
}

func zshPaths(home string) (zshrc string, completionPath string) {
	baseDir := os.Getenv("ZDOTDIR")
	if baseDir == "" {
		baseDir = home
	}
	return filepath.Join(baseDir, ".zshrc"), filepath.Join(baseDir, "completions", "_warp")
}

func bashCompletionScript() string {
	return `_warp_hosts() {
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
`
}

func zshCompletionScript() string {
	return `#compdef _warp warp
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
`
}

func fishCompletionScript() string {
	return `complete -c warp -f -a "connect c list ls add a edit ed remove rm export e shell-config"

function __warp_hosts
    warp list 2>/dev/null | tail -n +3 | awk '{print $1}' | while read -l host
        echo $host
    end
end

complete -c warp -f -n "__fish_seen_subcommand_from connect" -a "(__warp_hosts)"
complete -c warp -f -n "__fish_seen_subcommand_from c" -a "(__warp_hosts)"
`
}
