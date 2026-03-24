//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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

	// Fall back to bubblewrap when DeniedPaths are specified,
	// since Landlock LSM uses an allowlist model and cannot deny reads.
	if len(config.DeniedPaths) > 0 {
		return runWithBubblewrap(config)
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
			continue
		}

		// Use appropriate rule based on file type
		if info.IsDir() {
			if path == "/dev" || strings.HasPrefix(path, "/dev/") {
				rules = append(rules, landlock.RWDirs(path).WithIoctlDev())
				continue
			}
			rules = append(rules, landlock.RWDirs(path).WithRefer())
		} else {
			if strings.HasPrefix(path, "/dev/") {
				rules = append(rules, landlock.RWFiles(path).WithIoctlDev())
				continue
			}
			rules = append(rules, landlock.RWFiles(path))
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

// runWithBubblewrap implements sandbox execution for Linux using bubblewrap (bwrap)
// This is used as a fallback when DeniedPaths are specified, since Landlock does not support deny rules.
func runWithBubblewrap(config *SandboxConfig) error {
	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		return fmt.Errorf("-deny flag on Linux requires bubblewrap (bwrap): %w", err)
	}

	args := []string{"bwrap"}

	// Bind the entire root filesystem as read-only (equivalent to Landlock's RODirs("/"))
	args = append(args, "--ro-bind", "/", "/")

	// Allow write access to /dev/null by default (equivalent to Landlock's RWFiles("/dev/null"))
	args = append(args, "--dev-bind", "/dev/null", "/dev/null")

	// Grant write access to specified paths
	for _, path := range config.AllowedPaths {
		// Check if the path exists before adding the rule
		if _, err := os.Stat(path); err != nil {
			continue
		}
		// Use --dev-bind for /dev and paths under /dev/ (equivalent to Landlock's WithIoctlDev)
		if path == "/dev" || strings.HasPrefix(path, "/dev/") {
			args = append(args, "--dev-bind", path, path)
			continue
		}
		// Use --bind for other paths (works for both files and directories)
		args = append(args, "--bind", path, path)
	}

	// Deny read access to specified paths (placed last to take priority over allow rules)
	for _, path := range config.DeniedPaths {
		// Check if the path exists before adding the rule
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cage: warning: deny path does not exist, skipping: %s\n", path)
			continue
		}
		if info.IsDir() {
			// Directory: mount tmpfs to hide contents
			args = append(args, "--tmpfs", path)
		} else {
			// File: bind /dev/null over the file to make it appear empty
			args = append(args, "--bind", "/dev/null", path)
		}
	}

	args = append(args, "--")
	args = append(args, config.Command)
	args = append(args, config.Args...)

	return syscall.Exec(bwrapPath, args, os.Environ())
}
