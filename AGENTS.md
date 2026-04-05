# Warp! Agent Guidance

## Essential Commands
- Build: `mise run build` or `go build -ldflags="-s -w" -o warp ./cmd/warp/`
- Run: `go run ./cmd/warp [options]` or `./warp [options]`
- FZF must be installed on the system

## Key Files
- Entry point: `cmd/warp/main.go`
- SSH Config Parser: `internal/config/config.go`
- SSH Connection: `internal/ssh/ssh.go`
- FZF Integration: `internal/fzf/fzf.go`
- Module: `go.mod`

## Usage

### Connect
- `warp connect [options]` (alias: `c`) - Connect to a selected host
  - `--ssh-debug` - Enable SSH debug mode
  - `--ssh-extra-args <args>` - Pass extra arguments to SSH

### List
- `warp list` (alias: `ls`) - List all connections

### Add
- `warp add [options]` (alias: `a`) - Add a new connection
  - `--file, -f <csv>` - Import hosts from CSV file

### Edit
- `warp edit [options]` (alias: `ed`) - Edit an existing host
  - `--host, -n <name>` - Host name to edit (uses FZF if not specified)

### Remove
- `warp remove` (alias: `rm`) - Remove connections (multi-select with FZF)

### Export
- `warp export [options]` (alias: `e`) - Export connections to CSV
  - `--file, -f <path>` - Output file path

### Shell Integration
- `warp --<shell>` - Show shell integration config
  - `warp --zsh` - Show zsh config
  - `warp --bash` - Show bash config
  - `warp --fish` - Show fish config
- `warp --<shell> -i` - Install completions to system directories
  - `warp --zsh -i` - Install zsh completions
  - `warp --bash -i` - Install bash completions
  - `warp --fish -i` - Install fish completions

## Features
- Parses `~/.ssh/config` for SSH host definitions
- Supports SSH config `Include` directives (recursive)
- FZF integration for interactive fuzzy host selection
- Preview window showing host configuration
- Connection testing with timeout
- Cross-platform: Linux, macOS, Windows
- CSV export/import for backup
- Host editing with current value display
- Shell integration with aliases and SSH wrapper

## Notes
- Pure Go implementation
- SSH config file location: `~/.ssh/config` (Linux/macOS), `%USERPROFILE%\ssh\config` (Windows)
- FZF must be installed separately on the system
- No database - uses native SSH config file
