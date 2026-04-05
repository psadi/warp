# Warp! Agent Guidance

## Essential Commands
- Build: `mise run build` or `go build -ldflags="-s -w" -o warp ./cmd/warp/ && upx warp`
- Run: `go run ./cmd/warp [options]`
- Install fzf: Required dependency for FZF integration

## Key Files
- Entry point: `cmd/warp/main.go`
- SSH Config Parser: `internal/config/config.go`
- SSH Connection: `internal/ssh/ssh.go`
- FZF Integration: `internal/fzf/fzf.go`
- Module: `go.mod`

## Usage
- `warp connect [options]` (alias: `c`) - Connect to a selected host
  - `--ssh-debug` - Enable SSH debug mode
  - `--ssh-extra-args <args>` - Pass extra arguments to SSH
- `warp list` (alias: `ls`) - List all connections
- `warp add` (alias: `a`) - Add a new connection
- `warp edit [options]` (alias: `ed`) - Edit an existing host
  - `--host, -n <name>` - Host name to edit (uses FZF if not specified)
- `warp remove` (alias: `rm`) - Remove connections (multi-select with FZF)
- `warp export` (alias: `e`) - Export connections to CSV

## Features
- Parses `~/.ssh/config` for SSH host definitions
- Supports SSH config `Include` directives (recursive)
- FZF integration for interactive host selection
- Preview window showing host configuration
- Connection testing with timeout
- Cross-platform: Linux, macOS, Windows
- CSV export/import for backup
- Host editing with current value display

## Notes
- Pure Go implementation, no external dependencies except FZF CLI tool
- SSH config file location: `~/.ssh/config` (Linux/macOS), `%USERPROFILE%\ssh\config` (Windows)
- FZF must be installed separately on the system
- No database - uses native SSH config file
