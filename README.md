# Cage

A cross-platform security sandbox CLI tool that executes commands with restricted file system write access while maintaining full read permissions.

## Overview

Cage provides a unified way to run potentially untrusted commands or scripts with file system write restrictions across Linux and macOS. It's designed for scenarios where you need to:
- Run untrusted code safely
- Limit file system modifications during development
- Analyze scripts without risk of system changes
- Process sensitive data with controlled output locations

## Features

- **Write-only restriction**: Commands can read any file but cannot write unless explicitly allowed
- **Cross-platform**: Works on Linux (kernel 5.13+) and macOS
- **Flexible permissions**: Grant write access to specific paths via `-allow` flags
- **Debug mode**: Disable all restrictions with `-allow-all`
- **Transparent execution**: Uses `syscall.Exec` to replace the process, preventing sandbox bypass

## Installation

```bash
go install github.com/Warashi/cage@latest
```

Or build from source:

```bash
git clone https://github.com/Warashi/cage
cd cage
go build
```

## Usage

### Basic Syntax

```bash
cage [flags] <command> [args...]
```

### Flags

- `-allow <path>`: Grant write access to a specific path (can be used multiple times)
- `-allow-keychain`: Allow write access to the macOS keychain (macOS only)
- `-allow-git`: Allow access to git common directory (enables git operations in worktrees)
- `-allow-all`: Disable all restrictions (useful for debugging)
- `-preset <name>`: Use a predefined preset configuration
- `-list-presets`: List available presets
- `-config <path>`: Path to custom configuration file

### Examples

#### Run a script with temporary directory access
```bash
cage -allow /tmp python analyze.py input.txt
# Note: On macOS, /tmp is a symlink to /private/tmp
# You may need to use: cage -allow /private/tmp python analyze.py input.txt
```

#### Build a project with restricted output directories
```bash
cage -allow ./build -allow ./dist -- npm run build
```

#### Analyze untrusted scripts safely
```bash
cage python suspicious_script.py
```

#### Process data with controlled output
```bash
cage -allow ./output -- ./process_data.sh /sensitive/data
```

#### Allow keychain access (macOS)
```bash
cage -allow-keychain -- security add-generic-password -s "MyService" -a "username" -w
```

#### Debug mode (no restrictions)
```bash
cage -allow-all -- make install
```

#### Enable git operations in worktrees
```bash
# Allow git operations when working in a git worktree
cage -allow-git -allow . -- git checkout -b new-feature
cage -allow-git -allow . -- git commit -m "Update files"
```

#### Using presets
```bash
# Use npm preset for Node.js development
cage -preset npm npm install

# Use cargo preset for Rust development
cage -preset cargo cargo build

# Combine preset with additional allow paths
cage -preset npm -allow ./logs npm run test

# List available presets
cage -list-presets

# Use custom configuration file
cage -config ~/my-presets.yaml -preset custom-preset ./script.sh
```

### Configuration File

Cage supports YAML configuration files to define presets. The configuration file is searched in the following order:

1. Path specified with `-config` flag
2. `$XDG_CONFIG_HOME/cage/presets.yaml` (or platform-specific config directory)
3. `$XDG_CONFIG_HOME/cage/presets.yml` (or platform-specific config directory)

The default config directory is:
- Linux: `~/.config/cage/`
- macOS: `~/Library/Application Support/cage/`
- Windows: `%APPDATA%\cage\`

Example configuration file:

```yaml
presets:
  npm:
    allow:
      - "."
      - "~/.npm"
      - "~/.cache/npm"
      - "~/.npmrc"
  
  cargo:
    allow:
      - "."
      - "~/.cargo"
      - "~/.rustup"
      - "~/.cache/sccache"
  
  git-enabled:
    allow:
      - "."
      - "~/.ssh"
    allow-git: true
    allow-keychain: true  # macOS only
  
  custom:
    allow:
      - "./output"
      - "/tmp"  # Note: On macOS, use /private/tmp instead
      - "~/.myapp"
```

Presets support the following options:
- `allow`: List of paths to grant write access
- `allow-git`: Enable access to git common directory (boolean)
- `allow-keychain`: Enable macOS keychain access (boolean)

## Platform Implementation

### Linux
- Uses [Landlock LSM](https://landlock.io/) via go-landlock
- Requires kernel 5.13 or later
- Grants read/execute access to entire filesystem
- Write access only to /dev/null and explicitly allowed paths

### macOS
- Uses `sandbox-exec` with custom sandbox profiles
- Generates sandbox profiles that deny all writes except to allowed paths
- Supports keychain access with `-allow-keychain` flag
- Handles path resolution and proper escaping

### Other Platforms
- Returns an error indicating sandboxing is not implemented

## Security Policy

Cage enforces the following security policy by default:

| Operation | Default Policy | With `-allow` |
|-----------|---------------|----------------|
| File Read | ✅ Allowed | ✅ Allowed |
| File Write | ❌ Denied | ✅ Allowed for specified paths |
| File Execute | ✅ Allowed | ✅ Allowed |
| Network Access | ✅ Allowed | ✅ Allowed |
| Process Creation | ✅ Allowed | ✅ Allowed |

## Development

### Building
```bash
go build
```

### Testing
Run the comprehensive end-to-end test suite:
```bash
./test_e2e.sh
```

### Nix Development Environment
The project includes Nix flakes for reproducible development:
```bash
nix develop
```

## Use Cases

### 1. Development Workflows
Restrict build outputs to specific directories:
```bash
cage -allow ./build -allow ./node_modules -- npm install
cage -allow ./dist -- npm run build
```

### 2. Security Testing
Safely analyze potentially malicious scripts:
```bash
cage python malware_sample.py
cage -allow /tmp/analysis -- ./suspicious_binary
```

### 3. Data Processing
Process sensitive data with controlled output locations:
```bash
cage -allow ./reports -- python generate_report.py /confidential/data.csv
```

### 4. LLM Code Agents
```bash
cage \
  -allow . \                                   # Allow current directory
  -allow /tmp \                                # Allow temporary directory (on macOS, use /private/tmp)
  -allow $HOME/.npm \                          # Allow npm directory for MCP server executed via npx command
  -allow "$CLAUDE_CONFIG_DIR" \                # Allow Claude config directory
  -allow "$(git rev-parse --git-common-dir)" \ # Allow git common directory
  -allow-keychain \                            # Allow keychain access (macOS)
  claude --dangerously-skip-permissions
```

## Limitations

- Sandboxing is only implemented for Linux and macOS
- Linux requires kernel 5.13 or later for Landlock support
- Network and process execution are not restricted
- Cannot restrict reads (by design - focuses on write-only restrictions)

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.

## Related Documentation

- [CLI Design Document](CLI_DESIGN.md) - Detailed design and implementation notes
