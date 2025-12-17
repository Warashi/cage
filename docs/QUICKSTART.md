# Cage Quickstart Guide

Get sandboxed in 5 minutes.

## Installation

### macOS (Homebrew)

```bash
brew install --cask Warashi/tap/cage --no-quarantine
```

### Go Install

```bash
go install github.com/Warashi/cage@latest
```

### From Source

```bash
git clone https://github.com/Warashi/cage
cd cage
go build
go install
```

## Your First Sandboxed Command

```bash
# Run a command with the secure preset (recommended)
cage --preset builtin:secure -- ls -la

# See what the sandbox profile looks like (dry-run)
cage --dry-run --preset builtin:secure -- npm install
```

## Quick Setup for AI Coding Tools

Create your config file:

```bash
mkdir -p ~/.config/cage
cat > ~/.config/cage/presets.yaml << 'EOF'
# Apply secure defaults to ALL commands
# builtin:secure provides:
#   - strict mode (read restrictions)
#   - system paths readable
#   - secrets protected (SSH, AWS, browser data, etc.)
#   - write access to current directory
#   - git operations enabled
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
        eval-symlinks: true
    allow-keychain: true

auto-presets:
  - command-pattern: ^(claude|aider|cursor|opencode|windsurf|codex)$
    presets:
      - ai-coder
EOF
```

Now run your AI coding assistant:

```bash
# With auto-presets, cage automatically applies the right preset
cage claude --dangerously-skip-permissions

# Or explicitly specify the preset
cage --preset ai-coder -- aider

# Skip defaults for a specific command
cage --no-defaults --allow . -- some-trusted-command
```

## Built-in Presets

Cage ships with ready-to-use presets:

| Preset | Description |
|--------|-------------|
| `builtin:secure` | **Recommended default.** Strict mode + system reads + secrets deny + CWD write + git enabled |
| `builtin:strict-base` | Minimal system read access, strict mode enabled (no write paths) |
| `builtin:secrets-deny` | Blocks SSH keys, AWS creds, browser data, shell history (denylist only) |
| `builtin:safe-home` | Strict mode + safe home directories (Documents, Downloads, Projects) |
| `builtin:npm` | Write access for Node.js development |
| `builtin:cargo` | Write access for Rust development |

```bash
# List all available presets
cage --list-presets

# View a preset's contents
cage --show-preset builtin:secure

# Use the secure preset directly
cage --preset builtin:secure -- npm install
```

## Shell Aliases

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
# AI coding assistants (secure by default)
alias aider='cage aider'
alias claude='cage claude'
alias cursor='cage cursor'
alias opencode='cage opencode'

# Development tools
alias npm='cage --preset builtin:npm -- npm'
alias cargo='cage --preset builtin:cargo -- cargo'
```

## Verify It's Working

```bash
# Check if running inside cage
echo $IN_CAGE  # Returns "1" when sandboxed

# Test that secrets are protected (should fail)
cage --preset builtin:secure -- cat ~/.ssh/id_rsa

# Dry-run to see the full sandbox profile
cage --dry-run --preset builtin:secure -- claude
```

## Homebrew Fix

If you see errors about `/bin/ps` when using Homebrew commands inside cage, add this to your `~/.zprofile`:

```bash
if [[ -z $IN_CAGE ]]; then
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi
```

## Next Steps

- Read the [Developer Guide](DEVELOPER_GUIDE.md) for complete configuration reference
- Explore preset inheritance with `extends`
- Set up auto-presets for all your common tools
- Understand [platform differences](DEVELOPER_GUIDE.md#platform-differences) between macOS and Linux

## Key Points

1. **Recommended default**: Use `builtin:secure` for most use cases
2. **Config location**: `~/.config/cage/presets.yaml`
3. **Default presets**: Use `defaults.presets` to apply presets to ALL commands
4. **Skip defaults**: Use `--no-defaults` flag or `skip-defaults: true` in a preset
5. **Strict mode**: Restricts reads - essential for secrets protection on Linux
6. **Linux limitation**: Read denies only work with strict mode (Landlock is allowlist-only)
