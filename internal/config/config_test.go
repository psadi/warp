package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0755); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}

	configPath := filepath.Join(sshDir, "config")
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return configPath
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

func TestRemoveHostsPreservesUnrelatedText(t *testing.T) {
	configPath := writeTestConfig(t, `# global comment
Include ~/.ssh/conf.d/*.conf

Host keep
    HostName keep.example.com
    User keep-user

# host comment
Host remove-me
    HostName old.example.com
    User old-user
    LocalForward 8080 localhost:80

Match exec "true"
    User root
`)

	if err := RemoveHosts([]string{"remove-me"}); err != nil {
		t.Fatalf("RemoveHosts: %v", err)
	}

	got := readTestFile(t, configPath)
	if strings.Contains(got, "Host remove-me") {
		t.Fatalf("removed host still present:\n%s", got)
	}
	if !strings.Contains(got, "Host keep") {
		t.Fatalf("kept host missing:\n%s", got)
	}
	if !strings.Contains(got, "Match exec \"true\"") {
		t.Fatalf("non-host block not preserved:\n%s", got)
	}

	backup := readTestFile(t, configPath+".bak")
	if !strings.Contains(backup, "Host remove-me") {
		t.Fatalf("backup missing original content:\n%s", backup)
	}
}

func TestUpdateHostReplacesOnlyTargetBlock(t *testing.T) {
	configPath := writeTestConfig(t, `Host first
    HostName first.example.com
    User alice
    ProxyJump bastion

Host second
    HostName second.example.com
    User bob

# trailing comment
`)

	err := UpdateHost("first", Host{
		Name:         "first",
		HostName:     "new.example.com",
		User:         "carol",
		Port:         "2222",
		IdentityFile: "~/.ssh/id_test",
	})
	if err != nil {
		t.Fatalf("UpdateHost: %v", err)
	}

	got := readTestFile(t, configPath)
	if !strings.Contains(got, "Host first\n    HostName new.example.com\n    User carol\n    Port 2222\n    IdentityFile ~/.ssh/id_test\n") {
		t.Fatalf("updated block not rendered as expected:\n%s", got)
	}
	if !strings.Contains(got, "Host second\n    HostName second.example.com\n    User bob\n") {
		t.Fatalf("other host changed unexpectedly:\n%s", got)
	}
	if !strings.Contains(got, "# trailing comment") {
		t.Fatalf("trailing text lost:\n%s", got)
	}

	backup := readTestFile(t, configPath+".bak")
	if !strings.Contains(backup, "ProxyJump bastion") {
		t.Fatalf("backup missing original block:\n%s", backup)
	}
}

func TestParseSSHConfigParsesIncludedFile(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(filepath.Join(sshDir, "conf.d"), 0755); err != nil {
		t.Fatalf("mkdir conf.d: %v", err)
	}

	mainConfig := `Include conf.d/extra

Host main
    HostName main.example.com
    User app
`
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(mainConfig), 0600); err != nil {
		t.Fatalf("write main config: %v", err)
	}

	includeConfig := `Host extra
    HostName extra.example.com
    User ops
`
	if err := os.WriteFile(filepath.Join(sshDir, "conf.d", "extra"), []byte(includeConfig), 0600); err != nil {
		t.Fatalf("write include config: %v", err)
	}

	hosts, err := ParseSSHConfig()
	if err != nil {
		t.Fatalf("ParseSSHConfig: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d: %#v", len(hosts), hosts)
	}
	if hosts[0].Name != "extra" || hosts[1].Name != "main" {
		t.Fatalf("unexpected host order: %#v", hosts)
	}
}

func TestParseSSHConfigSupportsGlobAliasesAndQuotedValues(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(filepath.Join(sshDir, "conf.d"), 0755); err != nil {
		t.Fatalf("mkdir conf.d: %v", err)
	}

	mainConfig := `Include conf.d/*

Host api app-* "quoted-host"
    HostName "quoted.example.com" # inline comment
    User deploy
    IdentityFile ~/.ssh/id_ed25519
    IdentityFile "~/.ssh/id_backup"
`
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(mainConfig), 0600); err != nil {
		t.Fatalf("write main config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "conf.d", "ignored"), []byte("Host db\n    HostName db.example.com\n"), 0600); err != nil {
		t.Fatalf("write include config: %v", err)
	}

	hosts, err := ParseSSHConfig()
	if err != nil {
		t.Fatalf("ParseSSHConfig: %v", err)
	}

	if len(hosts) != 3 {
		t.Fatalf("expected 3 concrete hosts, got %d: %#v", len(hosts), hosts)
	}
	if hosts[1].Name != "api" {
		t.Fatalf("expected concrete alias api, got %#v", hosts[1])
	}
	if hosts[2].Name != "quoted-host" {
		t.Fatalf("expected quoted alias preserved, got %#v", hosts[2])
	}
	if hosts[1].HostName != "quoted.example.com" {
		t.Fatalf("expected unquoted hostname, got %#v", hosts[1])
	}
	if len(hosts[1].IdentityFiles) != 2 {
		t.Fatalf("expected 2 identity files, got %#v", hosts[1].IdentityFiles)
	}
}

func TestUpdateHostSplitsGroupedAlias(t *testing.T) {
	configPath := writeTestConfig(t, `Host web blue
    HostName old.example.com
    User old
`)

	err := UpdateHost("blue", Host{
		Name:         "blue",
		HostName:     "blue.example.com",
		User:         "deploy",
		IdentityFile: "~/.ssh/id_blue",
	})
	if err != nil {
		t.Fatalf("UpdateHost: %v", err)
	}

	got := readTestFile(t, configPath)
	if !strings.Contains(got, "Host web\n") {
		t.Fatalf("expected remaining alias block, got:\n%s", got)
	}
	if !strings.Contains(got, "Host blue\n    HostName blue.example.com\n    User deploy\n    IdentityFile ~/.ssh/id_blue\n") {
		t.Fatalf("expected split updated block, got:\n%s", got)
	}
}
