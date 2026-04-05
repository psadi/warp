<p align="center">
<h1 align="center">Warp! - Your lazy command line ssh helper</h1>

![](https://img.shields.io/badge/license-MIT-green.svg?style=flat)
<a href="https://www.buymeacoffee.com/addy3494" target="_blank"><img src="https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go" ></a>

*Manage your infrastructure, Never loose/forget passwords again.*

***
## Key Features:
- Parses your existing `~/.ssh/config` file
- Supports SSH config `Include` directives (recursive)
- Interactive host selection with FZF
- Preview window showing host configuration
- Connection testing with timeout
- Cross-platform: Linux, macOS, Windows
- CSV export/import for backup
- Host editing with current value display

***
## Requirements
- Go 1.19+
- FZF (fuzzy finder)

### Install FZF
- **Linux**: `sudo apt install fzf` or use your package manager
- **macOS**: `brew install fzf`
- **Windows**: `choco install fzf` or `winget install fzf`

***
## Build

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

***
## Usage

```bash
# Connect to a host (interactive selection)
warp connect
warp c

# Connect with SSH debug
warp connect --ssh-debug

# Connect with extra SSH args
warp connect --ssh-extra-args "-o StrictHostKeyChecking=no"

# List all hosts
warp list
warp ls

# Add a new host (interactive prompts)
warp add
warp a

# Import hosts from CSV
warp add --file hosts.csv
warp a -f hosts.csv

# Edit an existing host
warp edit
warp ed

# Edit specific host
warp edit -n myserver
warp edit --host myserver

# Remove hosts (multi-select with FZF)
warp remove
warp rm

# Export hosts to CSV
warp export
warp e
warp export --file ~/hosts.csv

# Show help for any command
warp connect --help
warp add --help
```

***
## How It Works

Warp reads your SSH config from `~/.ssh/config` and provides an interactive interface using FZF for selecting hosts to connect to. No database required - it uses your existing SSH configuration.

***
## Disclaimer

### This repository,
* Is Created to tackle my personal use-case.
* Is not production ready/safe.
* Is just a wrapper *(quality-of-life improvements)* of the existing details which you already have.
* Assumes you already have available connection for key-based auth and will not create/establish any.
* Will not take any responsibility of damage-dealt/passwords-leaks etc. It is assumed you are using this package in a controlled environment.

***
## Contributing
Bug reports and pull requests are welcome on GitHub at [warp]( https://github.com/psadi/warp ) repository.

This project is intended to be a safe, welcoming space for collaboration and contributors are expected to adhere to the
[Contributor Covenant](http://contributor-covenant.org) code of conduct.

  1. Fork it ( https://github.com/psadi/warp )
  1. Create your feature branch (`git checkout -b my-new-feature`)
  1. Commit your changes (`git commit -am 'Add some feature'`)
  1. Push to the branch (`git push origin my-new-feature`)
  1. Create a new Pull Request

***

## Author
* **psadi** - *Owner* - [psadi](https://github.com/psadi)
***

The project is available as open source under the terms of the [MIT License](LICENSE)
