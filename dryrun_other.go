//go:build !darwin && !linux

package main

import (
	"fmt"
	"os"
	"runtime"
)

// showDryRun displays an error that cage is not supported on this platform
func showDryRun(config *SandboxConfig) error {
	return fmt.Errorf("cage is not supported on %s", runtime.GOOS)
}

// printDryRunAndExit displays the dry-run information and exits
func printDryRunAndExit(config *SandboxConfig) {
	if err := showDryRun(config); err != nil {
		fmt.Fprintf(os.Stderr, "cage: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
