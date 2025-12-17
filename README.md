# Cage

A cross-platform security sandbox CLI tool that restricts file system access for untrusted commands.

## Overview

Cage provides a unified way to run potentially untrusted commands or scripts with file system restrictions across Linux and macOS. It's designed for scenarios where you need to:
- Run AI coding assistants (Claude, Aider, Cursor) with filesystem protection
- Limit file system modifications during development
- Protect secrets (SSH keys, AWS credentials) from untrusted code
- Process sensitive data with controlled output locations

## Features

- **Write-only restriction** (default): Commands can read any file but cannot write unless explicitly allowed
- **Strict mode**: Restrict read access too—only allow explicit paths
- **Secrets protection**: Built-in presets to block access to SSH keys, cloud credentials, shell history
- **Cross-platform**: Works on Linux (kernel 5.13+) and macOS
- **Flexible permissions**: Grant write access via `--allow`, read access via `--allow-read` (strict mode)
- **Deny rules**: Block specific paths with `--deny`, `--deny-read`, `--deny-write`
- **Preset system**: Built-in and custom presets with inheritance
- **Auto-presets**: Automatically apply presets based on command name
- **Transparent execution**: Uses `syscall.Exec` to replace the process, preventing sandbox bypass

## Installation

### Pre-built Binaries with Homebrew Cask
We doesn't sign the binaries, so you need to use `--no-quarantine` flag to avoid quarantine issues on macOS.
```bash
brew install --cask Warashi/tap/cage --no-quarantine
```

When upgrading Cage, you may need to run:
```bash
brew upgrade cage --no-quarantine
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

#### Write Access
- `--allow <path>`: Grant write access to a specific path (can be used multiple times)
- `--allow-keychain`: Allow write access to the macOS keychain (macOS only)
- `--allow-git`: Allow access to git common directory (enables git operations in worktrees)
- `--allow-all`: Disable all restrictions (useful for debugging)

#### Strict Mode & Read Access
- `--strict`: Enable strict mode (don't allow `/` read access by default)
- `--allow-read <path>`: Grant read access to specific paths (only meaningful with `--strict`)

#### Deny Rules
- `--deny <path>`: Deny both read and write access (read deny only effective on macOS)
- `--deny-read <path>`: Deny read access (only effective on macOS)
- `--deny-write <path>`: Deny write access (both platforms)

#### Presets
- `--preset <name>`: Use a predefined preset configuration (can be used multiple times)
- `--no-defaults`: Skip default presets defined in config
- `--list-presets`: List available presets
- `--show-preset <name>`: Show the contents of a preset
- `-o <format>`: Output format for `--show-preset`: text (default) or yaml
- `--config <path>`: Path to custom configuration file

#### Utility
- `--dry-run`: Show the generated sandbox profile without executing
- `--version`: Print version information

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
# Use built-in presets
cage --preset builtin:npm -- npm install
cage --preset builtin:cargo -- cargo build

# Combine built-in presets for security
cage --preset builtin:strict-base --preset builtin:secrets-deny --allow . -- ./script.sh

# Use custom preset from config file
cage --preset my-custom-preset -- ./script.sh

# List all available presets (built-in and custom)
cage --list-presets

# View preset contents
cage --show-preset builtin:secrets-deny
cage --show-preset builtin:strict-base -o yaml

# Auto-presets in action (when configured)
cage claude help  # Automatically applies claude-code preset
cage npm install  # Automatically applies npm preset
```

#### Strict mode for secrets protection
```bash
# Strict mode restricts read access to explicit paths only
cage --strict --allow-read /usr --allow-read /etc --allow . -- make

# Use built-in safe-home preset (strict mode + safe directories)
cage --preset builtin:safe-home --allow . -- npm install
```

#### Deny rules (macOS full support, Linux write-only)
```bash
# Deny access to secrets
cage --deny "$HOME/.ssh" --deny "$HOME/.aws" --allow . -- python script.py

# Use built-in secrets-deny preset
cage --preset builtin:secrets-deny --allow . -- ./untrusted-script.sh
```

### Configuration File

Cage supports YAML configuration files to define presets. The configuration file is searched in the following order:

1. Path specified with `--config` flag
2. `$XDG_CONFIG_HOME/cage/presets.yaml`
3. `$HOME/.config/cage/presets.yaml`
4. `$HOME/.config/cage/presets.yml`

### Built-in Presets

Cage ships with these built-in presets (use with `--preset builtin:NAME`):

| Preset | Description |
|--------|-------------|
| `builtin:secure` | **Recommended.** Strict mode + system reads + secrets deny + CWD write + git enabled |
| `builtin:strict-base` | Minimal system read access with strict mode enabled |
| `builtin:secrets-deny` | Blocks SSH keys, AWS/Azure/GCloud creds, GPG, shell history, browser data |
| `builtin:safe-home` | Strict mode + safe home directories (Documents, Downloads, Projects, etc.) |
| `builtin:home-dotfiles-deny` | Deny all dotfiles in home (macOS only - globs don't work on Linux) |
| `builtin:npm` | Write access for Node.js development (., ~/.npm, ~/.cache/npm) |
| `builtin:cargo` | Write access for Rust development (., ~/.cargo, ~/.rustup) |

Example configuration file:

```yaml
# Default presets applied to ALL commands
defaults:
  presets:
    - "builtin:secure"

presets:
  # Extend secure preset with keychain access for AI tools
  ai-coder:
    extends:
      - "builtin:secure"
    allow:
      - path: "/tmp"
        eval-symlinks: true  # Resolves /tmp -> /private/tmp on macOS
      - "$HOME/.config/claude"
    allow-keychain: true
    allow-git: true
  
  # Simple preset
  npm:
    allow:
      - "."
      - "$HOME/.npm"
      - "$HOME/.cache/npm"
  
  # Preset that skips defaults
  unrestricted:
    skip-defaults: true
    allow:
      - "."

auto-presets:
  # Exact command match
  - command: claude
    presets:
      - ai-coder
  
  # Regex pattern match
  - command-pattern: ^(npm|npx|yarn|pnpm)$
    presets:
      - npm
```

Presets support the following options:
- `extends`: List of presets to inherit from (including `builtin:*` presets)
- `skip-defaults`: Skip default presets when this preset is used (boolean)
- `strict`: Enable strict mode (don't allow `/` read by default)
- `allow`: List of paths to grant write access
- `read`: List of read-only paths (only used when `strict: true`)
- `deny`: List of paths to deny read+write (read deny only effective on macOS)
- `deny-read`: List of paths to deny read (macOS only)
- `deny-write`: List of paths to deny write (both platforms)
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
- **Allowlist-only**: Cannot deny subpaths under allowed parents
- **No glob patterns**: Paths must be literal
- Read denies only warn—use strict mode for read protection
- Restrictions inherit to all child processes (kernel-enforced)

### macOS
- Uses `sandbox-exec` with custom sandbox profiles
- Full allowlist AND denylist support
- Supports glob patterns via regex
- All deny rules fully enforced
- Restrictions inherit to all child processes (kernel-enforced)

### Other Platforms
- Returns an error indicating sandboxing is not implemented

## Security Policy

Cage enforces the following security policy:

| Operation | Default | With `--strict` | With `--allow` |
|-----------|---------|-----------------|----------------|
| File Read | ✅ Allowed | ❌ Denied (need `--allow-read`) | ✅ Allowed |
| File Write | ❌ Denied | ❌ Denied | ✅ Allowed for path |
| File Execute | ✅ Allowed | ✅ Allowed (if readable) | ✅ Allowed |
| Network Access | ✅ Allowed | ✅ Allowed | ✅ Allowed |
| Process Creation | ✅ Allowed | ✅ Allowed | ✅ Allowed |

### Linux Limitation: Protecting Secrets

On Linux, deny rules for **reads** cannot be enforced due to Landlock's allowlist-only model. The **only** way to protect secrets on Linux is to use strict mode:

```yaml
presets:
  linux-secure:
    strict: true          # Don't allow / read
    read:                 # Explicitly list what CAN be read
      - "/usr"
      - "/lib"
      - "/etc"
      - "$HOME/Documents"
      # .ssh, .aws NOT listed = NOT readable
    allow:
      - "."
```

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

### Homebrew Integration

When using Homebrew and cage together on macOS, you may encounter an issue with the standard Homebrew configuration. The typical Homebrew setup includes the following line in `.zprofile`:

```bash
eval "$(/opt/homebrew/bin/brew shellenv)"
```

However, when cage executes commands (such as when used with Claude Code), this configuration can cause the following error:

```
/opt/homebrew/Library/Homebrew/cmd/shellenv.sh: line 18: /bin/ps: Operation not permitted
```

This occurs because the `shellenv` script attempts to execute `/bin/ps`, which is blocked by the sandbox restrictions.
To resolve this issue, modify your `.zprofile` to conditionally evaluate `shellenv` only when not running inside cage:

```bash
if [[ -z $IN_CAGE ]]; then
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi
```

This configuration:
- Allows normal Homebrew functionality when using your shell directly
- Prevents the error when commands are executed within cage
- Works because cage inherits the necessary environment variables from the parent shell, making the `shellenv` evaluation unnecessary

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

The recommended way to run AI coding assistants:

```yaml
# In ~/.config/cage/presets.yaml
presets:
  ai-coder:
    extends:
      - "builtin:strict-base"
      - "builtin:secrets-deny"
    allow:
      - "."
      - path: "/tmp"
        eval-symlinks: true
    allow-keychain: true
    allow-git: true

auto-presets:
  - command-pattern: ^(claude|aider|cursor|opencode|windsurf)$
    presets:
      - ai-coder
```

Then simply run:
```bash
cage claude --dangerously-skip-permissions
cage aider
```

Or with shell aliases in `~/.bashrc` or `~/.zshrc`:
```bash
alias claude='cage claude'
alias aider='cage aider'
```

## Limitations

- Sandboxing is only implemented for Linux and macOS
- Linux requires kernel 5.13 or later for Landlock support
- **Linux**: Cannot deny read access under allowed parents (use strict mode instead)
- **Linux**: Glob patterns not supported (enumerate paths explicitly)
- Network and process execution are not restricted

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.

## Related Documentation

- [Quickstart Guide](docs/QUICKSTART.md) - Get started in 5 minutes
- [Developer Guide](docs/DEVELOPER_GUIDE.md) - Complete configuration reference
- [CLI Design Document](CLI_DESIGN.md) - Detailed design and implementation notes
