package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type flags struct {
	allowAll      bool
	allowKeychain bool
	allowGit      bool
	allowPaths    []string
	presets       []string
	listPresets   bool
	configPath    string
}

func parseFlags() (*flags, []string) {
	f := &flags{}

	flag.BoolVar(
		&f.allowAll,
		"allow-all",
		false,
		"Disable all restrictions (use for testing/debugging only)",
	)

	flag.BoolVar(
		&f.allowKeychain,
		"allow-keychain",
		false,
		"Allow write access to the macOS keychain (only for macOS)",
	)

	flag.BoolVar(
		&f.allowGit,
		"allow-git",
		false,
		"Allow access to git common directory (enables git operations in worktrees)",
	)

	// Custom flag parsing to handle multiple --allow flags
	var allowFlags arrayFlags
	flag.Var(
		&allowFlags,
		"allow",
		"Grant write access to specific paths (can be used multiple times)",
	)

	// Custom flag parsing to handle multiple --preset flags
	var presetFlags arrayFlags
	flag.Var(
		&presetFlags,
		"preset",
		"Use a predefined preset configuration (can be used multiple times)",
	)

	flag.BoolVar(
		&f.listPresets,
		"list-presets",
		false,
		"List available presets",
	)

	flag.StringVar(
		&f.configPath,
		"config",
		"",
		"Path to custom configuration file",
	)

	flag.Parse()

	f.allowPaths = []string(allowFlags)
	f.presets = []string(presetFlags)

	return f, flag.Args()
}

// arrayFlags is a custom flag type that accumulates values
type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ", ")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func main() {
	flags, args := parseFlags()

	// Load configuration
	config, err := loadConfig(flags.configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cage: error loading config: %v\n", err)
		os.Exit(1)
	}

	// Handle list-presets flag
	if flags.listPresets {
		presets := config.ListPresets()
		if len(presets) == 0 {
			fmt.Println("No presets available")
		} else {
			fmt.Println("Available presets:")
			for _, name := range presets {
				fmt.Printf("  - %s\n", name)
			}
		}
		os.Exit(0)
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: cage [flags] <command> [command-args...]\n")
		fmt.Fprintf(
			os.Stderr,
			"       cage [flags] -- <command> [command-flags] [command-args...]\n",
		)
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Merge preset paths with command-line paths
	allowedPaths := flags.allowPaths
	allowKeychain := flags.allowKeychain
	allowGit := flags.allowGit

	// Process each preset and merge their settings
	for _, presetName := range flags.presets {
		preset, ok := config.GetPreset(presetName)
		if !ok {
			fmt.Fprintf(os.Stderr, "cage: preset '%s' not found\n", presetName)
			os.Exit(1)
		}

		// Process preset to expand dynamic values
		processedPreset, err := preset.ProcessPreset()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cage: error processing preset '%s': %v\n", presetName, err)
			os.Exit(1)
		}

		// Preset paths are added before command-line paths
		allowedPaths = append(processedPreset.Allow, allowedPaths...)

		// Preset's allowKeychain is ORed with command-line flag
		allowKeychain = allowKeychain || processedPreset.AllowKeychain

		// Preset's allowGit is ORed with command-line flag
		allowGit = allowGit || processedPreset.AllowGit
	}

	// Add git common directory if allowGit is enabled and not already handled by preset
	if allowGit && len(flags.presets) == 0 {
		gitCommonDir, err := getGitCommonDir()
		if err != nil {
			// Log the error but don't fail - the directory might not be a git repo
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		} else {
			allowedPaths = append(allowedPaths, gitCommonDir)
		}
	}

	// Create sandbox configuration
	sandboxConfig := &SandboxConfig{
		AllowAll:      flags.allowAll,
		AllowKeychain: allowKeychain,
		AllowGit:      allowGit,
		AllowedPaths:  allowedPaths,
		Command:       args[0],
		Args:          args[1:],
	}

	// Execute in sandbox
	if err := RunInSandbox(sandboxConfig); err != nil {
		fmt.Fprintf(os.Stderr, "cage: %v\n", err)
		os.Exit(1)
	}
}
