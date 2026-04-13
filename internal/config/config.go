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
	Name          string
	HostName      string
	User          string
	Port          string
	IdentityFile  string
	IdentityFiles []string
}

type configBlock struct {
	Raw         string
	HostAliases []string
	IsHost      bool
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
	return parseSSHConfigFile(GetSSHConfigPath(), nil)
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
	var currentHost Host
	var currentAliases []string
	scanner := bufio.NewScanner(file)

	flush := func() {
		for _, alias := range concreteAliases(currentAliases) {
			host := currentHost
			host.Name = alias
			if len(host.IdentityFiles) > 0 {
				host.IdentityFile = host.IdentityFiles[0]
			}
			hosts = append(hosts, host)
		}
		currentHost = Host{}
		currentAliases = nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		key, value, ok := splitSSHDirective(line)
		if !ok {
			continue
		}

		switch strings.ToLower(key) {
		case "include":
			for _, includePath := range parseIncludePaths(value, filepath.Dir(filePath)) {
				includedHosts, includeErr := parseSSHConfigFile(includePath, visitedPaths)
				if includeErr == nil {
					hosts = append(hosts, includedHosts...)
				}
			}
		case "host":
			flush()
			currentAliases = parseSSHWords(value)
		case "hostname":
			currentHost.HostName = SanitizeValue(value)
		case "user":
			currentHost.User = SanitizeValue(value)
		case "port":
			currentHost.Port = SanitizeValue(value)
		case "identityfile":
			identity := SanitizeValue(value)
			if identity != "" {
				currentHost.IdentityFiles = append(currentHost.IdentityFiles, identity)
				currentHost.IdentityFile = identity
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	flush()
	return hosts, nil
}

func expandPath(path, baseDir string) string {
	path = strings.TrimSpace(path)
	path = unquote(path)
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
	file, err := os.OpenFile(GetSSHConfigPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(RenderHostBlock(host))
	return err
}

func RemoveHosts(hostNames []string) error {
	configPath := GetSSHConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	hostSet := make(map[string]bool)
	for _, h := range hostNames {
		hostSet[strings.ToLower(h)] = true
	}

	var sb strings.Builder
	for _, block := range parseConfigBlocks(string(data)) {
		if !block.IsHost {
			sb.WriteString(block.Raw)
			continue
		}

		keepAliases := subtractAliases(block.HostAliases, hostSet)
		switch {
		case len(keepAliases) == len(block.HostAliases):
			sb.WriteString(block.Raw)
		case len(keepAliases) > 0:
			sb.WriteString(rewriteHostDeclaration(block.Raw, keepAliases))
		}
	}

	return writeConfigFile(configPath, []byte(sb.String()))
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

	oldNameLower := strings.ToLower(oldName)
	var sb strings.Builder
	replaced := false

	for _, block := range parseConfigBlocks(string(data)) {
		if !block.IsHost || !containsAlias(block.HostAliases, oldNameLower) {
			sb.WriteString(block.Raw)
			continue
		}

		replaced = true
		if len(block.HostAliases) == 1 {
			sb.WriteString(renderHostBlockWithoutLeadingSpacing(newHost))
			continue
		}

		keepAliases := removeAlias(block.HostAliases, oldNameLower)
		sb.WriteString(rewriteHostDeclaration(block.Raw, keepAliases))
		if !strings.HasSuffix(sb.String(), "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		sb.WriteString(renderHostBlockWithoutLeadingSpacing(newHost))
	}

	if !replaced {
		return fmt.Errorf("host %q not found", oldName)
	}
	return writeConfigFile(configPath, []byte(sb.String()))
}

func RenderHostBlock(host Host) string {
	return "\n\n" + renderHostBlockWithoutLeadingSpacing(host)
}

func SanitizeValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", "")
	return unquote(value)
}

func QuoteValue(value string) string {
	value = SanitizeValue(value)
	if value == "" {
		return value
	}
	if strings.ContainsAny(value, " \t\"") {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return value
}

func renderHostBlockWithoutLeadingSpacing(host Host) string {
	var sb strings.Builder
	sb.WriteString("Host ")
	sb.WriteString(QuoteValue(host.Name))
	sb.WriteString("\n")
	if host.HostName != "" {
		sb.WriteString("    HostName ")
		sb.WriteString(QuoteValue(host.HostName))
		sb.WriteString("\n")
	}
	if host.User != "" {
		sb.WriteString("    User ")
		sb.WriteString(QuoteValue(host.User))
		sb.WriteString("\n")
	}
	if host.Port != "" && host.Port != "22" {
		sb.WriteString("    Port ")
		sb.WriteString(QuoteValue(host.Port))
		sb.WriteString("\n")
	}
	identityFiles := host.IdentityFiles
	if len(identityFiles) == 0 && host.IdentityFile != "" {
		identityFiles = []string{host.IdentityFile}
	}
	for _, identity := range identityFiles {
		if identity == "" {
			continue
		}
		sb.WriteString("    IdentityFile ")
		sb.WriteString(QuoteValue(identity))
		sb.WriteString("\n")
	}
	return sb.String()
}

func writeConfigFile(configPath string, data []byte) error {
	current, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	backupPath := configPath + ".bak"
	if err := os.WriteFile(backupPath, current, 0600); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	return os.WriteFile(configPath, data, 0600)
}

func parseConfigBlocks(data string) []configBlock {
	lines := strings.SplitAfter(data, "\n")
	if len(lines) == 0 {
		return nil
	}

	var blocks []configBlock
	var current strings.Builder
	currentIsHost := false
	var currentAliases []string

	flush := func() {
		if current.Len() == 0 {
			return
		}
		blocks = append(blocks, configBlock{
			Raw:         current.String(),
			HostAliases: append([]string(nil), currentAliases...),
			IsHost:      currentIsHost,
		})
		current.Reset()
		currentIsHost = false
		currentAliases = nil
	}

	for _, line := range lines {
		if isHostDeclaration(line) {
			flush()
			currentIsHost = true
			currentAliases = parseHostAliasesFromLine(line)
		} else if isNonHostStanzaDeclaration(line) {
			flush()
		}
		current.WriteString(line)
	}
	flush()

	return blocks
}

func isHostDeclaration(line string) bool {
	key, _, ok := splitSSHDirective(line)
	return ok && strings.EqualFold(key, "Host")
}

func isNonHostStanzaDeclaration(line string) bool {
	key, _, ok := splitSSHDirective(line)
	return ok && strings.EqualFold(key, "Match")
}

func parseHostAliasesFromLine(line string) []string {
	_, value, ok := splitSSHDirective(line)
	if !ok {
		return nil
	}
	return parseSSHWords(value)
}

func parseIncludePaths(value, baseDir string) []string {
	words := parseSSHWords(value)
	var paths []string
	seen := make(map[string]bool)

	for _, word := range words {
		expanded := expandPath(word, baseDir)
		matches := []string{expanded}
		if hasGlobMeta(expanded) {
			globMatches, err := filepath.Glob(expanded)
			if err == nil && len(globMatches) > 0 {
				matches = globMatches
			}
		}
		for _, match := range matches {
			if !seen[match] {
				seen[match] = true
				paths = append(paths, match)
			}
		}
	}

	return paths
}

func splitSSHDirective(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(stripInlineComment(line))
	if trimmed == "" {
		return "", "", false
	}

	for i, r := range trimmed {
		if r == ' ' || r == '\t' {
			key := trimmed[:i]
			value := strings.TrimSpace(trimmed[i+1:])
			if value == "" {
				return "", "", false
			}
			return key, value, true
		}
	}
	return "", "", false
}

func stripInlineComment(line string) string {
	inSingle := false
	inDouble := false
	escaped := false
	for i, r := range line {
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == '#' && !inSingle && !inDouble:
			return line[:i]
		}
	}
	return line
}

func parseSSHWords(value string) []string {
	var words []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		words = append(words, current.String())
		current.Reset()
	}

	for _, r := range value {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case (r == ' ' || r == '\t') && !inSingle && !inDouble:
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()

	for i := range words {
		words[i] = SanitizeValue(words[i])
	}

	return words
}

func concreteAliases(aliases []string) []string {
	var concrete []string
	for _, alias := range aliases {
		if alias != "" && !hasPattern(alias) {
			concrete = append(concrete, alias)
		}
	}
	return concrete
}

func hasPattern(alias string) bool {
	return strings.ContainsAny(alias, "*?!")
}

func hasGlobMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func containsAlias(aliases []string, targetLower string) bool {
	for _, alias := range aliases {
		if strings.ToLower(alias) == targetLower {
			return true
		}
	}
	return false
}

func removeAlias(aliases []string, targetLower string) []string {
	var out []string
	for _, alias := range aliases {
		if strings.ToLower(alias) != targetLower {
			out = append(out, alias)
		}
	}
	return out
}

func subtractAliases(aliases []string, removeSet map[string]bool) []string {
	var out []string
	for _, alias := range aliases {
		if !removeSet[strings.ToLower(alias)] {
			out = append(out, alias)
		}
	}
	return out
}

func rewriteHostDeclaration(raw string, aliases []string) string {
	lines := strings.SplitAfter(raw, "\n")
	for i, line := range lines {
		if !isHostDeclaration(line) {
			continue
		}
		prefix := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		suffix := ""
		if strings.HasSuffix(line, "\n") {
			suffix = "\n"
		}
		lines[i] = prefix + "Host " + strings.Join(aliases, " ") + suffix
		break
	}
	return strings.Join(lines, "")
}

func unquote(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
