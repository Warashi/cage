# Cage Developer Guide

Complete reference for configuring and using Cage effectively.

## Table of Contents

- [Configuration File](#configuration-file)
- [Default Presets](#default-presets)
- [CLI Reference](#cli-reference)
- [Built-in Presets](#built-in-presets)
- [Preset Configuration](#preset-configuration)
- [Auto-Presets](#auto-presets)
- [Shell Aliases](#shell-aliases)
- [Platform Differences](#platform-differences)
- [Security Best Practices](#security-best-practices)
- [Ready-to-Use Configurations](#ready-to-use-configurations)
- [Environment Variables](#environment-variables)
- [Troubleshooting](#troubleshooting)

---

## Configuration File

Cage looks for configuration in this order:

1. Path specified with `--config` flag
2. `$XDG_CONFIG_HOME/cage/presets.yaml`
3. `$HOME/.config/cage/presets.yaml`
4. `$HOME/.config/cage/presets.yml`

### Basic Structure

```yaml
# Default presets applied to ALL commands
defaults:
  presets:
    - "builtin:secrets-deny"

presets:
  my-preset:
    # Inherit from other presets
    extends:
      - "builtin:strict-base"
      - "other-preset"
    
    # Skip default presets when this preset is used
    skip-defaults: false
    
    # Enable strict mode (don't allow / read)
    strict: true
    
    # Write paths (allow writing)
    allow:
      - "."
      - "$HOME/.npm"
      - path: "/tmp"
        eval-symlinks: true  # Resolve symlinks before granting access
    
    # Read-only paths (only used when strict: true)
    read:
      - "/usr"
      - "$HOME/Documents"
    
    # Deny read+write (macOS: fully enforced, Linux: write only)
    deny:
      - "$HOME/.ssh"
    
    # Deny read only (macOS only)
    deny-read:
      - "$HOME/.bash_history"
    
    # Deny write only (both platforms)
    deny-write:
      - "$HOME/.npmrc"
    
    # macOS keychain access
    allow-keychain: true
    
    # Git common directory access (for worktrees)
    allow-git: true

auto-presets:
  - command: claude
    presets:
      - ai-coder
  
  - command-pattern: ^(npm|npx|yarn|pnpm)$
    presets:
      - npm
```

---

## Default Presets

You can configure presets that apply to **every** cage invocation:

```yaml
defaults:
  presets:
    - "builtin:secrets-deny"  # Always protect secrets
```

### Skipping Defaults

**Option 1: CLI flag**
```bash
cage --no-defaults --allow . -- some-command
```

**Option 2: Preset option**
```yaml
presets:
  unrestricted:
    skip-defaults: true  # This preset skips default presets
    allow:
      - "."
```

When any preset in the chain has `skip-defaults: true`, defaults are skipped.

### Precedence Order

1. Default presets (from `defaults.presets`)
2. Command-line `--preset` flags
3. Auto-presets (from `auto-presets` rules)

All settings are merged (union for paths, OR for booleans).

---

## CLI Reference

### Write Access

```bash
--allow PATH              # Grant write access to PATH (repeatable)
```

### Read Access (Strict Mode)

```bash
--strict                  # Enable strict mode: don't allow / read by default
--allow-read PATH         # Grant read access to PATH (only with --strict)
```

### Deny Rules

```bash
--deny PATH               # Deny both read AND write (read deny macOS only)
--deny-read PATH          # Deny read access (macOS only)
--deny-write PATH         # Deny write access (both platforms)
```

### Special Access

```bash
--allow-keychain          # Allow macOS keychain access
--allow-git               # Allow access to git common directory
--allow-all               # Disable ALL restrictions (debugging only)
```

### Presets

```bash
--preset NAME             # Use a preset (repeatable)
--no-defaults             # Skip default presets from config
--list-presets            # List all available presets
--show-preset NAME        # Show preset contents
-o FORMAT                 # Output format: text (default) or yaml
--config PATH             # Path to custom config file
```

### Utility

```bash
--dry-run                 # Show sandbox profile without executing
--version                 # Print version
```

### Examples

```bash
# Basic sandboxing
cage --allow . -- npm install

# Strict mode with explicit reads
cage --strict --allow-read /usr --allow-read /etc --allow . -- make

# Protect secrets
cage --deny "$HOME/.ssh" --deny "$HOME/.aws" --allow . -- python script.py

# Combine multiple presets
cage --preset builtin:strict-base --preset builtin:secrets-deny --allow . -- ./build.sh

# Dry-run to inspect profile
cage --dry-run --preset ai-coder -- claude

# Show preset as YAML (for copying to config)
cage --show-preset builtin:secrets-deny -o yaml
```

---

## Built-in Presets

Use with `--preset builtin:NAME` or extend in your config.

### builtin:secure (Recommended)

**The recommended default preset.** Inherits from `strict-base` and `secrets-deny`, adds write access to current directory, XDG directories, AI coding tools, and IDE configs:

```yaml
extends:
  - "builtin:strict-base"
  - "builtin:secrets-deny"
allow:
  - "."
  - "$HOME/.local/share"
  - "$HOME/.local/state"
  # AI coding tools
  - "$HOME/.bun"
  - "$HOME/.cache/opencode"
  - "$HOME/.claude"
  - "$HOME/.codeium"
  - "$HOME/.cody"
  - "$HOME/.config/aider"
  - "$HOME/.config/claude"
  - "$HOME/.config/opencode"
  - "$HOME/.continue"
  - "$HOME/.cursor"
  - "$HOME/.tabby"
  # IDE/editor config
  - "$HOME/.config/Code"
  - "$HOME/.config/Cursor"
  - "$HOME/.config/JetBrains"
  - "$HOME/.config/VSCodium"
  - "$HOME/.idea"
  - "$HOME/.vscode"
  - "$HOME/.vscode-server"
  # Shell tools
  - "$HOME/.cache/starship"
allow-git: true
```

This preset is designed for everyday development work:
- **Strict mode**: Only explicitly allowed paths are readable (essential for Linux secrets protection)
- **System paths**: Can read OS binaries, libraries, and config (from `strict-base`)
- **Write to CWD**: Can modify files in current directory
- **Write to XDG directories**: `~/.local/share` and `~/.local/state` for app data/state
- **AI tool configs**: OpenCode, Claude, Aider, Continue, Codeium, Cody, Tabby, Cursor
- **IDE configs**: VS Code, VSCodium, Cursor, JetBrains IDEs
- **Secrets protected**: SSH keys, cloud credentials, browser data blocked (from `secrets-deny`)
- **Git enabled**: Can perform git operations in worktrees

### builtin:strict-base

Minimal system read access with strict mode enabled:

```yaml
strict: true
read:
  - "/Applications"
  - "/Library"
  - "/System"
  - "/bin"
  - "/dev"
  - "/etc"
  - "/lib"
  - "/lib64"
  - "/opt"
  - "/private/var"
  - "/private/var/folders"
  - "/proc"
  - "/sbin"
  - "/sys"
  - "/usr"
  - "/var"
  - "$HOME/.config/fish"
```

### builtin:secrets-deny

Comprehensive protection for sensitive files and credentials:

```yaml
deny:
  # Browser data (sessions, cookies, saved passwords)
  - "$HOME/.config/BraveSoftware"
  - "$HOME/.config/chromium"
  - "$HOME/.config/google-chrome"
  - "$HOME/.mozilla/firefox"

  # CI/CD and deployment platforms
  - "$HOME/.config/circleci"        # CircleCI tokens
  - "$HOME/.config/heroku"          # Heroku API keys
  - "$HOME/.config/netlify"         # Netlify access tokens
  - "$HOME/.config/railway"         # Railway deployment tokens
  - "$HOME/.config/vercel"          # Vercel deployment tokens

  # Cloud provider credentials
  - "$HOME/.aws"                    # AWS CLI credentials
  - "$HOME/.azure"                  # Azure CLI credentials
  - "$HOME/.config/doctl"           # DigitalOcean CLI
  - "$HOME/.config/flyctl"          # Fly.io CLI
  - "$HOME/.config/gcloud"          # Google Cloud SDK
  - "$HOME/.config/hcloud"          # Hetzner Cloud CLI
  - "$HOME/.config/linode"          # Linode CLI
  - "$HOME/.config/scaleway"        # Scaleway CLI

  # Container and orchestration
  - "$HOME/.config/containers"       # Podman registry auth
  - "$HOME/.config/k3d"              # k3d kubeconfigs
  - "$HOME/.config/Lens"             # Lens Kubernetes IDE
  - "$HOME/.config/OpenLens"         # OpenLens Kubernetes IDE
  - "$HOME/.docker/config.json"      # Docker registry credentials
  - "$HOME/.helm"                    # Helm repository credentials
  - "$HOME/.kube"                    # Kubernetes config & tokens
  - "$HOME/.lima/_config"            # Lima VM SSH keys
  - "$HOME/.rd"                      # Rancher Desktop

  # Git forges and CLI tools
  - "$HOME/.config/gh"              # GitHub CLI (hosts.yml contains OAuth tokens)
  - "$HOME/.config/glab"            # GitLab CLI tokens
  - "$HOME/.config/hub"             # Hub CLI OAuth tokens
  - "$HOME/.git-credentials"        # Git credential storage
  - "$HOME/.netrc"                  # FTP/HTTP credentials (used by curl, etc.)

  # macOS sensitive data (Keychain, Mail, Messages, Safari, etc.)
  - "$HOME/Library"

  # Package manager credentials
  - "$HOME/.cargo/credentials.toml" # Cargo/crates.io tokens
  - "$HOME/.config/configstore"     # npm/yarn token storage (used by many Node tools)
  - "$HOME/.config/pip"             # pip config (can contain tokens)
  - "$HOME/.npmrc"                  # npm auth tokens
  - "$HOME/.pypirc"                 # PyPI upload credentials

  # Security and encryption
  - "$HOME/.config/op"              # 1Password CLI session data
  - "$HOME/.config/sops/age"        # SOPS age encryption keys
  - "$HOME/.gnupg"                  # GPG keys and config

  # Security scanning and dev tools
  - "$HOME/.config/ngrok"           # ngrok auth tokens
  - "$HOME/.config/snyk"            # Snyk API tokens

  # Shell history (contains commands, may expose secrets in env vars)
  - "$HOME/.bash_history"
  - "$HOME/.local/share/atuin"                # Atuin shell history
  - "$HOME/.local/share/fish/fish_history"
  - "$HOME/.mysql_history"
  - "$HOME/.node_repl_history"
  - "$HOME/.psql_history"
  - "$HOME/.python_history"
  - "$HOME/.rediscli_history"
  - "$HOME/.zsh_history"

  # SSH keys and config
  - "$HOME/.ssh"
```

### builtin:home-dotfiles-deny

Deny all dotfiles in home directory:

```yaml
deny:
  - "$HOME/.*"    # Glob pattern - only effective on macOS!
```

> **Warning**: On Linux, this glob pattern cannot be enforced for reads. Use `builtin:safe-home` instead.

### builtin:safe-home

Strict mode with safe home directories:

```yaml
strict: true
read:
  # System paths (same as strict-base)
  - "/usr"
  - "/bin"
  - "/sbin"
  - "/lib"
  - "/lib64"
  - "/etc"
  - "/opt"
  - "/var"
  - "/dev"
  - "/proc"
  - "/sys"
  - "/System"
  - "/Library"
  - "/Applications"
  - "/private/var/folders"
  
  # Safe home directories
  - "$HOME/Documents"
  - "$HOME/Downloads"
  - "$HOME/Desktop"
  - "$HOME/Pictures"
  - "$HOME/Music"
  - "$HOME/Videos"
  - "$HOME/Movies"
  - "$HOME/Projects"
  - "$HOME/Developer"
  - "$HOME/Code"
  - "$HOME/src"
  - "$HOME/go/src"
  - "$HOME/workspace"
```

### builtin:npm

Node.js development paths:

```yaml
allow:
  - "."
  - "$HOME/.npm"
  - "$HOME/.cache/npm"
  - "node_modules"
```

### builtin:cargo

Rust development paths:

```yaml
allow:
  - "."
  - "$HOME/.cargo"
  - "$HOME/.rustup"
  - "target"
```

---

## Preset Configuration

### Inheritance with extends

Presets can inherit from others:

```yaml
presets:
  secure-npm:
    extends:
      - "builtin:strict-base"
      - "builtin:secrets-deny"
      - "builtin:npm"
    allow:
      - "$HOME/.config/npm"
```

**Merge semantics:**
- `allow`, `read`, `deny`, `deny-read`, `deny-write`: **union** (combined)
- `strict`, `allow-keychain`, `allow-git`: **OR** (true if any is true)

### Path Options

```yaml
allow:
  # Simple string path
  - "."
  - "$HOME/.npm"
  
  # Object with options
  - path: "/tmp"
    eval-symlinks: true  # Resolve symlinks (useful for macOS /tmp -> /private/tmp)
```

### Environment Variable Expansion

All paths support `$VAR` and `${VAR}` expansion:

```yaml
allow:
  - "$HOME/.config/myapp"
  - "${XDG_CACHE_HOME}/myapp"
  - "."  # Current working directory
```

---

## Auto-Presets

Automatically apply presets based on the command being run:

```yaml
auto-presets:
  # Exact command match
  - command: claude
    presets:
      - ai-coder
  
  # Regex pattern match
  - command-pattern: ^(npm|npx|yarn|pnpm)$
    presets:
      - npm
  
  # Multiple presets
  - command: git
    presets:
      - git-ops
      - secrets-deny
```

**How it works:**
1. Cage extracts the base command name (e.g., `/usr/bin/npm` → `npm`)
2. Matches against `command` (exact) or `command-pattern` (regex)
3. Applies matched presets **after** command-line `--preset` flags

---

## Shell Aliases

### Bash/Zsh

Add to `~/.bashrc` or `~/.zshrc`:

```bash
# AI Coding Assistants
alias claude='cage claude'
alias aider='cage aider'
alias cursor='cage cursor'
alias opencode='cage opencode'
alias windsurf='cage windsurf'
alias codex='cage codex'

# With explicit preset (if not using auto-presets)
alias claude='cage --preset ai-coder -- claude'

# Development tools
alias npm='cage --preset builtin:npm -- npm'
alias npx='cage --preset builtin:npm -- npx'
alias yarn='cage --preset builtin:npm -- yarn'
alias pnpm='cage --preset builtin:npm -- pnpm'
alias cargo='cage --preset builtin:cargo -- cargo'

# Sandboxed shell for untrusted operations
alias sandbox='cage --preset builtin:strict-base --allow . -- bash'
```

### Fish

Add to `~/.config/fish/config.fish`:

```fish
alias claude 'cage claude'
alias aider 'cage aider'
alias npm 'cage --preset builtin:npm -- npm'
```

---

## Platform Differences

### macOS (sandbox-exec)

**Capabilities:**
- Full allowlist AND denylist support
- Glob patterns work via regex conversion
- All deny rules enforced

**Example - this works on macOS:**
```yaml
presets:
  selective-home:
    allow:
      - "$HOME"           # Allow all of home
    deny:
      - "$HOME/.ssh"      # Except SSH keys
      - "$HOME/.*"        # And all dotfiles (glob!)
```

### Linux (Landlock LSM)

**Capabilities:**
- Allowlist-only model
- Kernel 5.13+ required
- Restrictions inherit to all child processes

**Limitations:**
- **Cannot deny subpaths** under an allowed parent
- **No glob pattern support**
- Read denies only warn (cannot be enforced)

**Example - this does NOT protect secrets on Linux:**
```yaml
# WRONG for Linux! The deny rules cannot be enforced
presets:
  broken-on-linux:
    allow:
      - "$HOME"           # Allows reading EVERYTHING in home
    deny:
      - "$HOME/.ssh"      # WARNING: Cannot be enforced!
```

### The ONLY Way to Protect Secrets on Linux

Use **strict mode** with explicit path enumeration:

```yaml
presets:
  linux-secure:
    strict: true          # DON'T allow / read
    read:
      # System paths
      - "/usr"
      - "/lib"
      - "/etc"
      - "/bin"
      # EXPLICITLY list each home subdirectory needed:
      - "$HOME/Documents"
      - "$HOME/Projects"
      - "$HOME/.config/myapp"
      # .ssh, .aws, etc. are NOT listed = NOT readable
    allow:
      - "."
```

---

## Security Best Practices

### 1. Always Use Strict Mode for Sensitive Work

```yaml
presets:
  paranoid:
    extends:
      - "builtin:safe-home"
      - "builtin:secrets-deny"
    allow:
      - "."
```

### 2. Minimize Write Access

```bash
# Bad: allows writing anywhere
cage --allow-all -- ./script.sh

# Good: only current directory
cage --allow . -- ./script.sh

# Better: specific output directory
cage --allow ./output -- ./script.sh
```

### 3. Use Deny Rules for Defense in Depth (macOS)

Even with strict mode, add deny rules as a safety net:

```yaml
presets:
  defense-in-depth:
    extends:
      - "builtin:strict-base"
      - "builtin:secrets-deny"  # Extra protection
```

### 4. Audit with Dry-Run

Always check what a preset does before using it:

```bash
cage --dry-run --preset my-preset -- command
```

### 5. Protect Shell Configuration

Prevent malicious code from modifying your shell:

```yaml
deny-write:
  - "$HOME/.bashrc"
  - "$HOME/.zshrc"
  - "$HOME/.profile"
  - "$HOME/.bash_profile"
  - "$HOME/.zprofile"
```

---

## Ready-to-Use Configurations

### AI Coding Assistants

```yaml
presets:
  ai-coder:
    extends:
      - "builtin:strict-base"
      - "builtin:secrets-deny"
    allow:
      - "."
      - path: "/tmp"
        eval-symlinks: true
      - "$HOME/.config/claude"
      - "$HOME/.aider"
    allow-keychain: true
    allow-git: true

auto-presets:
  - command-pattern: ^(claude|aider|cursor|opencode|windsurf|codex|cody|continue|copilot)$
    presets:
      - ai-coder
```

### Node.js Development

```yaml
presets:
  node-dev:
    extends:
      - "builtin:npm"
    allow:
      - "$HOME/.cache/yarn"
      - "$HOME/.cache/pnpm"
      - "$HOME/.local/share/pnpm"
    deny-write:
      - "$HOME/.npmrc"  # Protect npm token

auto-presets:
  - command-pattern: ^(npm|npx|yarn|pnpm|node|tsx|ts-node)$
    presets:
      - node-dev
```

### Python Development

```yaml
presets:
  python-dev:
    allow:
      - "."
      - "$HOME/.cache/pip"
      - "$HOME/.local/lib/python*"
      - "$HOME/.virtualenvs"
      - ".venv"
      - "venv"
    deny-write:
      - "$HOME/.pypirc"  # Protect PyPI token

auto-presets:
  - command-pattern: ^(python|python3|pip|pip3|poetry|pdm|uv)$
    presets:
      - python-dev
```

### Rust Development

```yaml
presets:
  rust-dev:
    extends:
      - "builtin:cargo"
    allow:
      - "$HOME/.cache/sccache"

auto-presets:
  - command-pattern: ^(cargo|rustc|rustup)$
    presets:
      - rust-dev
```

### Full Developer Setup

Complete config for a typical development environment:

```yaml
presets:
  base-dev:
    extends:
      - "builtin:strict-base"
      - "builtin:secrets-deny"
    allow:
      - "."
      - path: "/tmp"
        eval-symlinks: true
    allow-git: true

  ai-coder:
    extends:
      - base-dev
    allow:
      - "$HOME/.config/claude"
    allow-keychain: true

  node-dev:
    extends:
      - base-dev
      - "builtin:npm"
    allow:
      - "$HOME/.cache/yarn"

  python-dev:
    extends:
      - base-dev
    allow:
      - "$HOME/.cache/pip"
      - ".venv"

  rust-dev:
    extends:
      - base-dev
      - "builtin:cargo"

auto-presets:
  - command-pattern: ^(claude|aider|cursor|opencode)$
    presets: [ai-coder]
  
  - command-pattern: ^(npm|npx|yarn|pnpm)$
    presets: [node-dev]
  
  - command-pattern: ^(python|pip|poetry)$
    presets: [python-dev]
  
  - command-pattern: ^(cargo|rustc)$
    presets: [rust-dev]
```

---

## Environment Variables

### IN_CAGE

Set to `"1"` when running inside cage:

```bash
# In your scripts
if [ "$IN_CAGE" = "1" ]; then
    echo "Running in sandbox"
fi
```

Useful for:
- Adjusting behavior in restricted environments
- Conditional logging
- Skipping operations that won't work sandboxed

### Homebrew Integration

Standard Homebrew setup can conflict with cage. Add to `~/.zprofile`:

```bash
if [[ -z $IN_CAGE ]]; then
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi
```

This prevents the `/bin/ps: Operation not permitted` error.

---

## Troubleshooting

### "Operation not permitted" Errors

1. Check if you're missing an `--allow` path
2. Run with `--dry-run` to see the profile
3. Try `--allow-all` temporarily to confirm it's a sandbox issue

### Deny Rules Not Working (Linux)

Read denies **cannot** be enforced on Linux due to Landlock limitations:

```
cage: warning: read deny "$HOME/.ssh" cannot be enforced on Linux
(Landlock is allowlist-only); use --strict for read protection
```

**Solution:** Use strict mode with explicit read paths.

### Glob Patterns Not Working (Linux)

```
cage: warning: glob pattern "$HOME/.*" cannot be enforced on Linux
(Landlock requires literal paths); pattern will be ignored
```

**Solution:** Enumerate paths explicitly instead of using globs.

### symlink Issues

If a path isn't being allowed correctly:

```yaml
allow:
  - path: "/tmp"
    eval-symlinks: true  # Resolves /tmp -> /private/tmp on macOS
```

### Finding the Right Paths

```bash
# Watch what files a command accesses
# macOS
sudo fs_usage -f filesys <command>

# Linux
strace -f -e trace=file <command>
```
