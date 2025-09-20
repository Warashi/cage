package main

import (
	"fmt"
	"os"
)

// printDryRunAndExit displays the dry-run information and exits
func printDryRunAndExit(config *SandboxConfig) {
	modifySandboxConfig(config)
	if err := showDryRun(config); err != nil {
		fmt.Fprintf(os.Stderr, "cage: error showing dry-run: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
