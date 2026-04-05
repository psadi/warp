package ssh

import (
	"os"
	"os/exec"
	"strings"

	"github.com/psadi/warp/internal/config"
)

func Connect(host string, debug bool, extraArgs string) error {
	args := []string{}

	if debug {
		args = append(args, "-v")
	}

	if extraArgs != "" {
		extra := strings.Fields(extraArgs)
		args = append(args, extra...)
	}

	args = append(args, host)

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func TestConnection(host string) bool {
	cmd := exec.Command("ssh", "-o", "ConnectTimeout=5", "-o", "BatchMode=yes", host, "exit")
	err := cmd.Run()
	return err == nil
}

func GetSSHCommand(host string) string {
	return "ssh " + host
}

func GetSSHConfigLocation() string {
	return config.GetSSHConfigPath()
}

func FormatHostPreview(host string) string {
	var sb strings.Builder
	sb.WriteString("Host: " + host + "\n")
	sb.WriteString("Command: ssh " + host + "\n")
	sb.WriteString("\nConnection Test:\n")

	if TestConnection(host) {
		sb.WriteString("  Status: Reachable\n")
	} else {
		sb.WriteString("  Status: Unreachable or key not configured\n")
	}

	return sb.String()
}
