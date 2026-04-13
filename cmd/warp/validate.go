package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

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

func readInput(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
	return strings.TrimSpace(input)
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
