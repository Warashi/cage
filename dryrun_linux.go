//go:build linux

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// showDryRun displays the sandbox configuration that would be applied for the given configuration
func showDryRun(config *SandboxConfig) error {
	fmt.Println("Sandbox Profile (dry-run):")
	fmt.Println("========================================")
	fmt.Println("Platform: Linux")
	if len(config.DeniedPaths) > 0 {
		fmt.Println("Technology: bubblewrap (bwrap)")
		if _, err := exec.LookPath("bwrap"); err != nil {
			fmt.Println(
				"WARNING: bubblewrap (bwrap) is not installed; -deny flag will fail at runtime",
			)
		}
	} else {
		fmt.Println("Technology: Landlock LSM")
	}
	fmt.Println()
	fmt.Println("The following restrictions would be applied:")
	fmt.Println()
	fmt.Println("Rules:")

	if config.AllowAll {
		fmt.Println("- Allow all operations (-allow-all flag)")
	} else {
		fmt.Println("- Allow read access to all files")
		fmt.Println("- Deny write access except to:")
		fmt.Println("  * /dev/null (for discarding output)")

		// Process allowed paths
		for _, path := range config.AllowedPaths {
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			source := "user specified"
			if config.AllowGit && strings.Contains(path, ".git") {
				source = "-allow-git"
			}
			fmt.Printf("  * %s (%s)\n", absPath, source)
		}

		if len(config.DeniedPaths) > 0 {
			fmt.Println("- Deny read access to:")
			for _, path := range config.DeniedPaths {
				absPath, err := filepath.Abs(path)
				if err != nil {
					absPath = path
				}
				fmt.Printf("  * %s\n", absPath)
			}
		}
	}

	fmt.Println()
	fmt.Printf("Command: %s", config.Command)
	if len(config.Args) > 0 {
		fmt.Printf(" %s", strings.Join(config.Args, " "))
	}
	fmt.Println()

	return nil
}
