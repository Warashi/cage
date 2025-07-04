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
	// If allow-all is set, run without restrictions
	if config.AllowAll {
		// Find the absolute path of the command
		path, err := exec.LookPath(config.Command)
		if err != nil {
			return fmt.Errorf("command not found: %w", err)
		}

		// Prepare arguments: first arg is the program name
		args := append([]string{config.Command}, config.Args...)

		// Replace current process with the command
		return syscall.Exec(path, args, os.Environ())
	}

	// Generate sandbox profile
	profile := generateSandboxProfile(config.AllowedPaths)

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
func generateSandboxProfile(allowedPaths []string) string {
	var profile bytes.Buffer

	// Write profile header
	profile.WriteString("(version 1)\n")
	profile.WriteString("(allow default)\n")

	// Deny writes to all paths except allowed ones
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

	return profile.String()
}

// escapePathForSandbox escapes special characters in paths for sandbox profiles
func escapePathForSandbox(path string) string {
	// Escape backslashes and double quotes
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, "\"", "\\\"")
	return path
}
