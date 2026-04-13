package main

type connectFlags struct {
	sshDebug     bool
	sshExtraArgs string
	selectOnly   bool
}

type shellConfigFlags struct {
	shell   string
	install bool
	wrapSSH bool
}

func parseConnectFlags(args []string) connectFlags {
	return connectFlags{
		sshDebug:     hasFlag(args, "--ssh-debug"),
		sshExtraArgs: optionValue(args, "--ssh-extra-args"),
		selectOnly:   hasFlag(args, "--select"),
	}
}

func parseFileFlag(args []string) string {
	return optionValue(args, "--file", "-f")
}

func parseHostFlag(args []string) string {
	return optionValue(args, "--host", "-n")
}

func parseShellFlags(args []string) shellConfigFlags {
	flags := shellConfigFlags{}
	for _, arg := range args {
		switch arg {
		case "--install", "-i", "install":
			flags.install = true
		case "--wrap-ssh":
			flags.wrapSSH = true
		case "--bash", "bash":
			flags.shell = "bash"
		case "--zsh", "zsh":
			flags.shell = "zsh"
		case "--fish", "fish":
			flags.shell = "fish"
		}
	}
	return flags
}

func hasHelpFlag(args []string) bool {
	return hasFlag(args, "--help", "-h")
}

func hasFlag(args []string, names ...string) bool {
	for _, arg := range args {
		for _, name := range names {
			if arg == name {
				return true
			}
		}
	}
	return false
}

func optionValue(args []string, names ...string) string {
	for i := 0; i < len(args); i++ {
		for _, name := range names {
			if args[i] == name && i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}
