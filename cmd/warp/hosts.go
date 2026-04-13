package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/psadi/warp/internal/config"
	"github.com/psadi/warp/internal/fzf"
	"github.com/psadi/warp/internal/ssh"
)

func connectToHost(hosts []config.Host, flags connectFlags) {
	selected, err := fzf.Select(config.GetHostNames(hosts), fzf.NewOptions().
		WithPrompt("Select host> ").
		WithHeight("60%").
		WithPreviewWindow("right:50%:wrap").
		WithPreview(generatePreviewScript()))
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
		if identity == "" && len(h.IdentityFiles) > 0 {
			identity = strings.Join(h.IdentityFiles, ", ")
		}
		if identity == "" {
			identity = "-"
		}
		fmt.Printf("%-20s %-25s %-15s %-6s %s\n", h.Name, h.HostName, h.User, port, identity)
	}
	fmt.Println()
}

func removeHost(hosts []config.Host) {
	selected, err := fzf.SelectMulti(config.GetHostNames(hosts), fzf.NewOptions().
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

	reader := bufio.NewReader(os.Stdin)
	name := readInput(reader, "Host name: ")
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

	hostname := readInput(reader, "Hostname (IP or domain): ")
	if hostname == "" {
		fmt.Fprintln(os.Stderr, "Error: hostname cannot be empty")
		os.Exit(1)
	}
	user := readInput(reader, "User: ")
	port := readInput(reader, "Port (default: 22): ")
	if port == "" {
		port = "22"
	}
	if err := validatePort(port); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	identityFile := promptIdentityFile("")
	newHost := config.Host{
		Name:          config.SanitizeValue(name),
		HostName:      config.SanitizeValue(hostname),
		User:          config.SanitizeValue(user),
		Port:          config.SanitizeValue(port),
		IdentityFile:  config.SanitizeValue(identityFile),
		IdentityFiles: singleIdentity(identityFile),
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
		existingSet[strings.ToLower(h.Name)] = true
	}

	var newHosts []config.Host
	var skipped []string
	for i, record := range records[1:] {
		if len(record) < 3 {
			fmt.Printf("Skipping row %d: insufficient columns\n", i+2)
			continue
		}

		name := config.SanitizeValue(record[0])
		if err := validateHostName(name); err != nil {
			fmt.Printf("Skipping row %d: invalid host name - %v\n", i+2, err)
			continue
		}
		if err := validateHostNameForSSH(name); err != nil {
			fmt.Printf("Skipping row %d: invalid host name - %v\n", i+2, err)
			continue
		}
		if existingSet[strings.ToLower(name)] {
			skipped = append(skipped, name)
			continue
		}

		hostname := ""
		if len(record) > 1 {
			hostname = config.SanitizeValue(record[1])
		}
		if hostname == "" {
			fmt.Printf("Skipping row %d: empty hostname\n", i+2)
			continue
		}

		user := ""
		if len(record) > 2 {
			user = config.SanitizeValue(record[2])
		}

		port := "22"
		if len(record) > 3 && strings.TrimSpace(record[3]) != "" {
			port = config.SanitizeValue(record[3])
			if err := validatePort(port); err != nil {
				fmt.Printf("Skipping row %d: invalid port - %v\n", i+2, err)
				continue
			}
		}

		identityFile := ""
		if len(record) > 4 {
			identityFile = config.SanitizeValue(record[4])
		}

		newHosts = append(newHosts, config.Host{
			Name:          name,
			HostName:      hostname,
			User:          user,
			Port:          port,
			IdentityFile:  identityFile,
			IdentityFiles: singleIdentity(identityFile),
		})
		existingSet[strings.ToLower(name)] = true
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
		}
	}

	fmt.Printf("Successfully added %d host(s)\n", len(newHosts))
	if len(skipped) > 0 {
		fmt.Printf("Skipped %d duplicate(s): %s\n", len(skipped), strings.Join(skipped, ", "))
	}
}

func exportHosts(hosts []config.Host, exportFile string) {
	exportPath := exportFile
	if exportPath == "" {
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
	_ = writer.Write([]string{"host", "hostname", "user", "port", "identity_file"})
	for _, h := range hosts {
		identity := h.IdentityFile
		if identity == "" && len(h.IdentityFiles) > 0 {
			identity = strings.Join(h.IdentityFiles, ";")
		}
		_ = writer.Write([]string{h.Name, h.HostName, h.User, h.Port, identity})
	}

	fmt.Printf("Exported %d hosts to: %s\n", len(hosts), exportPath)
}

func editHost(hosts []config.Host, hostName string) {
	targetHost := selectHostForEdit(hosts, hostName)

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
			if strings.EqualFold(h.Name, newName) && !strings.EqualFold(h.Name, targetHost.Name) {
				fmt.Fprintf(os.Stderr, "Error: Host '%s' already exists in config.\n", newName)
				os.Exit(1)
			}
		}
	}

	newHostname := readInput(reader, fmt.Sprintf("Hostname (current: %s): ", targetHost.HostName))
	if newHostname == "" {
		newHostname = targetHost.HostName
	}
	newUser := readInput(reader, fmt.Sprintf("User (current: %s): ", targetHost.User))
	if newUser == "" {
		newUser = targetHost.User
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
	if currentIdentity == "" && len(targetHost.IdentityFiles) > 0 {
		currentIdentity = targetHost.IdentityFiles[0]
	}
	newIdentity := promptIdentityFile(currentIdentity)
	if newIdentity == "" {
		newIdentity = currentIdentity
	}

	newHost := config.Host{
		Name:          config.SanitizeValue(newName),
		HostName:      config.SanitizeValue(newHostname),
		User:          config.SanitizeValue(newUser),
		Port:          config.SanitizeValue(newPort),
		IdentityFile:  config.SanitizeValue(newIdentity),
		IdentityFiles: singleIdentity(newIdentity),
	}
	if err := config.UpdateHost(targetHost.Name, newHost); err != nil {
		fmt.Fprintln(os.Stderr, "Error updating host:", err)
		os.Exit(1)
	}
	fmt.Println("\nHost updated successfully!")
}

func selectHostForEdit(hosts []config.Host, hostName string) *config.Host {
	if hostName != "" {
		targetHost := config.GetHostByName(hosts, hostName)
		if targetHost == nil {
			fmt.Fprintf(os.Stderr, "Host '%s' not found in config\n", hostName)
			os.Exit(1)
		}
		return targetHost
	}

	selected, err := fzf.Select(config.GetHostNames(hosts), fzf.NewOptions().
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
	return config.GetHostByName(hosts, selected)
}

func promptIdentityFile(currentValue string) string {
	keys := getSSHKeys()
	var options []string
	var keyPaths []string

	if currentValue != "" {
		options = append(options, "Keep current: "+currentValue)
		keyPaths = append(keyPaths, currentValue)
	}
	options = append(options, "(none)")
	keyPaths = append(keyPaths, "")
	for _, key := range keys {
		displayPath := "~/.ssh/" + key
		options = append(options, displayPath)
		keyPaths = append(keyPaths, displayPath)
	}

	fmt.Println("\nSelect SSH key:")
	selected, err := fzf.Select(options, fzf.NewOptions().
		WithPrompt("Select key> ").
		WithHeight("40%").
		WithPreviewWindow("right:50%:wrap").
		WithPreview(generateKeyPreviewScript()))
	if err != nil {
		fmt.Println("Selection cancelled")
		return ""
	}
	if selected == "" || selected == "(none)" {
		return ""
	}
	for i, opt := range options {
		if opt == selected {
			return keyPaths[i]
		}
	}
	return ""
}

func singleIdentity(identity string) []string {
	if identity == "" {
		return nil
	}
	return []string{identity}
}
