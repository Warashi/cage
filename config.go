package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Presets     map[string]Preset `yaml:"presets"`
	AutoPresets []AutoPresetRule  `yaml:"auto-presets"`
}

type Preset struct {
	Allow         []AllowPath `yaml:"allow"`
	AllowKeychain bool        `yaml:"allow-keychain"`
	AllowGit      bool        `yaml:"allow-git"`
}

type AllowPath struct {
	Path        string `yaml:"path"`
	EvalSymLink bool   `yaml:"eval-symlink,omitempty"`
}

type AutoPresetRule struct {
	Command        string   `yaml:"command,omitempty"`
	CommandPattern string   `yaml:"command-pattern,omitempty"`
	Presets        []string `yaml:"presets"`
}

func (p *AllowPath) UnmarshalYAML(b []byte) error {
	var a any
	if err := yaml.Unmarshal(b, &a); err != nil {
		return fmt.Errorf("unmarshal AllowPath: %w", err)
	}
	switch v := a.(type) {
	case string:
		*p = AllowPath{
			Path:        v,
			EvalSymLink: false,
		}
		return nil
	case map[string]any:
		type alias AllowPath
		var ap alias
		if err := yaml.Unmarshal(b, &ap); err != nil {
			return fmt.Errorf("unmarshal AllowPath map: %w", err)
		}
		*p = (AllowPath)(ap)
		return nil
	default:
		return fmt.Errorf("unmarshal AllowPath: unsupported type %T", a)
	}
}

func userConfigDir() (string, error) {
	// os.UserConfigDir() does not respect XDG_CONFIG_HOME on darwin.
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir, nil
	}
	return os.UserConfigDir()
}

func loadConfig(configPath string) (*Config, error) {
	paths := []string{}

	if configPath != "" {
		paths = append(paths, configPath)
	} else {
		configDir, err := userConfigDir()
		if err == nil {
			paths = append(paths, filepath.Join(configDir, "cage", "presets.yaml"))
			paths = append(paths, filepath.Join(configDir, "cage", "presets.yml"))
		}
	}

	for _, path := range paths {
		config, err := loadConfigFromFile(path)
		if err == nil {
			return config, nil
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading config from %s: %w", path, err)
		}
	}

	return &Config{Presets: make(map[string]Preset)}, nil
}

func loadConfigFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) GetPreset(name string) (Preset, bool) {
	preset, ok := c.Presets[name]
	return preset, ok
}

func (c *Config) ListPresets() []string {
	presets := make([]string, 0, len(c.Presets))
	for name := range c.Presets {
		presets = append(presets, name)
	}
	return presets
}

// GetAutoPresets returns the preset names that should be automatically applied for the given command
func (c *Config) GetAutoPresets(command string) ([]string, error) {
	var presets []string

	// Extract just the base command name from the full path
	baseCommand := filepath.Base(command)

	for _, rule := range c.AutoPresets {
		matched := false

		// Check exact command match
		if rule.Command != "" && rule.Command == baseCommand {
			matched = true
		}

		// Check regex pattern match
		if !matched && rule.CommandPattern != "" {
			re, err := regexp.Compile(rule.CommandPattern)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid regex pattern in auto-preset: %s: %w",
					rule.CommandPattern,
					err,
				)
			}
			if re.MatchString(baseCommand) {
				matched = true
			}
		}

		if matched {
			presets = append(presets, rule.Presets...)
		}
	}

	return presets, nil
}

// expandEnvOnly expands environment variables in a path
// This is safer than shell expansion as it doesn't allow command execution
func expandEnvOnly(path string) string {
	return os.ExpandEnv(path)
}

// getGitCommonDir returns the git common directory for the current repository
// This is useful for git worktrees where the .git directory is a file pointing to the common dir
func getGitCommonDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git common directory: %w", err)
	}
	// Trim newline from output
	return strings.TrimSpace(string(output)), nil
}

// ProcessPreset expands all dynamic values in a preset
func (p *Preset) ProcessPreset() (*Preset, error) {
	processed := &Preset{
		AllowKeychain: p.AllowKeychain,
		AllowGit:      p.AllowGit,
		Allow:         make([]AllowPath, 0, len(p.Allow)),
	}

	// Expand environment variables in paths
	for _, path := range p.Allow {
		expanded := os.ExpandEnv(path.Path)
		if path.EvalSymLink {
			// Resolve symlinks if EvalSymLink is true
			resolvedPath, err := filepath.EvalSymlinks(expanded)
			if err != nil {
				resolvedPath = expanded // Fallback to original path if eval fails
			}
			expanded = resolvedPath
		}

		processed.Allow = append(processed.Allow, AllowPath{Path: expanded})
	}

	// Add git common directory if AllowGit is enabled
	if p.AllowGit {
		gitCommonDir, err := getGitCommonDir()
		if err != nil {
			// Log the error but don't fail - the directory might not be a git repo
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		} else {
			processed.Allow = append(processed.Allow, AllowPath{Path: gitCommonDir})
		}
	}

	return processed, nil
}
