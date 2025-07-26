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

### Pre-built Binaries with Homebrew Cask
We doesn't sign the binaries, so you need to use `--no-quarantine` flag to avoid quarantine issues on macOS.
```bash
brew install --cask Warashi/tap/cage --no-quarantine
```

### With `go install`
```bash
go install github.com/Warashi/cage@latest
```

### From Source
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
- `-preset <name>`: Use a predefined preset configuration (can be used multiple times)
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
cage -config $HOME/my-presets.yaml -preset custom-preset ./script.sh

# Auto-presets in action (when configured)
cage claude help  # Automatically applies claude-code preset
cage npm install  # Automatically applies npm preset
cage yarn build   # Automatically applies npm preset (via regex pattern)
```

### Configuration File

Cage supports YAML configuration files to define presets. The configuration file is searched in the following order:

1. Path specified with `-config` flag
2. `$XDG_CONFIG_HOME/cage/presets.yaml` (or platform-specific config directory)
3. `$XDG_CONFIG_HOME/cage/presets.yml` (or platform-specific config directory)

The default config directory is:
- Linux: `$HOME/.config/cage/`
- macOS: `$HOME/Library/Application Support/cage/`
- Windows: `%APPDATA%\cage\`

Example configuration file:

```yaml
presets:
  npm:
    allow:
      - "."
      - "$HOME/.npm"
      - "$HOME/.cache/npm"
      - "$HOME/.npmrc"
  
  cargo:
    allow:
      - "."
      - "$HOME/.cargo"
      - "$HOME/.rustup"
      - "$HOME/.cache/sccache"
  
  git-enabled:
    allow:
      - "."
      - "$HOME/.ssh"
    allow-git: true
    allow-keychain: true  # macOS only
  
  custom:
    allow:
      - "./output"
      - "/tmp"  # Note: On macOS, use /private/tmp instead
      - "$HOME/.myapp"
```

Presets support the following options:
- `allow`: List of paths to grant write access (can be strings or objects with `eval-symlinks` option)
- `allow-git`: Enable access to git common directory (boolean)
- `allow-keychain`: Enable macOS keychain access (boolean)

#### Symlink Evaluation in Presets

The `allow` field in presets supports both simple string paths and objects with an `eval-symlinks` option. When `eval-symlinks` is set to `true`, the symlink will be resolved to its target path before granting access.

```yaml
presets:
  symlink-aware:
    allow:
      # Simple string (default: eval-symlinks = false)
      - "./direct-path"
      
      # Object with eval-symlinks option
      - path: "/tmp"
        eval-symlinks: true  # Resolves symlink to actual path
      
      # Mixed usage
      - "$HOME/.cache"
      - path: "$HOME/.local/share"
        eval-symlinks: true
```

This is particularly useful when:
- Working with tools that create symlinks to actual data directories
- macOS's `/tmp` is a symlink to `/private/tmp`
- Using symlinked configuration directories
- Working with package managers that use symlinks

Example use case for macOS:
```yaml
presets:
  macos-tmp:
    allow:
      - path: "/tmp"
        eval-symlinks: true  # Automatically resolves to /private/tmp
```

#### Auto-Presets

Cage can automatically apply presets based on the command being executed. This feature helps reduce typing and ensures consistent permissions for common tools.

Example auto-presets configuration:

```yaml
presets:
  claude-code:
    allow:
      - "$HOME/.config/claude"
      - "$HOME/tmp"
      - "/tmp"
    allow-keychain: true
    
  npm:
    allow:
      - "."
      - "$HOME/.npm"
      
auto-presets:
  # Exact command match
  - command: claude
    presets:
      - claude-code
      - tmp
      
  # Regex pattern match
  - command-pattern: ^(npm|npx|yarn)$
    presets:
      - npm
      
  # Multiple presets can be applied
  - command: git
    presets:
      - git-enabled
      - tmp
```

Auto-preset rules support:
- `command`: Exact command name match (basename of the command)
- `command-pattern`: Regular expression pattern to match command names
- `presets`: List of preset names to apply

**Note**: Auto-presets are merged with explicit `--preset` flags. Command-line presets are processed first, maintaining their priority over auto-presets.

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

## Environment Variables

### IN_CAGE

When a command is executed inside cage, the `IN_CAGE` environment variable is automatically set to `1`. This allows programs to detect if they are running within the cage sandbox.

```bash
# Check if running inside cage from a shell script
if [ "$IN_CAGE" = "1" ]; then
    echo "Running inside cage sandbox"
    # Adjust behavior accordingly
fi
```

This can be useful for:
- Adjusting application behavior when running in a restricted environment
- Providing warnings about limited functionality
- Debugging sandbox-related issues
- Conditional logging or telemetry

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
