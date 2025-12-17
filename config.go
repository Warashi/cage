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

var BuiltinPresets = map[string]Preset{
	"secure": {
		Extends: []string{"builtin:strict-base", "builtin:secrets-deny"},
		Allow: []AllowPath{
			{Path: "."},
			{Path: "$HOME/.local/share"},
			{Path: "$HOME/.local/state"},

			// AI coding tools config
			{Path: "$HOME/.bun"}, // Bun package manager (used by opencode)
			{Path: "$HOME/.cache/opencode"},
			{Path: "$HOME/.claude"},
			{Path: "$HOME/.codeium"}, // Codeium
			{Path: "$HOME/.cody"},    // Sourcegraph Cody
			{Path: "$HOME/.config/aider"},
			{Path: "$HOME/.config/claude"},
			{Path: "$HOME/.config/opencode"},
			{Path: "$HOME/.continue"}, // Continue.dev
			{Path: "$HOME/.cursor"},   // Cursor editor
			{Path: "$HOME/.tabby"},    // Tabby

			// IDE/editor config
			{Path: "$HOME/.config/Code"},
			{Path: "$HOME/.config/Cursor"},
			{Path: "$HOME/.config/JetBrains"},
			{Path: "$HOME/.config/VSCodium"},
			{Path: "$HOME/.idea"},
			{Path: "$HOME/.vscode"},
			{Path: "$HOME/.vscode-server"},

			// Shell tools
			{Path: "$HOME/.cache/starship"},
		},
		AllowGit: true,
	},
	"strict-base": {
		Strict: true,
		Read: []AllowPath{
			{Path: "/Applications"},
			{Path: "/Library"},
			{Path: "/System"},
			{Path: "/bin"},
			{Path: "/dev"},
			{Path: "/etc"},
			{Path: "/lib"},
			{Path: "/lib64"},
			{Path: "/opt"},
			{Path: "/private/var"},
			{Path: "/private/var/folders"},
			{Path: "/proc"},
			{Path: "/sbin"},
			{Path: "/sys"},
			{Path: "/usr"},
			{Path: "/var"},
			{Path: "$HOME/.config/fish"},
		},
	},
	"secrets-deny": {
		Deny: []AllowPath{
			// SSH keys and config
			{Path: "$HOME/.ssh"},

			// Cloud provider credentials
			{Path: "$HOME/.aws"},
			{Path: "$HOME/.azure"},
			{Path: "$HOME/.config/gcloud"},
			{Path: "$HOME/.config/doctl"},    // DigitalOcean
			{Path: "$HOME/.config/flyctl"},   // Fly.io
			{Path: "$HOME/.config/hcloud"},   // Hetzner Cloud
			{Path: "$HOME/.config/linode"},   // Linode
			{Path: "$HOME/.config/scaleway"}, // Scaleway

			// Container and orchestration
			{Path: "$HOME/.kube"},
			{Path: "$HOME/.docker/config.json"},
			{Path: "$HOME/.helm"},
			{Path: "$HOME/.config/containers"}, // Podman auth
			{Path: "$HOME/.lima/_config"},      // Lima VM SSH keys
			{Path: "$HOME/.rd"},                // Rancher Desktop
			{Path: "$HOME/.config/k3d"},        // k3d kubeconfigs
			{Path: "$HOME/.config/Lens"},       // Lens IDE
			{Path: "$HOME/.config/OpenLens"},   // OpenLens IDE

			// CI/CD and deployment platforms
			{Path: "$HOME/.config/vercel"},
			{Path: "$HOME/.config/netlify"},
			{Path: "$HOME/.config/railway"},
			{Path: "$HOME/.config/heroku"},
			{Path: "$HOME/.config/circleci"},

			// Git forges and CLI tools
			{Path: "$HOME/.config/gh"},   // GitHub CLI
			{Path: "$HOME/.config/hub"},  // Hub CLI
			{Path: "$HOME/.config/glab"}, // GitLab CLI
			{Path: "$HOME/.git-credentials"},
			{Path: "$HOME/.netrc"},

			// Security and encryption
			{Path: "$HOME/.gnupg"},
			{Path: "$HOME/.config/sops/age"}, // SOPS age keys
			{Path: "$HOME/.config/op"},       // 1Password CLI

			// Package manager credentials
			{Path: "$HOME/.npmrc"},
			{Path: "$HOME/.pypirc"},
			{Path: "$HOME/.config/pip"},
			{Path: "$HOME/.config/configstore"}, // npm/yarn token storage
			{Path: "$HOME/.cargo/credentials.toml"},

			// Security scanning and dev tools
			{Path: "$HOME/.config/snyk"},
			{Path: "$HOME/.config/ngrok"},

			// Shell history (contains commands, secrets in env vars)
			{Path: "$HOME/.bash_history"},
			{Path: "$HOME/.zsh_history"},
			{Path: "$HOME/.local/share/atuin"}, // Atuin shell history
			{Path: "$HOME/.local/share/fish/fish_history"},
			{Path: "$HOME/.node_repl_history"},
			{Path: "$HOME/.python_history"},
			{Path: "$HOME/.psql_history"},
			{Path: "$HOME/.mysql_history"},
			{Path: "$HOME/.rediscli_history"},

			// macOS sensitive data
			{Path: "$HOME/Library"},

			// Browser data (sessions, cookies, saved passwords)
			{Path: "$HOME/.config/google-chrome"},
			{Path: "$HOME/.config/chromium"},
			{Path: "$HOME/.mozilla/firefox"},
			{Path: "$HOME/.config/BraveSoftware"},
		},
	},
	"home-dotfiles-deny": {
		Deny: []AllowPath{
			{Path: "$HOME/.*"},
		},
	},
	"safe-home": {
		Strict: true,
		Read: []AllowPath{
			{Path: "/usr"},
			{Path: "/bin"},
			{Path: "/sbin"},
			{Path: "/lib"},
			{Path: "/lib64"},
			{Path: "/etc"},
			{Path: "/opt"},
			{Path: "/var"},
			{Path: "/dev"},
			{Path: "/proc"},
			{Path: "/sys"},
			{Path: "/System"},
			{Path: "/Library"},
			{Path: "/Applications"},
			{Path: "/private/var/folders"},
			{Path: "$HOME/Documents"},
			{Path: "$HOME/Downloads"},
			{Path: "$HOME/Desktop"},
			{Path: "$HOME/Pictures"},
			{Path: "$HOME/Music"},
			{Path: "$HOME/Videos"},
			{Path: "$HOME/Movies"},
			{Path: "$HOME/Projects"},
			{Path: "$HOME/Developer"},
			{Path: "$HOME/Code"},
			{Path: "$HOME/src"},
			{Path: "$HOME/go/src"},
			{Path: "$HOME/workspace"},
		},
	},
	"npm": {
		Allow: []AllowPath{
			{Path: "."},
			{Path: "$HOME/.npm"},
			{Path: "$HOME/.cache/npm"},
			{Path: "node_modules"},
		},
	},
	"cargo": {
		Allow: []AllowPath{
			{Path: "."},
			{Path: "$HOME/.cargo"},
			{Path: "$HOME/.rustup"},
			{Path: "target"},
		},
	},
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
