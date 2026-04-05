package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Host struct {
	Name         string
	HostName     string
	User         string
	Port         string
	IdentityFile string
}

func GetSSHConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		if runtime.GOOS == "windows" {
			homeDir = os.Getenv("USERPROFILE")
		} else {
			homeDir = os.Getenv("HOME")
		}
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(homeDir, "ssh", "config")
	}
	return filepath.Join(homeDir, ".ssh", "config")
}

func ParseSSHConfig() ([]Host, error) {
	configPath := GetSSHConfigPath()
	return parseSSHConfigFile(configPath, nil)
}

func parseSSHConfigFile(filePath string, visitedPaths map[string]bool) ([]Host, error) {
	if visitedPaths == nil {
		visitedPaths = make(map[string]bool)
	}

	absPath, err := filepath.Abs(filePath)
	if err == nil {
		if visitedPaths[absPath] {
			return nil, nil
		}
		visitedPaths[absPath] = true
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var hosts []Host
	var currentHost *Host
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := strings.Join(parts[1:], " ")

		switch strings.ToLower(key) {
		case "include":
			includePath := expandPath(value, filepath.Dir(filePath))
			includedHosts, err := parseSSHConfigFile(includePath, visitedPaths)
			if err == nil {
				hosts = append(hosts, includedHosts...)
			}
		case "host":
			if currentHost != nil && currentHost.Name != "" {
				hosts = append(hosts, *currentHost)
			}
			currentHost = &Host{Name: value}
		case "hostname":
			if currentHost != nil {
				currentHost.HostName = value
			}
		case "user":
			if currentHost != nil {
				currentHost.User = value
			}
		case "port":
			if currentHost != nil {
				currentHost.Port = value
			}
		case "identityfile":
			if currentHost != nil {
				currentHost.IdentityFile = value
			}
		}
	}

	if currentHost != nil && currentHost.Name != "" {
		hosts = append(hosts, *currentHost)
	}

	return hosts, nil
}

func expandPath(path, baseDir string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return homeDir
			}
			if strings.HasPrefix(path, "~/") {
				return filepath.Join(homeDir, path[2:])
			}
			return filepath.Join(homeDir, path[1:])
		}
	}

	if !filepath.IsAbs(path) {
		return filepath.Join(baseDir, path)
	}
	return path
}

func GetHostNames(hosts []Host) []string {
	names := make([]string, len(hosts))
	for i, h := range hosts {
		names[i] = h.Name
	}
	return names
}

func GetSSHConnectString(host Host) string {
	if host.Port != "" && host.Port != "22" {
		return host.User + "@" + host.HostName + " -p " + host.Port
	}
	return host.User + "@" + host.HostName
}

func AddHost(host Host) error {
	configPath := GetSSHConfigPath()

	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	var sb strings.Builder
	sb.WriteString("\n\nHost ")
	sb.WriteString(host.Name)
	sb.WriteString("\n    HostName ")
	sb.WriteString(host.HostName)
	sb.WriteString("\n    User ")
	sb.WriteString(host.User)
	if host.Port != "" && host.Port != "22" {
		sb.WriteString("\n    Port ")
		sb.WriteString(host.Port)
	}
	if host.IdentityFile != "" {
		sb.WriteString("\n    IdentityFile ")
		sb.WriteString(host.IdentityFile)
	}
	sb.WriteString("\n")

	_, err = file.WriteString(sb.String())
	return err
}

func RemoveHosts(hostNames []string) error {
	configPath := GetSSHConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	backupPath := configPath + ".bak"
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	skipUntilNextHost := false

	hostSet := make(map[string]bool)
	for _, h := range hostNames {
		hostSet[strings.ToLower(h)] = true
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Host ") && !strings.HasPrefix(trimmed, "#") {
			hostName := strings.TrimSpace(strings.TrimPrefix(trimmed, "Host "))

			if hostSet[strings.ToLower(hostName)] {
				skipUntilNextHost = true
				continue
			} else {
				skipUntilNextHost = false
			}
		}

		if skipUntilNextHost {
			if i+1 < len(lines) {
				nextTrimmed := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextTrimmed, "Host ") && !strings.HasPrefix(nextTrimmed, "#") {
					skipUntilNextHost = false
				}
				continue
			}
		}

		if !skipUntilNextHost {
			newLines = append(newLines, line)
		}
	}

	err = os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0600)
	return err
}

func GetHostByName(hosts []Host, name string) *Host {
	lowerName := strings.ToLower(name)
	for i := range hosts {
		if strings.ToLower(hosts[i].Name) == lowerName {
			return &hosts[i]
		}
	}
	return nil
}

func UpdateHost(oldName string, newHost Host) error {
	configPath := GetSSHConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	inHostBlock := false
	oldNameLower := strings.ToLower(oldName)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Host ") && !strings.HasPrefix(trimmed, "#") {
			hostName := strings.TrimSpace(strings.TrimPrefix(trimmed, "Host "))

			if strings.ToLower(hostName) == oldNameLower {
				inHostBlock = true

				newLines = append(newLines, "Host "+newHost.Name)
				if newHost.HostName != "" {
					newLines = append(newLines, "    HostName "+newHost.HostName)
				}
				if newHost.User != "" {
					newLines = append(newLines, "    User "+newHost.User)
				}
				if newHost.Port != "" && newHost.Port != "22" {
					newLines = append(newLines, "    Port "+newHost.Port)
				}
				if newHost.IdentityFile != "" {
					newLines = append(newLines, "    IdentityFile "+newHost.IdentityFile)
				}
				continue
			} else {
				inHostBlock = false
			}
		}

		if inHostBlock {
			isHostKey := false
			upperLine := strings.ToUpper(trimmed)
			for _, key := range []string{"HOSTNAME", "USER", "PORT", "IDENTITYFILE"} {
				if strings.HasPrefix(upperLine, key+" ") || upperLine == key {
					isHostKey = true
					break
				}
			}
			if isHostKey {
				continue
			}
			if trimmed == "" && i+1 < len(lines) {
				nextTrimmed := strings.TrimSpace(lines[i+1])
				upperNext := strings.ToUpper(nextTrimmed)
				isNextHostKey := false
				for _, key := range []string{"HOST ", "HOSTNAME", "USER", "PORT", "IDENTITYFILE"} {
					if strings.HasPrefix(upperNext, key) {
						isNextHostKey = true
						break
					}
				}
				if !isNextHostKey {
					newLines = append(newLines, line)
				}
				continue
			}
			continue
		}

		newLines = append(newLines, line)
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0600)
}
