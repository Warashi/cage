//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runInSandbox implements sandbox execution for macOS using sandbox-exec
func runInSandbox(config *SandboxConfig) error {
	// If allow-all is set, run without restrictions
	if config.AllowAll {
		cmd := exec.Command(config.Command, config.Args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Generate sandbox profile
	profile := generateSandboxProfile(config.AllowedPaths)

	// Prepare sandbox-exec command
	args := []string{"-p", profile, config.Command}
	args = append(args, config.Args...)

	cmd := exec.Command("sandbox-exec", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sandbox execution failed: %w", err)
	}

	return nil
}

// generateSandboxProfile creates a sandbox-exec profile with write restrictions
func generateSandboxProfile(allowedPaths []string) string {
	var profile bytes.Buffer

	// Write profile header
	profile.WriteString("(version 1)\n")
	profile.WriteString("(import \"system.sb\")\n")
	profile.WriteString("(deny default)\n")

	// Allow all file reads
	profile.WriteString("(allow file-read*)\n")

	// Deny all file writes by default
	profile.WriteString("(deny file-write*)\n")

	// Allow writes to specified paths
	for _, path := range allowedPaths {
		// Expand path to absolute
		absPath, err := filepath.Abs(path)
		if err != nil {
			// If we can't resolve the path, use it as-is
			absPath = path
		}

		// Resolve symlinks to get the real path
		realPath, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			// If we can't resolve symlinks (e.g., path doesn't exist yet), use the absolute path
			realPath = absPath
		}

		// Escape the path for the sandbox profile
		escapedPath := escapePathForSandbox(realPath)

		// Allow writes to the path and all subpaths
		profile.WriteString(fmt.Sprintf("(allow file-write* (subpath \"%s\"))\n", escapedPath))

		// Also allow writes to the literal path (for directory creation)
		profile.WriteString(fmt.Sprintf("(allow file-write* (literal \"%s\"))\n", escapedPath))
	}

	// Allow network access
	profile.WriteString("(allow network*)\n")

	// Allow process execution
	profile.WriteString("(allow process-exec*)\n")
	profile.WriteString("(allow process-fork)\n")

	// Allow IPC and other system operations
	profile.WriteString("(allow ipc-posix*)\n")
	profile.WriteString("(allow mach*)\n")
	profile.WriteString("(allow sysctl-read)\n")
	profile.WriteString("(allow system-socket)\n")

	// Allow signal operations
	profile.WriteString("(allow signal)\n")

	// Allow various file operations that don't modify content
	profile.WriteString("(allow file-read-metadata)\n")
	profile.WriteString("(allow file-read-xattr)\n")
	profile.WriteString("(allow file-write-xattr)\n")
	profile.WriteString("(allow file-ioctl)\n")

	return profile.String()
}

// escapePathForSandbox escapes special characters in paths for sandbox profiles
func escapePathForSandbox(path string) string {
	// Escape backslashes and double quotes
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, "\"", "\\\"")
	return path
}
