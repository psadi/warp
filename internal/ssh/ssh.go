package ssh

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

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
	return TestConnectionWithTimeout(host, 5)
}

func TestConnectionWithTimeout(host string, seconds int) bool {
	cmd := exec.Command("ssh", "-o", "ConnectTimeout="+strconv.Itoa(seconds), "-o", "BatchMode=yes", host, "exit")
	err := cmd.Run()
	return err == nil
}

func CheckReachability(host, fallbackHostName, fallbackPort string, timeout time.Duration) bool {
	targetHost, targetPort, proxy := ResolveEndpoint(host)
	if proxy {
		return true
	}
	if targetHost == "" {
		targetHost = fallbackHostName
	}
	if targetHost == "" {
		targetHost = host
	}
	if targetPort == "" {
		targetPort = fallbackPort
	}
	if targetPort == "" {
		targetPort = "22"
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(targetHost, targetPort), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func ResolveEndpoint(host string) (hostName string, port string, viaProxy bool) {
	output, err := exec.Command("ssh", "-G", host).Output()
	if err != nil {
		return "", "", false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")
		switch key {
		case "hostname":
			hostName = value
		case "port":
			port = value
		case "proxyjump", "proxycommand":
			if value != "" && value != "none" {
				viaProxy = true
			}
		}
	}

	return hostName, port, viaProxy
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
