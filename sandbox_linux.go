//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/landlock-lsm/go-landlock/landlock"
)

// runInSandbox implements sandbox execution for Linux using go-landlock
func runInSandbox(config *SandboxConfig) error {
	// If allow-all is set, run without restrictions
	if config.AllowAll {
		// Find the absolute path of the command
		path, err := exec.LookPath(config.Command)
		if err != nil {
			return fmt.Errorf("command not found: %w", err)
		}

		// Prepare argv: command + args
		argv := append([]string{config.Command}, config.Args...)
		return syscall.Exec(path, argv, os.Environ())
	}

	// Build FSRules
	var rules []landlock.Rule

	// Grant read and execute access to the entire filesystem by default
	// This allows all file reads and command executions
	rules = append(rules, landlock.RODirs("/"))

	// Grant write access to /dev/null by default
	// Many programs write to /dev/null for discarding output
	rules = append(rules, landlock.RWFiles("/dev/null"))

	// Grant read-write access to specified paths
	for _, path := range config.AllowedPaths {
		// Check if the path exists before adding the rule
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				// Skip non-existent paths silently
				continue
			}
			// For other errors, still try to add the rule as a directory
			rules = append(rules, landlock.RWDirs(path).WithIoctlDev())
			continue
		}

		// Use appropriate rule based on file type
		if info.IsDir() {
			rules = append(rules, landlock.RWDirs(path).WithIoctlDev())
		} else {
			// For regular files, device files, etc.
			rules = append(rules, landlock.RWFiles(path).WithIoctlDev())
		}
	}

	// Apply Landlock restrictions using the best available version
	// BestEffort ensures graceful degradation on older kernels
	err := landlock.V5.BestEffort().RestrictPaths(rules...)
	if err != nil {
		return fmt.Errorf("failed to apply Landlock restrictions: %w", err)
	}

	// Find the absolute path of the command
	path, err := exec.LookPath(config.Command)
	if err != nil {
		return fmt.Errorf("command not found: %w", err)
	}

	// Execute the command with restrictions applied
	// syscall.Exec replaces the current process
	argv := append([]string{config.Command}, config.Args...)
	err = syscall.Exec(path, argv, os.Environ())
	// If we reach here, exec failed
	return fmt.Errorf("syscall.Exec failed: %w", err)
}
