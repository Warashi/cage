//go:build darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// showDryRun displays the sandbox profile that would be generated for the given configuration
func showDryRun(config *SandboxConfig) error {
	fmt.Println("Sandbox Profile (dry-run):")
	fmt.Println("========================================")
	fmt.Println("Version: macOS Sandbox v1")
	fmt.Println("Base profile: system.sb")
	fmt.Println()
	fmt.Println("Rules:")

	if config.AllowAll {
		fmt.Println("- Allow all operations (--allow-all flag)")
	} else {
		fmt.Println("- Allow all operations by default")
		fmt.Println("- Deny all file writes")
		fmt.Println("- Allow writes to:")
		fmt.Println("  * System temporary directories")

		if config.AllowKeychain {
			fmt.Println("  * Keychain directories (--allow-keychain)")
		}

		// Process allowed paths
		for _, path := range config.AllowedPaths {
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			source := "user specified"
			if config.AllowGit && strings.Contains(path, ".git") {
				source = "--allow-git"
			}
			fmt.Printf("  * %s (%s)\n", absPath, source)
		}
	}

	fmt.Println()
	fmt.Println("Raw profile:")
	fmt.Println("----------------------------------------")

	// Generate and display the actual profile
	profile, err := generateSandboxProfile(config)
	if err != nil {
		return fmt.Errorf("generate sandbox profile: %w", err)
	}
	fmt.Print(profile)
	fmt.Println("----------------------------------------")

	fmt.Println()
	fmt.Printf("Command: %s", config.Command)
	if len(config.Args) > 0 {
		fmt.Printf(" %s", strings.Join(config.Args, " "))
	}
	fmt.Println()

	return nil
}

// printDryRunAndExit displays the dry-run information and exits
func printDryRunAndExit(config *SandboxConfig) {
	if err := showDryRun(config); err != nil {
		fmt.Fprintf(os.Stderr, "cage: error showing dry-run: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
