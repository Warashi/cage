//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/landlock-lsm/go-landlock/landlock"
)

// runInSandbox implements sandbox execution for Linux using go-landlock
func runInSandbox(config *SandboxConfig) error {
	// If allow-all is set, run without restrictions
	if config.AllowAll {
		cmd := exec.Command(config.Command, config.Args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Build FSRules
	var rules []landlock.Rule

	// Grant read access to essential system directories for command execution
	// We check if each directory exists before adding it
	systemDirs := []string{
		"/bin",      // System binaries
		"/usr",      // User binaries and libraries
		"/lib",      // System libraries
		"/lib64",    // 64-bit system libraries (may not exist on all systems)
		"/etc",      // Configuration files
		"/dev",      // Device files
		"/proc",     // Process information
		"/sys",      // System information
		"/nix",      // Nix store (if using NixOS)
		"/run",      // Runtime data
		"/home",     // Home directories (read-only access)
	}

	for _, dir := range systemDirs {
		if _, err := os.Stat(dir); err == nil {
			rules = append(rules, landlock.RODirs(dir))
		}
	}

	// Grant read-write access to specified paths
	for _, path := range config.AllowedPaths {
		rules = append(rules, landlock.RWDirs(path))
	}

	// Apply Landlock restrictions using the best available version
	// BestEffort ensures graceful degradation on older kernels
	err := landlock.V5.BestEffort().RestrictPaths(rules...)
	if err != nil {
		return fmt.Errorf("failed to apply Landlock restrictions: %w", err)
	}

	// Run the command with restrictions applied
	cmd := exec.Command(config.Command, config.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}