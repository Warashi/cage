package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() (string, func())
		wantErr   bool
		checkFunc func(*Config) error
	}{
		{
			name: "load valid config file",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "test.yaml")
				content := `presets:
  test:
    allow:
      - "/tmp"
      - "/var"`
				os.WriteFile(configPath, []byte(content), 0o644)
				return configPath, func() {}
			},
			wantErr: false,
			checkFunc: func(c *Config) error {
				preset, ok := c.GetPreset("test")
				if !ok {
					t.Error("preset 'test' not found")
				}
				if len(preset.Allow) != 2 {
					t.Errorf("expected 2 allow paths, got %d", len(preset.Allow))
				}
				return nil
			},
		},
		{
			name: "config file not found returns empty config",
			setupFunc: func() (string, func()) {
				return "/nonexistent/path.yaml", func() {}
			},
			wantErr: false,
			checkFunc: func(c *Config) error {
				if len(c.Presets) != 0 {
					t.Errorf("expected empty presets, got %d", len(c.Presets))
				}
				return nil
			},
		},
		{
			name: "invalid yaml syntax",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "invalid.yaml")
				content := `presets:
  test:
    allow: [
      invalid yaml`
				os.WriteFile(configPath, []byte(content), 0o644)
				return configPath, func() {}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := tt.setupFunc()
			defer cleanup()

			config, err := loadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkFunc != nil && err == nil {
				if err := tt.checkFunc(config); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestConfigGetPreset(t *testing.T) {
	config := &Config{
		Presets: map[string]Preset{
			"test": {
				Allow: []AllowPath{{Path: "/tmp"}, {Path: "/var"}},
			},
		},
	}

	tests := []struct {
		name       string
		presetName string
		wantFound  bool
		wantPaths  int
	}{
		{
			name:       "existing preset",
			presetName: "test",
			wantFound:  true,
			wantPaths:  2,
		},
		{
			name:       "non-existing preset",
			presetName: "nonexistent",
			wantFound:  false,
			wantPaths:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preset, found := config.GetPreset(tt.presetName)
			if found != tt.wantFound {
				t.Errorf("GetPreset() found = %v, want %v", found, tt.wantFound)
			}
			if found && len(preset.Allow) != tt.wantPaths {
				t.Errorf("GetPreset() paths = %d, want %d", len(preset.Allow), tt.wantPaths)
			}
		})
	}
}

func TestConfigListPresets(t *testing.T) {
	config := &Config{
		Presets: map[string]Preset{
			"npm":   {Allow: []AllowPath{{Path: "~/.npm"}}},
			"cargo": {Allow: []AllowPath{{Path: "~/.cargo"}}},
			"pip":   {Allow: []AllowPath{{Path: "~/.pip"}}},
		},
	}

	presets := config.ListPresets()
	expectedUserPresets := 3
	expectedBuiltinPresets := len(BuiltinPresets)
	expectedTotal := expectedUserPresets + expectedBuiltinPresets

	if len(presets) != expectedTotal {
		t.Errorf("ListPresets() returned %d presets, want %d", len(presets), expectedTotal)
	}

	found := make(map[string]bool)
	for _, name := range presets {
		found[name] = true
	}

	for _, expected := range []string{"npm", "cargo", "pip"} {
		if !found[expected] {
			t.Errorf("ListPresets() missing user preset: %s", expected)
		}
	}

	for name := range BuiltinPresets {
		builtinName := "builtin:" + name
		if !found[builtinName] {
			t.Errorf("ListPresets() missing builtin preset: %s", builtinName)
		}
	}
}

func TestBuiltinPresetsYAMLLoaded(t *testing.T) {
	expectedPresets := []string{
		"secure",
		"strict-base",
		"secrets-deny",
		"home-dotfiles-deny",
		"safe-home",
		"npm",
		"cargo",
	}

	for _, name := range expectedPresets {
		if _, ok := BuiltinPresets[name]; !ok {
			t.Errorf("expected builtin preset %q not found", name)
		}
	}

	if len(BuiltinPresets) != len(expectedPresets) {
		t.Errorf("expected %d builtin presets, got %d", len(expectedPresets), len(BuiltinPresets))
	}
}

func TestBuiltinSecurePreset(t *testing.T) {
	config := &Config{Presets: make(map[string]Preset)}

	resolved, err := config.ResolvePreset("builtin:secure", nil)
	if err != nil {
		t.Fatalf("ResolvePreset(builtin:secure) error = %v", err)
	}

	if !resolved.Strict {
		t.Error("builtin:secure should have Strict=true (inherited from strict-base)")
	}

	if !resolved.AllowGit {
		t.Error("builtin:secure should have AllowGit=true")
	}

	if len(resolved.Allow) == 0 {
		t.Error("builtin:secure should have Allow paths")
	}

	foundCwd := false
	for _, p := range resolved.Allow {
		if p.Path == "." {
			foundCwd = true
			break
		}
	}
	if !foundCwd {
		t.Error("builtin:secure should allow '.' (current directory)")
	}

	if len(resolved.Read) == 0 {
		t.Error("builtin:secure should have Read paths (inherited from strict-base)")
	}

	if len(resolved.Deny) == 0 {
		t.Error("builtin:secure should have Deny paths (inherited from secrets-deny)")
	}

	foundSshDeny := false
	for _, p := range resolved.Deny {
		if p.Path == "$HOME/.ssh" {
			foundSshDeny = true
			break
		}
	}
	if !foundSshDeny {
		t.Error("builtin:secure should deny $HOME/.ssh (inherited from secrets-deny)")
	}
}

func TestExpandEnvOnly(t *testing.T) {
	// Set test environment variable
	t.Setenv("TEST_VAR", "test_value")

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "plain path",
			path: "/tmp/test",
			want: "/tmp/test",
		},
		{
			name: "environment variable with braces",
			path: "${TEST_VAR}/path",
			want: "test_value/path",
		},
		{
			name: "environment variable without braces",
			path: "$TEST_VAR/path",
			want: "test_value/path",
		},
		{
			name: "HOME environment variable",
			path: "$HOME/.config",
			want: os.Getenv("HOME") + "/.config",
		},
		{
			name: "path with spaces",
			path: "/path with spaces/test",
			want: "/path with spaces/test",
		},
		{
			name: "path with special characters",
			path: "/path'with\"quotes/test",
			want: "/path'with\"quotes/test",
		},
		{
			name: "command substitution not expanded",
			path: "$(echo /tmp)/test",
			want: "$(echo /tmp)/test", // Command substitution is not expanded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvOnly(tt.path)
			if got != tt.want {
				t.Errorf("expandEnvOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessPreset(t *testing.T) {
	// Set test environment variable
	t.Setenv("TEST_DIR", "/test/directory")

	tests := []struct {
		name      string
		preset    Preset
		wantPaths []string
		wantErr   bool
	}{
		{
			name: "preset with environment variables",
			preset: Preset{
				Allow: []AllowPath{
					{Path: "$HOME/.npm"},
					{Path: "${TEST_DIR}/data"},
					{Path: "/tmp"},
				},
				AllowKeychain: true,
			},
			wantPaths: []string{
				os.Getenv("HOME") + "/.npm",
				"/test/directory/data",
				"/tmp",
			},
			wantErr: false,
		},
		{
			name: "preset with command substitution not expanded",
			preset: Preset{
				Allow: []AllowPath{
					{Path: "$(echo /dynamic/path)"},
				},
			},
			wantPaths: []string{
				"$(echo /dynamic/path)", // Command substitution is not expanded
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processed, err := tt.preset.ProcessPreset()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessPreset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(processed.Allow) != len(tt.wantPaths) {
					t.Errorf(
						"ProcessPreset() returned %d paths, want %d",
						len(processed.Allow),
						len(tt.wantPaths),
					)
					return
				}

				for i, got := range processed.Allow {
					if got.Path != tt.wantPaths[i] {
						t.Errorf(
							"ProcessPreset() path[%d] = %v, want %v",
							i,
							got.Path,
							tt.wantPaths[i],
						)
					}
				}

				if processed.AllowKeychain != tt.preset.AllowKeychain {
					t.Errorf(
						"ProcessPreset() AllowKeychain = %v, want %v",
						processed.AllowKeychain,
						tt.preset.AllowKeychain,
					)
				}
			}
		})
	}
}

func TestPresetWithAllowKeychain(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `presets:
  test:
    allow:
      - "/tmp"
    allow-keychain: true`
	os.WriteFile(configPath, []byte(content), 0o644)

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	preset, ok := config.GetPreset("test")
	if !ok {
		t.Fatal("preset 'test' not found")
	}

	if !preset.AllowKeychain {
		t.Error("expected AllowKeychain to be true")
	}
}

func TestPresetWithAllowGit(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `presets:
  test:
    allow:
      - "/tmp"
    allow-git: true`
	os.WriteFile(configPath, []byte(content), 0o644)

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	preset, ok := config.GetPreset("test")
	if !ok {
		t.Fatal("preset 'test' not found")
	}

	if !preset.AllowGit {
		t.Error("expected AllowGit to be true")
	}
}

func TestProcessPresetWithAllowGit(t *testing.T) {
	// This test will only check the AllowGit flag is preserved
	// We can't easily test the git directory addition without a real git repo
	preset := Preset{
		Allow: []AllowPath{
			{Path: "$HOME/.npm"},
			{Path: "/tmp"},
		},
		AllowKeychain: true,
		AllowGit:      true,
	}

	processed, err := preset.ProcessPreset()
	if err != nil {
		t.Fatalf("ProcessPreset() error = %v", err)
	}

	// Check that AllowGit is preserved
	if processed.AllowGit != preset.AllowGit {
		t.Errorf("ProcessPreset() AllowGit = %v, want %v", processed.AllowGit, preset.AllowGit)
	}

	// Check that AllowKeychain is preserved
	if processed.AllowKeychain != preset.AllowKeychain {
		t.Errorf(
			"ProcessPreset() AllowKeychain = %v, want %v",
			processed.AllowKeychain,
			preset.AllowKeychain,
		)
	}

	// Check that paths are expanded
	expectedPaths := []string{
		os.Getenv("HOME") + "/.npm",
		"/tmp",
	}

	// The git directory might be added if we're in a git repo, but we can't control that in tests
	// So we check that at least the expected paths are present
	if len(processed.Allow) < len(expectedPaths) {
		t.Errorf(
			"ProcessPreset() returned %d paths, want at least %d",
			len(processed.Allow),
			len(expectedPaths),
		)
	}

	for i, expected := range expectedPaths {
		if processed.Allow[i].Path != expected {
			t.Errorf("ProcessPreset() path[%d] = %v, want %v", i, processed.Allow[i].Path, expected)
		}
	}
}

func TestGetAutoPresets(t *testing.T) {
	config := &Config{
		Presets: map[string]Preset{
			"claude-code": {Allow: []AllowPath{{Path: "/tmp"}}},
			"npm":         {Allow: []AllowPath{{Path: "~/.npm"}}},
			"python":      {Allow: []AllowPath{{Path: "~/.python"}}},
		},
		AutoPresets: []AutoPresetRule{
			{
				Command: "claude",
				Presets: []string{"claude-code"},
			},
			{
				CommandPattern: "^(npm|npx|yarn)$",
				Presets:        []string{"npm"},
			},
			{
				Command: "python",
				Presets: []string{"python"},
			},
			{
				CommandPattern: "^python[0-9]+$",
				Presets:        []string{"python"},
			},
		},
	}

	tests := []struct {
		name        string
		command     string
		wantPresets []string
		wantErr     bool
	}{
		{
			name:        "exact command match",
			command:     "claude",
			wantPresets: []string{"claude-code"},
			wantErr:     false,
		},
		{
			name:        "exact command match with path",
			command:     "/usr/bin/claude",
			wantPresets: []string{"claude-code"},
			wantErr:     false,
		},
		{
			name:        "regex pattern match npm",
			command:     "npm",
			wantPresets: []string{"npm"},
			wantErr:     false,
		},
		{
			name:        "regex pattern match npx",
			command:     "npx",
			wantPresets: []string{"npm"},
			wantErr:     false,
		},
		{
			name:        "regex pattern match yarn",
			command:     "/usr/local/bin/yarn",
			wantPresets: []string{"npm"},
			wantErr:     false,
		},
		{
			name:        "both exact and pattern match",
			command:     "python",
			wantPresets: []string{"python"},
			wantErr:     false,
		},
		{
			name:        "pattern match python3",
			command:     "python3",
			wantPresets: []string{"python"},
			wantErr:     false,
		},
		{
			name:        "no match",
			command:     "ls",
			wantPresets: []string{},
			wantErr:     false,
		},
		{
			name:        "no match with path",
			command:     "/bin/ls",
			wantPresets: []string{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presets, err := config.GetAutoPresets(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAutoPresets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(presets) != len(tt.wantPresets) {
					t.Errorf(
						"GetAutoPresets() returned %d presets, want %d",
						len(presets),
						len(tt.wantPresets),
					)
					return
				}

				for i, got := range presets {
					if got != tt.wantPresets[i] {
						t.Errorf(
							"GetAutoPresets() preset[%d] = %v, want %v",
							i,
							got,
							tt.wantPresets[i],
						)
					}
				}
			}
		})
	}
}

func TestGetAutoPresetsInvalidRegex(t *testing.T) {
	config := &Config{
		AutoPresets: []AutoPresetRule{
			{
				CommandPattern: "[invalid regex",
				Presets:        []string{"test"},
			},
		},
	}

	_, err := config.GetAutoPresets("test")
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestLoadConfigWithAutoPresets(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `presets:
  claude-code:
    allow:
      - "/tmp"
  npm:
    allow:
      - "~/.npm"

auto-presets:
  - command: claude
    presets:
      - claude-code
  - command-pattern: ^(npm|npx)$
    presets:
      - npm`
	os.WriteFile(configPath, []byte(content), 0o644)

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// Check presets loaded correctly
	if len(config.Presets) != 2 {
		t.Errorf("expected 2 presets, got %d", len(config.Presets))
	}

	// Check auto-presets loaded correctly
	if len(config.AutoPresets) != 2 {
		t.Errorf("expected 2 auto-preset rules, got %d", len(config.AutoPresets))
	}

	// Check first auto-preset rule
	if config.AutoPresets[0].Command != "claude" {
		t.Errorf(
			"expected first rule command to be 'claude', got %s",
			config.AutoPresets[0].Command,
		)
	}
	if len(config.AutoPresets[0].Presets) != 1 ||
		config.AutoPresets[0].Presets[0] != "claude-code" {
		t.Errorf("unexpected presets for first rule: %v", config.AutoPresets[0].Presets)
	}

	// Check second auto-preset rule
	if config.AutoPresets[1].CommandPattern != "^(npm|npx)$" {
		t.Errorf(
			"expected second rule pattern to be '^(npm|npx)$', got %s",
			config.AutoPresets[1].CommandPattern,
		)
	}
}

func TestAllowPathUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		want     AllowPath
		wantErr  bool
		errMatch string
	}{
		{
			name: "string format",
			yaml: `"/tmp/test"`,
			want: AllowPath{
				Path:         "/tmp/test",
				EvalSymLinks: false,
			},
			wantErr: false,
		},
		{
			name: "object format with eval-symlinks false",
			yaml: `path: "/tmp/test"
eval-symlinks: false`,
			want: AllowPath{
				Path:         "/tmp/test",
				EvalSymLinks: false,
			},
			wantErr: false,
		},
		{
			name: "object format with eval-symlinks true",
			yaml: `path: "/tmp/test"
eval-symlinks: true`,
			want: AllowPath{
				Path:         "/tmp/test",
				EvalSymLinks: true,
			},
			wantErr: false,
		},
		{
			name: "object format without eval-symlinks",
			yaml: `path: "/tmp/test"`,
			want: AllowPath{
				Path:         "/tmp/test",
				EvalSymLinks: false,
			},
			wantErr: false,
		},
		{
			name:     "invalid type - number",
			yaml:     `123`,
			wantErr:  true,
			errMatch: "unsupported type",
		},
		{
			name:     "invalid type - array",
			yaml:     `["/tmp", "/var"]`,
			wantErr:  true,
			errMatch: "unsupported type",
		},
		{
			name:     "invalid yaml",
			yaml:     `{path: "/tmp", eval-symlinks: [invalid}`,
			wantErr:  true,
			errMatch: "',' or ']' must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ap AllowPath
			err := yaml.Unmarshal([]byte(tt.yaml), &ap)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMatch != "" {
				if !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf(
						"UnmarshalYAML() error = %v, want error containing %q",
						err,
						tt.errMatch,
					)
				}
				return
			}

			if !tt.wantErr {
				if ap.Path != tt.want.Path {
					t.Errorf("UnmarshalYAML() Path = %v, want %v", ap.Path, tt.want.Path)
				}
				if ap.EvalSymLinks != tt.want.EvalSymLinks {
					t.Errorf(
						"UnmarshalYAML() EvalSymLinks = %v, want %v",
						ap.EvalSymLinks,
						tt.want.EvalSymLinks,
					)
				}
			}
		})
	}
}

func TestLoadConfigWithAllowPathFormats(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `presets:
  test:
    allow:
      - "/tmp"
      - path: "/var"
        eval-symlinks: false
      - path: "/home/user"
        eval-symlinks: true`
	os.WriteFile(configPath, []byte(content), 0o644)

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	preset, ok := config.GetPreset("test")
	if !ok {
		t.Fatal("preset 'test' not found")
	}

	if len(preset.Allow) != 3 {
		t.Fatalf("expected 3 allow paths, got %d", len(preset.Allow))
	}

	// Check first path (string format)
	if preset.Allow[0].Path != "/tmp" || preset.Allow[0].EvalSymLinks != false {
		t.Errorf("first path = %+v, want {Path: /tmp, EvalSymLinks: false}", preset.Allow[0])
	}

	// Check second path (object format, eval-symlinks: false)
	if preset.Allow[1].Path != "/var" || preset.Allow[1].EvalSymLinks != false {
		t.Errorf("second path = %+v, want {Path: /var, EvalSymLinks: false}", preset.Allow[1])
	}

	// Check third path (object format, eval-symlinks: true)
	if preset.Allow[2].Path != "/home/user" || preset.Allow[2].EvalSymLinks != true {
		t.Errorf("third path = %+v, want {Path: /home/user, EvalSymLinks: true}", preset.Allow[2])
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `defaults:
  presets:
    - "builtin:secrets-deny"
    - "my-preset"

presets:
  my-preset:
    allow:
      - "/tmp"`
	os.WriteFile(configPath, []byte(content), 0o644)

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// Check defaults loaded correctly
	if len(config.Defaults.Presets) != 2 {
		t.Errorf("expected 2 default presets, got %d", len(config.Defaults.Presets))
	}

	if config.Defaults.Presets[0] != "builtin:secrets-deny" {
		t.Errorf("expected first default preset to be 'builtin:secrets-deny', got %s", config.Defaults.Presets[0])
	}

	if config.Defaults.Presets[1] != "my-preset" {
		t.Errorf("expected second default preset to be 'my-preset', got %s", config.Defaults.Presets[1])
	}

	// Check presets loaded correctly
	preset, ok := config.GetPreset("my-preset")
	if !ok {
		t.Fatal("preset 'my-preset' not found")
	}

	if len(preset.Allow) != 1 || preset.Allow[0].Path != "/tmp" {
		t.Errorf("unexpected preset allow paths: %v", preset.Allow)
	}
}

func TestLoadConfigWithEmptyDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `presets:
  test:
    allow:
      - "/tmp"`
	os.WriteFile(configPath, []byte(content), 0o644)

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// Check defaults is empty when not specified
	if len(config.Defaults.Presets) != 0 {
		t.Errorf("expected 0 default presets, got %d", len(config.Defaults.Presets))
	}
}

func TestMergePresetsSkipDefaults(t *testing.T) {
	tests := []struct {
		name        string
		dst         Preset
		src         Preset
		wantSkipDef bool
	}{
		{
			name:        "both false",
			dst:         Preset{SkipDefaults: false},
			src:         Preset{SkipDefaults: false},
			wantSkipDef: false,
		},
		{
			name:        "dst true, src false",
			dst:         Preset{SkipDefaults: true},
			src:         Preset{SkipDefaults: false},
			wantSkipDef: true,
		},
		{
			name:        "dst false, src true",
			dst:         Preset{SkipDefaults: false},
			src:         Preset{SkipDefaults: true},
			wantSkipDef: true,
		},
		{
			name:        "both true",
			dst:         Preset{SkipDefaults: true},
			src:         Preset{SkipDefaults: true},
			wantSkipDef: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergePresets(&tt.dst, &tt.src)
			if tt.dst.SkipDefaults != tt.wantSkipDef {
				t.Errorf("mergePresets() SkipDefaults = %v, want %v", tt.dst.SkipDefaults, tt.wantSkipDef)
			}
		})
	}
}

func TestProcessPresetSkipDefaults(t *testing.T) {
	tests := []struct {
		name        string
		preset      Preset
		wantSkipDef bool
	}{
		{
			name:        "skip-defaults false",
			preset:      Preset{SkipDefaults: false, Allow: []AllowPath{{Path: "/tmp"}}},
			wantSkipDef: false,
		},
		{
			name:        "skip-defaults true",
			preset:      Preset{SkipDefaults: true, Allow: []AllowPath{{Path: "/tmp"}}},
			wantSkipDef: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processed, err := tt.preset.ProcessPreset()
			if err != nil {
				t.Fatalf("ProcessPreset() error = %v", err)
			}
			if processed.SkipDefaults != tt.wantSkipDef {
				t.Errorf("ProcessPreset() SkipDefaults = %v, want %v", processed.SkipDefaults, tt.wantSkipDef)
			}
		})
	}
}

func TestLoadConfigWithSkipDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	content := `defaults:
  presets:
    - "builtin:secrets-deny"

presets:
  regular:
    allow:
      - "/tmp"
  skip-defaults-preset:
    skip-defaults: true
    allow:
      - "/var"`
	os.WriteFile(configPath, []byte(content), 0o644)

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	regularPreset, ok := config.GetPreset("regular")
	if !ok {
		t.Fatal("preset 'regular' not found")
	}
	if regularPreset.SkipDefaults {
		t.Error("expected regular preset SkipDefaults to be false")
	}

	skipPreset, ok := config.GetPreset("skip-defaults-preset")
	if !ok {
		t.Fatal("preset 'skip-defaults-preset' not found")
	}
	if !skipPreset.SkipDefaults {
		t.Error("expected skip-defaults-preset SkipDefaults to be true")
	}
}

func TestResolvePresetWithSkipDefaults(t *testing.T) {
	config := &Config{
		Presets: map[string]Preset{
			"base": {
				Allow: []AllowPath{{Path: "/base"}},
			},
			"child-no-skip": {
				Extends: []string{"base"},
				Allow:   []AllowPath{{Path: "/child"}},
			},
			"child-with-skip": {
				Extends:      []string{"base"},
				SkipDefaults: true,
				Allow:        []AllowPath{{Path: "/child"}},
			},
			"parent-with-skip": {
				SkipDefaults: true,
				Allow:        []AllowPath{{Path: "/parent"}},
			},
			"child-inherits-skip": {
				Extends: []string{"parent-with-skip"},
				Allow:   []AllowPath{{Path: "/child"}},
			},
		},
	}

	tests := []struct {
		name        string
		presetName  string
		wantSkipDef bool
	}{
		{
			name:        "base preset without skip-defaults",
			presetName:  "base",
			wantSkipDef: false,
		},
		{
			name:        "child without skip-defaults",
			presetName:  "child-no-skip",
			wantSkipDef: false,
		},
		{
			name:        "child with skip-defaults",
			presetName:  "child-with-skip",
			wantSkipDef: true,
		},
		{
			name:        "child inherits skip-defaults from parent",
			presetName:  "child-inherits-skip",
			wantSkipDef: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := config.ResolvePreset(tt.presetName, nil)
			if err != nil {
				t.Fatalf("ResolvePreset() error = %v", err)
			}
			if resolved.SkipDefaults != tt.wantSkipDef {
				t.Errorf("ResolvePreset() SkipDefaults = %v, want %v", resolved.SkipDefaults, tt.wantSkipDef)
			}
		})
	}
}

func TestProcessPresetWithSymlinkEvaluation(t *testing.T) {
	// Create a temporary directory with a symlink
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")
	symlinkPath := filepath.Join(tmpDir, "symlink")

	// Create target directory
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}

	// Create symlink
	if err := os.Symlink(targetDir, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Resolve the expected paths fully (to handle macOS /var -> /private/var)
	resolvedTargetDir, _ := filepath.EvalSymlinks(targetDir)

	tests := []struct {
		name      string
		preset    Preset
		wantPaths []string
	}{
		{
			name: "symlink evaluation disabled",
			preset: Preset{
				Allow: []AllowPath{
					{Path: symlinkPath, EvalSymLinks: false},
				},
			},
			wantPaths: []string{symlinkPath},
		},
		{
			name: "symlink evaluation enabled",
			preset: Preset{
				Allow: []AllowPath{
					{Path: symlinkPath, EvalSymLinks: true},
				},
			},
			wantPaths: []string{resolvedTargetDir},
		},
		{
			name: "mix of symlink and regular paths",
			preset: Preset{
				Allow: []AllowPath{
					{Path: "/tmp", EvalSymLinks: false},
					{Path: symlinkPath, EvalSymLinks: true},
					{Path: "/var", EvalSymLinks: false},
				},
			},
			wantPaths: []string{"/tmp", resolvedTargetDir, "/var"},
		},
		{
			name: "non-existent symlink falls back to original path",
			preset: Preset{
				Allow: []AllowPath{
					{Path: "/non/existent/symlink", EvalSymLinks: true},
				},
			},
			wantPaths: []string{"/non/existent/symlink"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processed, err := tt.preset.ProcessPreset()
			if err != nil {
				t.Fatalf("ProcessPreset() error = %v", err)
			}

			if len(processed.Allow) != len(tt.wantPaths) {
				t.Errorf(
					"ProcessPreset() returned %d paths, want %d",
					len(processed.Allow),
					len(tt.wantPaths),
				)
				return
			}

			for i, got := range processed.Allow {
				if got.Path != tt.wantPaths[i] {
					t.Errorf("ProcessPreset() path[%d] = %v, want %v", i, got.Path, tt.wantPaths[i])
				}
			}
		})
	}
}
