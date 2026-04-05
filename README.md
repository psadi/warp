# Warp! - SSH Fuzzy launcher

![](https://img.shields.io/badge/license-MIT-green.svg?style=flat)

---

## Key Features

- Parses your existing `~/.ssh/config` file
- Supports SSH config `Include` directives (recursive)
- Fuzzy host selection with FZF
- Preview window showing host configuration
- Connection testing with timeout
- Cross-platform: Linux, macOS, Windows
- CSV export/import for backup
- Host editing with current value display
- Shell integration with aliases and SSH wrapper

---

## Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/psadi/warp/main/install.sh | bash
```

This installs warp and fzf (if not present) to `~/.local/bin/`.

---

## Requirements

- FZF (fuzzy finder) - auto-installed if missing

---

## Build from Source

```bash
# Clone and build
git clone https://github.com/psadi/warp && cd warp
go build -o warp ./cmd/warp/

# Install globally (optional)
sudo mv warp /usr/local/bin/

# Or add to PATH
export PATH=$PATH:$(pwd)
```

### Release Build (smaller binary)

```bash
go build -ldflags="-s -w" -o warp ./cmd/warp/
```

---

## Usage

### Connect

```bash
warp connect        # Interactive selection (alias: c)
warp connect --ssh-debug              # Enable SSH debug mode
warp connect --ssh-extra-args "-o StrictHostKeyChecking=no"
```

### List

```bash
warp list           # List all hosts (alias: ls)
```

### Add

```bash
warp add            # Interactive prompts (alias: a)
warp add --file hosts.csv            # Import from CSV
```

### Edit

```bash
warp edit           # Interactive selection (alias: ed)
warp edit -n myserver               # Edit specific host
```

### Remove

```bash
warp remove         # Multi-select with FZF (alias: rm)
```

### Export

```bash
warp export         # Export to CSV (alias: e)
warp export --file ~/hosts.csv
```

---

## Shell Integration

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
warp --bash -i     # Install bash completions
warp --fish -i     # Install fish completions
```

---

## How It Works

Warp reads your SSH config from `~/.ssh/config` and provides an interactive interface using FZF for selecting hosts to connect to. No database required - it uses your existing SSH configuration.

---

## Disclaimer

This tool assumes you already have SSH key-based authentication configured and will not create or manage keys.

---

## Contributing

Bug reports and pull requests are welcome on GitHub at [warp](https://github.com/psadi/warp) repository.

---

## Author

- **psadi** - _Owner_ - [psadi](https://github.com/psadi)

---

The project is available as open source under the terms of the [MIT License](LICENSE)
