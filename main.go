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
	allowPaths    []string
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

	// Custom flag parsing to handle multiple --allow flags
	var allowFlags arrayFlags
	flag.Var(
		&allowFlags,
		"allow",
		"Grant write access to specific paths (can be used multiple times)",
	)

	flag.Parse()

	f.allowPaths = []string(allowFlags)

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

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: cage [flags] <command> [command-args...]\n")
		fmt.Fprintf(
			os.Stderr,
			"       cage [flags] -- <command> [command-flags] [command-args...]\n",
		)
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Create sandbox configuration
	config := &SandboxConfig{
		AllowAll:      flags.allowAll,
		AllowKeychain: flags.allowKeychain,
		AllowedPaths:  flags.allowPaths,
		Command:       args[0],
		Args:          args[1:],
	}

	// Execute in sandbox
	if err := RunInSandbox(config); err != nil {
		fmt.Fprintf(os.Stderr, "cage: %v\n", err)
		os.Exit(1)
	}
}
