//go:build linux

package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func showDryRun(config *SandboxConfig) error {
	fmt.Println("Sandbox Profile (dry-run):")
	fmt.Println("========================================")
	fmt.Println("Platform: Linux")
	fmt.Println("Technology: Landlock LSM")
	fmt.Println()
	fmt.Println("The following restrictions would be applied:")
	fmt.Println()
	fmt.Println("Rules:")

	if config.AllowAll {
		fmt.Println("- Allow all operations (-allow-all flag)")
	} else {
		if config.Strict {
			fmt.Println("- STRICT MODE: Only explicit read paths are allowed")
			fmt.Println("- Allow read access to:")
			fmt.Println("  * System paths (/usr, /bin, /sbin, /lib, /lib64, /etc, /opt, /var, /dev, /proc, /sys)")

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
		} else {
			fmt.Println("- Allow read access to all files")
		}

		fmt.Println("- Deny write access except to:")
		fmt.Println("  * /dev/null (for discarding output)")

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
				note := ""
				if rule.Modes&AccessRead != 0 {
					if rule.IsGlob {
						note = " (WARNING: glob patterns not supported on Linux)"
					} else {
						note = " (WARNING: read deny only effective with --strict on Linux)"
					}
				}
				fmt.Printf("  * %s (%s)%s\n", absPath, modeStr, note)
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
