# Cage - Cross-platform Sandboxing CLI Tool

## Overview
Cage is a security-focused command-line tool that creates sandboxed environments for running untrusted or potentially dangerous commands. It provides unified sandboxing capabilities across Linux (using go-landlock) and macOS/Darwin (using sandbox-exec), ensuring consistent security policies across platforms.

The primary goal of Cage is to prevent unintended file system modifications by restricting write access to only explicitly allowed paths, while maintaining full read access and network capabilities.

## Command Structure

```bash
# Basic syntax
cage [flags] <command> [command-args...]

# Use double-dash to separate cage flags from command flags
cage [flags] -- <command> [command-flags] [command-args...]
```

## Core Options

### Security Flags

```bash
--allow-all        # Disable all restrictions (use for testing/debugging only)
```

### File System Permissions

```bash
--allow <path>     # Grant write access to specific paths (can be used multiple times)
                   # Example: --allow /tmp --allow ~/output
```

## Usage Examples

### Basic Usage

```bash
# Run a command with default restrictions (no write access)
cage ls -la

# Allow writing to temporary directory
cage --allow /tmp python script.py

# Allow writing to multiple directories
cage --allow /tmp --allow ~/output -- npm run build

# Run in investigation mode (no restrictions)
cage --allow-all make install
```

### Real-world Scenarios

```bash
# Test an untrusted script safely
cage python untrusted_script.py

# Build a project with limited write access
cage --allow ./build --allow ./dist -- cargo build --release

# Run a web scraper with output restrictions
cage --allow ./data -- python scraper.py --output ./data/results.json
```

## Important Notes

### Default Security Policy
- **Read Access**: Unrestricted - commands can read any file the user has access to
- **Write Access**: Denied by default - must be explicitly granted using `--allow` flags
- **Network Access**: Allowed - commands can make network connections
- **Process Execution**: Allowed - commands can spawn child processes

### Best Practices
- Always run untrusted code with minimal permissions
- Grant write access only to specific directories needed for output
- Use `--allow-all` only for debugging or when you fully trust the command
- Review the command's behavior in restricted mode before granting additional permissions

### Platform-Specific Implementation

**Linux (using go-landlock):**
- Requires kernel version 5.13 or later
- Uses the Landlock LSM (Linux Security Module) for fine-grained access control
- Provides robust, kernel-level security guarantees

**macOS (using sandbox-exec):**
- Compatible with all modern macOS versions
- Uses Apple's sandbox-exec mechanism (note: officially deprecated but still functional)
- Provides application-level sandboxing through system profiles

## Common Usage Patterns

### Development Workflows

```bash
# Running tests with restricted file access
cage --allow ./test-output -- pytest -v

# Building projects with specific output directories
cage --allow ./build --allow ./node_modules -- npm install
cage --allow ./dist -- npm run build
```

### Security Testing

```bash
# Analyze potentially malicious scripts
cage python suspicious_script.py

# Test third-party tools with minimal permissions
cage --allow ~/safe-output -- third-party-tool process data.json
```

### Data Processing

```bash
# Process sensitive data with output restrictions
cage --allow ./processed-data -- python etl_script.py --input ./raw-data

# Run data transformations with temporary file access
cage --allow /tmp --allow ./output -- ./transform.sh input.csv
```
