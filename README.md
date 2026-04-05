# Warp! - SSH Fuzzy launcher

![](https://img.shields.io/badge/license-MIT-green.svg?style=flat)
![](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)

_Select and connect to your servers with fuzzy search._

## Key Features

- Parses your existing `~/.ssh/config` file
- Supports SSH config `Include` directives (recursive)
- Fuzzy host selection with FZF
- Preview window showing host configuration and connection test
- SSH key selection from `~/.ssh/` with fingerprint preview
- Cross-platform: Linux, macOS, Windows
- CSV export/import for backup
- Host editing with current value display
- Shell integration with aliases and SSH wrapper

## Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/psadi/warp/main/install.sh | bash
```

This installs warp and fzf (if not present) to `~/.local/bin/`.

## Requirements

- FZF (fuzzy finder) - auto-installed if missing

## Build from Source

```bash
# Clone and build
git clone https://github.com/psadi/warp && cd warp
go build -o warp ./cmd/warp/

# Install globally (optional)
sudo mv warp /usr/local/bin/
```

### Release Build (smaller binary)

```bash
go build -ldflags="-s -w" -o warp ./cmd/warp/
```

## Usage

| Command | Description | Alias |
|---------|-------------|-------|
| `warp connect` | Interactive host selection with FZF | `c` |
| `warp connect --ssh-debug` | Enable SSH debug mode | - |
| `warp connect --ssh-extra-args "<args>"` | Pass extra SSH arguments | - |
| `warp list` | List all configured hosts | `ls` |
| `warp add` | Add new host interactively | `a` |
| `warp add --file <csv>` | Import hosts from CSV | - |
| `warp edit` | Edit host (FZF selection) | `ed` |
| `warp edit -n <host>` | Edit specific host | - |
| `warp remove` | Remove hosts (multi-select) | `rm` |
| `warp export` | Export hosts to CSV | `e` |
| `warp export --file <path>` | Export to specific path | - |

## Shell Integration

Add to your shell config for aliases and SSH wrapper:

```bash
# bash/zsh
eval "$(warp --zsh)"
eval "$(warp --bash)"

# fish
eval (warp --fish)
```

This adds:

- **Aliases**: `c` (connect), `cl` (list), `ca` (add), `ce` (edit), `cr` (remove)
- **SSH wrapper**: Run `ssh` to select and connect to a host
- **Completions**: Shell completions for commands and hosts (auto-installed)

Install completions separately:

```bash
warp --zsh -i      # Install zsh completions
warp --bash -i      # Install bash completions
warp --fish -i       # Install fish completions
```

## How It Works

Warp reads your SSH config from `~/.ssh/config` and provides an interactive interface using FZF for selecting hosts to connect to.

## Disclaimer

This tool assumes you already have SSH key-based authentication configured and will not create or manage keys.

## Contributing

Bug reports and pull requests are welcome on GitHub at [warp](https://github.com/psadi/warp) repository.

## Author

- **psadi** - _Owner_ - [psadi](https://github.com/psadi)

---

The project is available as open source under the terms of the [MIT License](LICENSE)
