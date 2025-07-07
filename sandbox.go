package main

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

// RunInSandbox executes the given command with sandbox restrictions
// This is implemented differently for each platform
func RunInSandbox(config *SandboxConfig) error {
	return runInSandbox(config)
}
