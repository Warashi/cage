//go:build !darwin

package main

import (
	"fmt"
	"runtime"
)

// runInSandbox is not implemented for platforms other than Darwin
func runInSandbox(config *SandboxConfig) error {
	return fmt.Errorf("sandboxing is not yet implemented for %s", runtime.GOOS)
}
