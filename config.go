package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed builtin_presets.yaml
var builtinPresetsYAML []byte

var BuiltinPresets map[string]Preset

func init() {
	var config struct {
		Presets map[string]Preset `yaml:"presets"`
	}
	if err := yaml.Unmarshal(builtinPresetsYAML, &config); err != nil {
		panic("failed to parse builtin presets: " + err.Error())
	}
	BuiltinPresets = config.Presets
}

type Config struct {
	Defaults    Defaults          `yaml:"defaults"`
	Presets     map[string]Preset `yaml:"presets"`
	AutoPresets []AutoPresetRule  `yaml:"auto-presets"`
}

type Defaults struct {
	Presets []string `yaml:"presets"`
}

type Preset struct {
	Extends       []string    `yaml:"extends,omitempty"`
	SkipDefaults  bool        `yaml:"skip-defaults,omitempty"`
	Strict        bool        `yaml:"strict,omitempty"`
	Allow         []AllowPath `yaml:"allow,omitempty"`
	AllowKeychain bool        `yaml:"allow-keychain"`
	AllowGit      bool        `yaml:"allow-git"`
	Read          []AllowPath `yaml:"read,omitempty"`
	Deny          []AllowPath `yaml:"deny,omitempty"`
	DenyRead      []AllowPath `yaml:"deny-read,omitempty"`
	DenyWrite     []AllowPath `yaml:"deny-write,omitempty"`
}

type AllowPath struct {
	Path         string `yaml:"path"`
	EvalSymLinks bool   `yaml:"eval-symlinks,omitempty"`
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
			Path:         v,
			EvalSymLinks: false,
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
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
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
	if strings.HasPrefix(name, "builtin:") {
		builtinName := strings.TrimPrefix(name, "builtin:")
		preset, ok := BuiltinPresets[builtinName]
		return preset, ok
	}
	preset, ok := c.Presets[name]
	return preset, ok
}

func (c *Config) ResolvePreset(name string, visited map[string]bool) (*Preset, error) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	if visited[name] {
		return nil, fmt.Errorf("circular preset reference detected: %s", name)
	}
	visited[name] = true

	preset, ok := c.GetPreset(name)
	if !ok {
		return nil, fmt.Errorf("preset not found: %s", name)
	}

	if len(preset.Extends) == 0 {
		return &preset, nil
	}

	merged := &Preset{}

	for _, parentName := range preset.Extends {
		parent, err := c.ResolvePreset(parentName, visited)
		if err != nil {
			return nil, fmt.Errorf("resolving parent preset %s: %w", parentName, err)
		}
		mergePresets(merged, parent)
	}

	mergePresets(merged, &preset)

	return merged, nil
}

func mergePresets(dst, src *Preset) {
	dst.Allow = append(dst.Allow, src.Allow...)
	dst.Read = append(dst.Read, src.Read...)
	dst.Deny = append(dst.Deny, src.Deny...)
	dst.DenyRead = append(dst.DenyRead, src.DenyRead...)
	dst.DenyWrite = append(dst.DenyWrite, src.DenyWrite...)

	dst.Strict = dst.Strict || src.Strict
	dst.SkipDefaults = dst.SkipDefaults || src.SkipDefaults
	dst.AllowKeychain = dst.AllowKeychain || src.AllowKeychain
	dst.AllowGit = dst.AllowGit || src.AllowGit
}

func (c *Config) ListPresets() []string {
	presets := make([]string, 0, len(c.Presets)+len(BuiltinPresets))
	for name := range BuiltinPresets {
		presets = append(presets, "builtin:"+name)
	}
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
		SkipDefaults:  p.SkipDefaults,
		Strict:        p.Strict,
		AllowKeychain: p.AllowKeychain,
		AllowGit:      p.AllowGit,
		Allow:         make([]AllowPath, 0, len(p.Allow)),
		Read:          make([]AllowPath, 0, len(p.Read)),
		Deny:          make([]AllowPath, 0, len(p.Deny)),
		DenyRead:      make([]AllowPath, 0, len(p.DenyRead)),
		DenyWrite:     make([]AllowPath, 0, len(p.DenyWrite)),
	}

	expandPath := func(path AllowPath) AllowPath {
		expanded := os.ExpandEnv(path.Path)
		if path.EvalSymLinks {
			resolvedPath, err := filepath.EvalSymlinks(expanded)
			if err == nil {
				expanded = resolvedPath
			}
		}
		return AllowPath{Path: expanded}
	}

	for _, path := range p.Allow {
		processed.Allow = append(processed.Allow, expandPath(path))
	}
	for _, path := range p.Read {
		processed.Read = append(processed.Read, expandPath(path))
	}
	for _, path := range p.Deny {
		processed.Deny = append(processed.Deny, expandPath(path))
	}
	for _, path := range p.DenyRead {
		processed.DenyRead = append(processed.DenyRead, expandPath(path))
	}
	for _, path := range p.DenyWrite {
		processed.DenyWrite = append(processed.DenyWrite, expandPath(path))
	}

	return processed, nil
}
