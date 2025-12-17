//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func runInSandbox(config *SandboxConfig) error {
	profile, err := generateSandboxProfile(config)
	if err != nil {
		return fmt.Errorf("generate sandbox profile: %w", err)
	}

	sandboxPath, err := exec.LookPath("sandbox-exec")
	if err != nil {
		return fmt.Errorf("sandbox-exec not found: %w", err)
	}

	args := []string{"sandbox-exec", "-p", profile, config.Command}
	args = append(args, config.Args...)

	return syscall.Exec(sandboxPath, args, os.Environ())
}

func generateSandboxProfile(config *SandboxConfig) (string, error) {
	var profile bytes.Buffer

	profile.WriteString("(version 1)\n")
	profile.WriteString(`(import "system.sb")` + "\n")
	profile.WriteString("(allow default)\n")

	if config.AllowAll {
		return profile.String(), nil
	}

	profile.WriteString("(deny file-write*)\n")
	profile.WriteString(
		`(allow file-write* (regex #"^/private/var/folders/[^/]+/[^/]+/(C|T|0)($|/)"))` + "\n",
	)

	if config.AllowKeychain {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		fmt.Fprintf(&profile, `(allow file-write* (subpath "%s/Library/Keychains"))`+"\n", homeDir)
	}

	for _, path := range config.AllowedPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		escapedPath := escapePathForSandbox(absPath)

		fmt.Fprintf(&profile, "(allow file-write* (subpath \"%s\"))\n", escapedPath)
		fmt.Fprintf(&profile, "(allow file-write* (literal \"%s\"))\n", escapedPath)
	}

	if config.Strict {
		profile.WriteString("(deny file-read*)\n")

		systemRoots := []string{
			"/usr", "/bin", "/sbin", "/lib", "/etc", "/opt", "/var",
			"/dev", "/System", "/Library", "/Applications",
			"/private/var/folders",
		}
		for _, root := range systemRoots {
			escapedPath := escapePathForSandbox(root)
			fmt.Fprintf(&profile, "(allow file-read* (subpath \"%s\"))\n", escapedPath)
		}

		for _, path := range config.ReadPaths {
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			escapedPath := escapePathForSandbox(absPath)
			fmt.Fprintf(&profile, "(allow file-read* (subpath \"%s\"))\n", escapedPath)
			fmt.Fprintf(&profile, "(allow file-read* (literal \"%s\"))\n", escapedPath)
		}

		for _, path := range config.AllowedPaths {
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			escapedPath := escapePathForSandbox(absPath)
			fmt.Fprintf(&profile, "(allow file-read* (subpath \"%s\"))\n", escapedPath)
			fmt.Fprintf(&profile, "(allow file-read* (literal \"%s\"))\n", escapedPath)
		}
	}

	for _, rule := range config.DenyRules {
		if rule.Modes&AccessRead != 0 {
			if rule.IsGlob {
				regexPattern := globToSBPLRegex(rule.Pattern)
				fmt.Fprintf(&profile, "(deny file-read* (regex #\"%s\"))\n", regexPattern)
			} else {
				absPath, err := filepath.Abs(rule.Pattern)
				if err != nil {
					absPath = rule.Pattern
				}
				escapedPath := escapePathForSandbox(absPath)
				fmt.Fprintf(&profile, "(deny file-read* (subpath \"%s\"))\n", escapedPath)
			}
		}
		if rule.Modes&AccessWrite != 0 {
			if rule.IsGlob {
				regexPattern := globToSBPLRegex(rule.Pattern)
				fmt.Fprintf(&profile, "(deny file-write* (regex #\"%s\"))\n", regexPattern)
			} else {
				absPath, err := filepath.Abs(rule.Pattern)
				if err != nil {
					absPath = rule.Pattern
				}
				escapedPath := escapePathForSandbox(absPath)
				fmt.Fprintf(&profile, "(deny file-write* (subpath \"%s\"))\n", escapedPath)
			}
		}
	}

	return profile.String(), nil
}

func escapePathForSandbox(path string) string {
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, "\"", "\\\"")
	return path
}

func globToSBPLRegex(pattern string) string {
	absPattern, err := filepath.Abs(pattern)
	if err != nil {
		absPattern = pattern
	}

	var result strings.Builder
	result.WriteString("^")

	for i := 0; i < len(absPattern); i++ {
		c := absPattern[i]
		switch c {
		case '*':
			if i+1 < len(absPattern) && absPattern[i+1] == '*' {
				result.WriteString(".*")
				i++
			} else {
				result.WriteString("[^/]*")
			}
		case '?':
			result.WriteString("[^/]")
		case '.', '(', ')', '[', ']', '{', '}', '+', '^', '$', '|', '\\':
			result.WriteByte('\\')
			result.WriteByte(c)
		default:
			result.WriteByte(c)
		}
	}

	result.WriteString("($|/)")
	return result.String()
}
