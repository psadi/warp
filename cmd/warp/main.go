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
	fmt.Println("Warp! - SSH connection manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  warp connect [options]    Connect to a host (alias: c)")
	fmt.Println("  warp list                 List all hosts (alias: ls)")
	fmt.Println("  warp add [options]        Add a new host (alias: a)")
	fmt.Println("  warp edit [options]       Edit a host (alias: ed)")
	fmt.Println("  warp remove               Remove hosts (alias: rm)")
	fmt.Println("  warp export [options]     Export hosts to CSV (alias: e)")
	fmt.Println()
	fmt.Println("Run 'warp <command> --help' for more information on a command.")
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
		fmt.Println("No host selected")
		os.Exit(0)
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

	identityFile = readInput(reader, "Identity file (e.g., ~/.ssh/id_rsa): ")

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
	newIdentity := readInput(reader, fmt.Sprintf("Identity file (current: %s): ", currentIdentity))
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
