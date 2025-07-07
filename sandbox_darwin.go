//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// runInSandbox implements sandbox execution for macOS using sandbox-exec
func runInSandbox(config *SandboxConfig) error {
	// Generate sandbox profile
	profile, err := generateSandboxProfile(config)
	if err != nil {
		return fmt.Errorf("generate sandbox profile: %w", err)
	}

	// Find sandbox-exec executable
	sandboxPath, err := exec.LookPath("sandbox-exec")
	if err != nil {
		return fmt.Errorf("sandbox-exec not found: %w", err)
	}

	// Prepare sandbox-exec command
	args := []string{"sandbox-exec", "-p", profile, config.Command}
	args = append(args, config.Args...)

	// Replace current process with sandbox-exec
	return syscall.Exec(sandboxPath, args, os.Environ())
}

// generateSandboxProfile creates a sandbox-exec profile with write restrictions
func generateSandboxProfile(config *SandboxConfig) (string, error) {
	var profile bytes.Buffer

	// Write profile header
	profile.WriteString("(version 1)\n")
	profile.WriteString(`(import "system.sb")` + "\n")
	profile.WriteString("(allow default)\n")

	if config.AllowAll {
		return profile.String(), nil
	}

	// Deny writes to all paths except allowed ones
	profile.WriteString("(deny file-write*)\n")
	// allow allow for /private/var/folders
	profile.WriteString(
		`(allow file-write* (regex #"^/private/var/folders/[^/]+/[^/]+/(C|T|0)($|/)"))` + "\n",
	)

	// If allow-keychain is set, allow access to the keychain
	if config.AllowKeychain {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		// allow for keychain
		fmt.Fprintf(&profile, `(allow file-write* (subpath "%s/Library/Keychains"))`+"\n", homeDir)
	}

	// Allow writes to specified paths
	for _, path := range config.AllowedPaths {
		// Expand path to absolute
		absPath, err := filepath.Abs(path)
		if err != nil {
			// If we can't resolve the path, use it as-is
			absPath = path
		}

		// Escape the path for the sandbox profile
		escapedPath := escapePathForSandbox(absPath)

		// Allow writes to the path and all subpaths
		fmt.Fprintf(&profile, "(allow file-write* (subpath \"%s\"))\n", escapedPath)

		// Also allow writes to the literal path (for directory creation)
		fmt.Fprintf(&profile, "(allow file-write* (literal \"%s\"))\n", escapedPath)
	}

	return profile.String(), nil
}

// escapePathForSandbox escapes special characters in paths for sandbox profiles
func escapePathForSandbox(path string) string {
	// Escape backslashes and double quotes
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, "\"", "\\\"")
	return path
}
