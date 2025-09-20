package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

// SandboxConfig contains the configuration for running a command in a sandbox
type SandboxConfig struct {
	// AllowAll disables all restrictions (for testing/debugging)
	AllowAll bool

	// AllowKeychain allows access to the keychain
	// This is only applicable on macOS
	AllowKeychain bool

	// AllowGit allows access to git common directory
	// This enables git operations in worktrees
	AllowGit bool

	// AllowedPaths are paths where write access is granted
	AllowedPaths []string

	// Command is the command to execute
	Command string

	// Args are the arguments to pass to the command
	Args []string
}

func modifySandboxConfig(config *SandboxConfig) {
	pathSet := make(map[string]struct{})
	for _, path := range config.AllowedPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		pathSet[absPath] = struct{}{}
	}

	// Add git common directory if allowGit is enabled and not already handled by preset
	if config.AllowGit {
		gitCommonDir, err := getGitCommonDir()
		if err != nil {
			// Log the error but don't fail - the directory might not be a git repo
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		} else {
			pathSet[gitCommonDir] = struct{}{}
		}
	}

	config.AllowedPaths = slices.Sorted(maps.Keys(pathSet))
}

// RunInSandbox executes the given command with sandbox restrictions
// This is implemented differently for each platform
func RunInSandbox(config *SandboxConfig) error {
	modifySandboxConfig(config)
	return runInSandbox(config)
}
