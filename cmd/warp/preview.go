package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/psadi/warp/internal/config"
)

func isIP(s string) bool {
	return net.ParseIP(s) != nil
}

const previewCacheTTL = 30 * time.Second

type previewCache struct {
	IPs        []string `json:"ips"`
	Reachable  bool     `json:"reachable"`
	Latency    int64    `json:"latency_ms"`
	AuthOK     bool     `json:"auth_ok"`
	Status     string   `json:"status"`
	AuthStatus string   `json:"auth_status"`
	SSHBanner  string   `json:"ssh_banner"`
	KnownHost  bool     `json:"known_host"`
	ProxyJump  string   `json:"proxy_jump"`
	CheckedAt  int64    `json:"checked_at"`
}

func handlePreviewCommand(args []string) bool {
	if len(args) == 2 && args[0] == "--preview" {
		hosts, err := config.ParseSSHConfig()
		if err != nil {
			fmt.Println("Error reading config:", err)
			os.Exit(1)
		}
		showPreview(hosts, args[1])
		return true
	}
	if len(args) == 2 && args[0] == "--preview-key" {
		showKeyPreview(args[1])
		return true
	}
	return false
}

func generateKeyPreviewScript() string {
	exePath, err := os.Executable()
	if err != nil {
		exePath = "warp"
	}
	return "'" + exePath + "' --preview-key {}"
}

func generatePreviewScript() string {
	exePath, err := os.Executable()
	if err != nil {
		exePath = "warp"
	}
	return "'" + exePath + "' --preview {}"
}

func showKeyPreview(keyName string) {
	keyName = strings.TrimSpace(keyName)
	if keyName == "(none)" || keyName == "" {
		fmt.Println("No key selected")
		return
	}

	home, _ := os.UserHomeDir()
	keyPath := keyName
	switch {
	case strings.HasPrefix(keyName, "~/.ssh/"):
		keyPath = filepath.Join(home, ".ssh", strings.TrimPrefix(keyName, "~/.ssh/"))
	case strings.HasPrefix(keyName, "Keep current:"):
		keyPath = strings.TrimPrefix(strings.TrimSpace(keyName), "Keep current:")
	case !strings.Contains(keyName, "/"):
		keyPath = filepath.Join(home, ".ssh", keyName)
	}

	keyPath = strings.ReplaceAll(keyPath, "~", home)
	fmt.Println("Key:", keyName)
	fmt.Println("Path:", keyPath)
	fmt.Println()

	output, err := exec.Command("ssh-keygen", "-l", "-f", keyPath).Output()
	if err == nil {
		fmt.Println(string(output))
		return
	}
	fmt.Println("Could not read key info")
}

func getKeyFingerprint(keyPath string) string {
	keyPath = expandTilde(keyPath)
	cmd := exec.Command("ssh-keygen", "-l", "-f", keyPath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	parts := strings.Fields(string(output))
	if len(parts) >= 4 {
		return strings.Join(parts[3:], " ")
	}
	return strings.TrimSpace(string(output))
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}

func showPreview(hosts []config.Host, hostName string) {
	host := config.GetHostByName(hosts, strings.TrimSpace(hostName))
	if host == nil {
		fmt.Println("Host not found:", hostName)
		return
	}

	info := readPreviewCache(host.Name)
	if info == nil || time.Since(time.Unix(info.CheckedAt, 0)) > previewCacheTTL {
		port := host.Port
		if port == "" {
			port = "22"
		}
		targetHost := host.HostName
		if targetHost == "" {
			targetHost = host.Name
		}
		reachable, latency := checkTCP(targetHost, port, 2*time.Second)
		authOK, authStatus := checkSSHAuth(host.Name, 3*time.Second)
		sshBanner := ""
		if reachable {
			sshBanner = getSSHBanner(targetHost, port, 2*time.Second)
		}
		knownHost := checkKnownHost(host.Name)
		proxyJump := getProxyJump(host.Name)
		status := "Unreachable"
		if reachable {
			status = "Reachable"
		}
		info = &previewCache{
			IPs:        lookupHostIPs(host.HostName),
			Reachable:  reachable,
			Latency:    latency,
			AuthOK:     authOK,
			Status:     status,
			AuthStatus: authStatus,
			SSHBanner:  sshBanner,
			KnownHost:  knownHost,
			ProxyJump:  proxyJump,
			CheckedAt:  time.Now().Unix(),
		}
		writePreviewCache(host.Name, info)
	}

	fmt.Println("Host:", host.Name)
	fmt.Println()
	if host.HostName != "" {
		fmt.Println("  HostName:", host.HostName)
		if len(info.IPs) > 0 {
			fmt.Printf("  IP: %s", info.IPs[0])
			if len(info.IPs) > 1 {
				fmt.Printf(" (+%d more)", len(info.IPs)-1)
			}
			fmt.Println()
		}
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
		if keyInfo := getKeyFingerprint(host.IdentityFile); keyInfo != "" {
			fmt.Println("    ", keyInfo)
		}
	} else if len(host.IdentityFiles) > 0 {
		fmt.Println("  IdentityFile:", strings.Join(host.IdentityFiles, ", "))
	}

	fmt.Println()
	fmt.Println("Connection Stats:")
	if info.Reachable {
		fmt.Printf("  TCP: %s (", info.Status)
		if info.Latency > 0 {
			fmt.Printf("%dms", info.Latency)
		} else {
			fmt.Printf("<3ms")
		}
		fmt.Println(")")
		fmt.Printf("  SSH: %s\n", info.AuthStatus)
		if info.SSHBanner != "" {
			fmt.Printf("  Server: %s\n", info.SSHBanner)
		}
		if info.KnownHost {
			fmt.Println("  KnownHost: yes")
		} else {
			fmt.Println("  KnownHost: no (new host)")
		}
		if info.ProxyJump != "" {
			fmt.Printf("  Proxy: %s\n", info.ProxyJump)
		}
	} else {
		fmt.Println("  TCP:", info.Status)
		fmt.Println("  SSH: skipped")
	}

	if len(info.IPs) > 0 && hasDig() {
		hasReverseDNS := false
		for _, ip := range info.IPs {
			if runDig(ip) != "" {
				hasReverseDNS = true
				break
			}
		}
		if hasReverseDNS {
			fmt.Println()
			fmt.Println("DNS:")
			for _, ip := range info.IPs {
				if digOutput := runDig(ip); digOutput != "" {
					fmt.Printf("  %s -> %s\n", ip, digOutput)
				}
			}
		}
	}
}

func checkTCP(host, port string, timeout time.Duration) (bool, int64) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false, -1
	}
	_ = conn.Close()
	latency := time.Since(start).Milliseconds()
	return true, latency
}

func getSSHBanner(host, port string, timeout time.Duration) string {
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return ""
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return ""
	}
	banner := strings.TrimSpace(string(buf[:n]))
	if strings.HasPrefix(banner, "SSH-") {
		parts := strings.Split(banner, " ")
		if len(parts) >= 3 {
			return parts[1] + " " + parts[2]
		}
		return parts[0]
	}
	return ""
}

func checkKnownHost(hostname string) bool {
	home, _ := os.UserHomeDir()
	knownHosts := filepath.Join(home, ".ssh", "known_hosts")
	data, err := os.ReadFile(knownHosts)
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 0 {
			hosts := strings.Split(parts[0], ",")
			for _, h := range hosts {
				h = strings.TrimSpace(h)
				if h == hostname || strings.HasPrefix(hostname, h+".") {
					return true
				}
			}
		}
	}
	return false
}

func getProxyJump(host string) string {
	cmd := exec.Command("ssh", "-G", host)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var proxyJump string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "proxyjump ") {
			proxyJump = strings.TrimPrefix(line, "proxyjump ")
			break
		}
		if strings.HasPrefix(line, "proxycommand ") {
			cmdStr := strings.TrimPrefix(line, "proxycommand ")
			if cmdStr == "/usr/bin/false" || cmdStr == "none" {
				continue
			}
			proxyJump = "(ProxyCommand)"
			break
		}
	}
	return proxyJump
}

func checkSSHAuth(host string, timeout time.Duration) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=3", "-o", "StrictHostKeyChecking=no", host, "exit")
	err := cmd.Run()
	if err == nil {
		return true, "Auth OK"
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()
		switch exitCode {
		case 0:
			return true, "Auth OK"
		case 1:
			return false, "Auth failed (bad credentials)"
		case 255:
			return false, "Auth failed (no key/refused)"
		default:
			return false, fmt.Sprintf("Auth failed (exit %d)", exitCode)
		}
	}
	return false, "Auth failed"
}

func lookupHostIPs(hostName string) []string {
	if hostName == "" {
		return nil
	}
	ips, err := net.LookupHost(hostName)
	if err != nil {
		return nil
	}
	return ips
}

func runDig(hostName string) string {
	cmd := exec.Command("dig", "+short", "+time=2", "+tries=1", hostName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func hasDig() bool {
	_, err := exec.LookPath("dig")
	return err == nil
}

func previewCachePath(hostName string) string {
	safe := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_").Replace(hostName)
	return filepath.Join(os.TempDir(), "warp-preview-"+safe+".json")
}

func readPreviewCache(hostName string) *previewCache {
	data, err := os.ReadFile(previewCachePath(hostName))
	if err != nil {
		return nil
	}
	var info previewCache
	if json.Unmarshal(data, &info) != nil {
		return nil
	}
	return &info
}

func writePreviewCache(hostName string, info *previewCache) {
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	_ = os.WriteFile(previewCachePath(hostName), data, 0600)
}
