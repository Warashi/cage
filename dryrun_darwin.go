//go:build darwin

package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func showDryRun(config *SandboxConfig) error {
	fmt.Println("Sandbox Profile (dry-run):")
	fmt.Println("========================================")
	fmt.Println("Version: macOS Sandbox v1")
	fmt.Println("Base profile: system.sb")
	fmt.Println()
	fmt.Println("Rules:")

	if config.AllowAll {
		fmt.Println("- Allow all operations (-allow-all flag)")
	} else {
		fmt.Println("- Allow all operations by default")
		fmt.Println("- Deny all file writes")
		fmt.Println("- Allow writes to:")
		fmt.Println("  * System temporary directories")

		if config.AllowKeychain {
			fmt.Println("  * Keychain directories (-allow-keychain)")
		}

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

		if config.Strict {
			fmt.Println()
			fmt.Println("- STRICT MODE: Deny all file reads by default")
			fmt.Println("- Allow reads to:")
			fmt.Println("  * System paths (/usr, /bin, /sbin, /lib, /etc, /opt, /var, /dev)")
			fmt.Println("  * macOS paths (/System, /Library, /Applications)")

			for _, path := range config.ReadPaths {
				absPath, err := filepath.Abs(path)
				if err != nil {
					absPath = path
				}
				fmt.Printf("  * %s (user specified)\n", absPath)
			}

			for _, path := range config.AllowedPaths {
				absPath, err := filepath.Abs(path)
				if err != nil {
					absPath = path
				}
				fmt.Printf("  * %s (implicit from write allow)\n", absPath)
			}
		}

		if len(config.DenyRules) > 0 {
			fmt.Println()
			fmt.Println("- Deny rules:")
			for _, rule := range config.DenyRules {
				modeStr := ""
				switch rule.Modes {
				case AccessRead:
					modeStr = "read"
				case AccessWrite:
					modeStr = "write"
				case AccessReadWrite:
					modeStr = "read+write"
				}
				absPath, err := filepath.Abs(rule.Pattern)
				if err != nil {
					absPath = rule.Pattern
				}
				globNote := ""
				if rule.IsGlob {
					globNote = " (glob pattern)"
				}
				fmt.Printf("  * %s (%s)%s\n", absPath, modeStr, globNote)
			}
		}
	}

	fmt.Println()
	fmt.Println("Raw profile:")
	fmt.Println("----------------------------------------")

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
