package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestMultiplePresetsWithDuplicatePaths(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Create test presets with overlapping paths
	content := `presets:
  preset1:
    allow:
      - "/tmp"
      - "/var/log"
      - "$HOME/.cache"
  preset2:
    allow:
      - "/tmp"
      - "/usr/local"
      - "$HOME/.cache"
  preset3:
    allow:
      - "/var/log"
      - "/usr/local"
      - "/tmp"`
	os.WriteFile(configPath, []byte(content), 0o644)

	// Create a mock flags struct
	flags := flags{
		presets:    []string{"preset1", "preset2", "preset3"},
		allowPaths: []string{"/tmp", "/custom/path"},
	}

	// Load config
	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// Process presets
	allowedPaths := flags.allowPaths
	pathSet := make(map[string]struct{})
	var uniquePaths []string

	for _, presetName := range flags.presets {
		preset, ok := config.GetPreset(presetName)
		if !ok {
			t.Fatalf("preset '%s' not found", presetName)
		}

		processedPreset, err := preset.ProcessPreset()
		if err != nil {
			t.Fatalf("error processing preset '%s': %v", presetName, err)
		}

		for _, path := range processedPreset.Allow {
			absPath, err := filepath.Abs(path.Path)
			if err != nil {
				absPath = path.Path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path.Path)
			}
		}
	}

	// Add command-line paths
	for _, path := range allowedPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		if _, exists := pathSet[absPath]; !exists {
			pathSet[absPath] = struct{}{}
			uniquePaths = append(uniquePaths, path)
		}
	}

	// Check that duplicates were removed
	// We should have: /tmp, /var/log, $HOME/.cache (expanded), /usr/local, /custom/path
	// Total: 5 unique paths
	if len(uniquePaths) != 5 {
		t.Errorf("expected 5 unique paths, got %d: %v", len(uniquePaths), uniquePaths)
	}

	// Verify specific paths are present
	expectedPaths := map[string]bool{
		"/tmp":         false,
		"/var/log":     false,
		"/usr/local":   false,
		"/custom/path": false,
	}

	homeCache := os.ExpandEnv("$HOME/.cache")
	expectedPaths[homeCache] = false

	for _, path := range uniquePaths {
		if _, ok := expectedPaths[path]; ok {
			expectedPaths[path] = true
		}
	}

	for path, found := range expectedPaths {
		if !found {
			t.Errorf("expected path %s not found in unique paths", path)
		}
	}
}

func TestPresetPathsWithRelativeAndAbsolute(t *testing.T) {
	// Create a temporary directory and change to it
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	configPath := filepath.Join(tmpDir, "test.yaml")

	// Create test presets with relative and absolute paths that resolve to the same location
	content := `presets:
  preset1:
    allow:
      - "./data"
      - "logs"
  preset2:
    allow:
      - "` + filepath.Join(tmpDir, "data") + `"
      - "` + filepath.Join(tmpDir, "logs") + `"`
	os.WriteFile(configPath, []byte(content), 0o644)

	// Create directories
	os.Mkdir("data", 0o755)
	os.Mkdir("logs", 0o755)

	// Create a mock flags struct
	flags := flags{
		presets:    []string{"preset1", "preset2"},
		allowPaths: []string{},
	}

	// Load config
	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// Process presets
	pathSet := make(map[string]struct{})
	var uniquePaths []string

	for _, presetName := range flags.presets {
		preset, ok := config.GetPreset(presetName)
		if !ok {
			t.Fatalf("preset '%s' not found", presetName)
		}

		processedPreset, err := preset.ProcessPreset()
		if err != nil {
			t.Fatalf("error processing preset '%s': %v", presetName, err)
		}

		for _, path := range processedPreset.Allow {
			absPath, err := filepath.Abs(path.Path)
			if err != nil {
				absPath = path.Path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path.Path)
			}
		}
	}

	// Should have only 2 unique paths (data and logs)
	if len(uniquePaths) != 2 {
		t.Errorf("expected 2 unique paths, got %d: %v", len(uniquePaths), uniquePaths)
	}
}

func TestEnvironmentVariableExpansionDuplicates(t *testing.T) {
	// Set a test environment variable
	testDir := "/test/directory"
	t.Setenv("TEST_DIR", testDir)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Create test presets with environment variables that expand to the same path
	content := `presets:
  preset1:
    allow:
      - "$TEST_DIR/data"
      - "${TEST_DIR}/logs"
  preset2:
    allow:
      - "` + testDir + `/data"
      - "` + testDir + `/logs"`
	os.WriteFile(configPath, []byte(content), 0o644)

	// Create a mock flags struct
	flags := flags{
		presets:    []string{"preset1", "preset2"},
		allowPaths: []string{},
	}

	// Load config
	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// Process presets
	pathSet := make(map[string]struct{})
	var uniquePaths []string

	for _, presetName := range flags.presets {
		preset, ok := config.GetPreset(presetName)
		if !ok {
			t.Fatalf("preset '%s' not found", presetName)
		}

		processedPreset, err := preset.ProcessPreset()
		if err != nil {
			t.Fatalf("error processing preset '%s': %v", presetName, err)
		}

		for _, path := range processedPreset.Allow {
			absPath, err := filepath.Abs(path.Path)
			if err != nil {
				absPath = path.Path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path.Path)
			}
		}
	}

	// Should have only 2 unique paths after expansion
	if len(uniquePaths) != 2 {
		t.Errorf("expected 2 unique paths, got %d: %v", len(uniquePaths), uniquePaths)
	}

	// Sort for consistent comparison
	sort.Strings(uniquePaths)
	expectedPaths := []string{
		testDir + "/data",
		testDir + "/logs",
	}
	sort.Strings(expectedPaths)

	// Check that the paths match expected values
	for i, path := range uniquePaths {
		if path != expectedPaths[i] {
			t.Errorf("path[%d] = %v, want %v", i, path, expectedPaths[i])
		}
	}
}

func TestPresetOrderPreservation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Create test presets
	content := `presets:
  preset1:
    allow:
      - "/first"
      - "/second"
  preset2:
    allow:
      - "/third"
      - "/first"
      - "/fourth"`
	os.WriteFile(configPath, []byte(content), 0o644)

	// Create a mock flags struct
	flags := flags{
		presets:    []string{"preset1", "preset2"},
		allowPaths: []string{"/fifth", "/first"},
	}

	// Load config
	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// Process presets
	allowedPaths := flags.allowPaths
	pathSet := make(map[string]struct{})
	var uniquePaths []string

	for _, presetName := range flags.presets {
		preset, ok := config.GetPreset(presetName)
		if !ok {
			t.Fatalf("preset '%s' not found", presetName)
		}

		processedPreset, err := preset.ProcessPreset()
		if err != nil {
			t.Fatalf("error processing preset '%s': %v", presetName, err)
		}

		for _, path := range processedPreset.Allow {
			absPath, err := filepath.Abs(path.Path)
			if err != nil {
				absPath = path.Path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path.Path)
			}
		}
	}

	// Add command-line paths
	for _, path := range allowedPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		if _, exists := pathSet[absPath]; !exists {
			pathSet[absPath] = struct{}{}
			uniquePaths = append(uniquePaths, path)
		}
	}

	// Check order: should be /first (from preset1), /second, /third, /fourth, /fifth
	expectedOrder := []string{"/first", "/second", "/third", "/fourth", "/fifth"}
	if !reflect.DeepEqual(uniquePaths, expectedOrder) {
		t.Errorf("path order incorrect.\nGot:  %v\nWant: %v", uniquePaths, expectedOrder)
	}
}

func TestAutoPresetsWithCommandLinePresets(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	// Create test config with presets and auto-presets
	content := `presets:
  git-preset:
    allow:
      - "/usr/bin/git"
      - "$HOME/.gitconfig"
  extra-preset:
    allow:
      - "/opt/tools"
      - "/var/cache"
auto-presets:
  - command: git
    presets:
      - git-preset`
	os.WriteFile(configPath, []byte(content), 0o644)

	// Load config
	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	// Simulate having command-line presets already set
	flags := flags{
		presets:    []string{"extra-preset"},
		allowPaths: []string{"/custom/path"},
	}

	// Simulate auto-preset detection (normally done in main())
	autoPresets, err := config.GetAutoPresets("git")
	if err != nil {
		t.Fatalf("GetAutoPresets() error = %v", err)
	}

	// Merge auto-detected presets with command-line presets
	// Command-line presets come first to maintain priority
	flags.presets = append(flags.presets, autoPresets...)

	// Process all presets
	pathSet := make(map[string]struct{})
	var uniquePaths []string

	for _, presetName := range flags.presets {
		preset, ok := config.GetPreset(presetName)
		if !ok {
			t.Fatalf("preset '%s' not found", presetName)
		}

		processedPreset, err := preset.ProcessPreset()
		if err != nil {
			t.Fatalf("error processing preset '%s': %v", presetName, err)
		}

		for _, path := range processedPreset.Allow {
			absPath, err := filepath.Abs(path.Path)
			if err != nil {
				absPath = path.Path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path.Path)
			}
		}
	}

	// Add command-line paths
	for _, path := range flags.allowPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		if _, exists := pathSet[absPath]; !exists {
			pathSet[absPath] = struct{}{}
			uniquePaths = append(uniquePaths, path)
		}
	}

	// Verify we have paths from both command-line preset and auto-preset
	expectedPaths := map[string]bool{
		"/opt/tools":   false,
		"/var/cache":   false,
		"/usr/bin/git": false,
		"/custom/path": false,
	}

	gitconfig := os.ExpandEnv("$HOME/.gitconfig")
	expectedPaths[gitconfig] = false

	for _, path := range uniquePaths {
		if _, ok := expectedPaths[path]; ok {
			expectedPaths[path] = true
		}
	}

	for path, found := range expectedPaths {
		if !found {
			t.Errorf("expected path %s not found in unique paths", path)
		}
	}

	// Verify order: command-line preset paths should come before auto-preset paths
	if len(uniquePaths) < 4 {
		t.Fatalf("expected at least 4 paths, got %d", len(uniquePaths))
	}

	// First two should be from extra-preset (command-line preset)
	if uniquePaths[0] != "/opt/tools" || uniquePaths[1] != "/var/cache" {
		t.Errorf("command-line preset paths should come first, got: %v", uniquePaths[:2])
	}
}

func TestSortedPaths(t *testing.T) {
	tests := []struct {
		name  string
		input []AllowPath
		want  []string
	}{
		{
			name:  "empty",
			input: []AllowPath{},
			want:  []string{},
		},
		{
			name: "already sorted",
			input: []AllowPath{
				{Path: "/a"},
				{Path: "/b"},
				{Path: "/c"},
			},
			want: []string{"/a", "/b", "/c"},
		},
		{
			name: "reverse order",
			input: []AllowPath{
				{Path: "/z"},
				{Path: "/m"},
				{Path: "/a"},
			},
			want: []string{"/a", "/m", "/z"},
		},
		{
			name: "mixed paths",
			input: []AllowPath{
				{Path: "$HOME/.ssh"},
				{Path: "/var/log"},
				{Path: "$HOME/.aws"},
				{Path: "/etc"},
				{Path: "$HOME/.config/gcloud"},
			},
			want: []string{"$HOME/.aws", "$HOME/.config/gcloud", "$HOME/.ssh", "/etc", "/var/log"},
		},
		{
			name: "single element",
			input: []AllowPath{
				{Path: "/only"},
			},
			want: []string{"/only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedPaths(tt.input)

			if len(got) != len(tt.want) {
				t.Fatalf("sortedPaths() returned %d elements, want %d", len(got), len(tt.want))
			}

			for i, p := range got {
				if p.Path != tt.want[i] {
					t.Errorf("sortedPaths()[%d] = %q, want %q", i, p.Path, tt.want[i])
				}
			}
		})
	}
}

func TestSortedPathsDoesNotMutateOriginal(t *testing.T) {
	original := []AllowPath{
		{Path: "/z"},
		{Path: "/a"},
		{Path: "/m"},
	}

	originalCopy := make([]AllowPath, len(original))
	copy(originalCopy, original)

	_ = sortedPaths(original)

	for i, p := range original {
		if p.Path != originalCopy[i].Path {
			t.Errorf("sortedPaths() mutated original: index %d changed from %q to %q",
				i, originalCopy[i].Path, p.Path)
		}
	}
}

func TestPrintPresetText(t *testing.T) {
	tests := []struct {
		name           string
		presetName     string
		preset         *Preset
		extends        []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:       "preset with inheritance chain",
			presetName: "test-preset",
			preset: &Preset{
				AllowGit: true,
				Strict:   true,
				Allow:    []AllowPath{{Path: "/tmp"}},
			},
			extends: []string{"base-preset", "other-preset"},
			wantContains: []string{
				"Preset: test-preset",
				"Extends: base-preset → other-preset",
				"allow-git: true",
				"strict: true",
				"/tmp",
			},
		},
		{
			name:       "preset without inheritance",
			presetName: "simple-preset",
			preset: &Preset{
				Allow: []AllowPath{{Path: "/var/log"}},
			},
			extends: nil,
			wantContains: []string{
				"Preset: simple-preset",
				"/var/log",
			},
			wantNotContain: []string{
				"Extends:",
			},
		},
		{
			name:       "raw preset with extends field",
			presetName: "child-preset",
			preset: &Preset{
				Extends:  []string{"builtin:strict-base", "builtin:secrets-deny"},
				AllowGit: true,
				Allow:    []AllowPath{{Path: "."}},
			},
			extends: nil,
			wantContains: []string{
				"Preset: child-preset",
				"extends:",
				"builtin:strict-base",
				"builtin:secrets-deny",
				"allow-git: true",
			},
			wantNotContain: []string{
				"Extends:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printPresetText(tt.presetName, tt.preset, tt.extends)
			})

			for _, want := range tt.wantContains {
				if !containsString(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if containsString(output, notWant) {
					t.Errorf("output should not contain %q\nGot:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestPrintPresetYAML(t *testing.T) {
	tests := []struct {
		name           string
		presetName     string
		preset         *Preset
		extends        []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:       "resolved preset with inheritance comment",
			presetName: "builtin:secure",
			preset: &Preset{
				AllowGit: true,
				Strict:   true,
				Allow:    []AllowPath{{Path: "."}},
			},
			extends: []string{"builtin:strict-base", "builtin:secrets-deny"},
			wantContains: []string{
				"# Extends: builtin:strict-base → builtin:secrets-deny",
				"presets:",
				"  secure:",
				"    allow-git: true",
				"    strict: true",
			},
		},
		{
			name:       "raw preset with extends field in YAML",
			presetName: "my-preset",
			preset: &Preset{
				Extends:       []string{"builtin:secure"},
				AllowKeychain: true,
				Allow:         []AllowPath{{Path: "/tmp"}},
			},
			extends: nil,
			wantContains: []string{
				"presets:",
				"  my-preset:",
				"    extends:",
				`      - "builtin:secure"`,
				"    allow-keychain: true",
			},
			wantNotContain: []string{
				"# Extends:",
			},
		},
		{
			name:       "preset without inheritance",
			presetName: "builtin:npm",
			preset: &Preset{
				Allow: []AllowPath{
					{Path: "."},
					{Path: "$HOME/.npm"},
				},
			},
			extends: nil,
			wantContains: []string{
				"presets:",
				"  npm:",
				"    allow:",
			},
			wantNotContain: []string{
				"# Extends:",
				"extends:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printPresetYAML(tt.presetName, tt.preset, tt.extends)
			})

			for _, want := range tt.wantContains {
				if !containsString(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if containsString(output, notWant) {
					t.Errorf("output should not contain %q\nGot:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestPrintPresetFormats(t *testing.T) {
	preset := &Preset{
		Extends:  []string{"parent-preset"},
		AllowGit: true,
		Allow:    []AllowPath{{Path: "/tmp"}},
	}
	extends := []string{"parent-preset"}

	t.Run("text format", func(t *testing.T) {
		output := captureOutput(func() {
			printPreset("test", preset, "text", extends)
		})
		if !containsString(output, "Preset: test") {
			t.Errorf("text format should contain 'Preset: test', got:\n%s", output)
		}
		if !containsString(output, "========") {
			t.Errorf("text format should contain separator line, got:\n%s", output)
		}
	})

	t.Run("yaml format", func(t *testing.T) {
		output := captureOutput(func() {
			printPreset("test", preset, "yaml", extends)
		})
		if !containsString(output, "presets:") {
			t.Errorf("yaml format should contain 'presets:', got:\n%s", output)
		}
		if !containsString(output, "# Extends:") {
			t.Errorf("yaml format should contain inheritance comment, got:\n%s", output)
		}
	})

	t.Run("raw format uses yaml output", func(t *testing.T) {
		rawPreset := &Preset{
			Extends:  []string{"parent-preset"},
			AllowGit: true,
			Allow:    []AllowPath{{Path: "/tmp"}},
		}
		output := captureOutput(func() {
			printPreset("test", rawPreset, "yaml", nil)
		})
		if !containsString(output, "presets:") {
			t.Errorf("raw format should use yaml output, got:\n%s", output)
		}
		if !containsString(output, "extends:") {
			t.Errorf("raw format should show extends field, got:\n%s", output)
		}
		if containsString(output, "# Extends:") {
			t.Errorf("raw format should not have inheritance comment, got:\n%s", output)
		}
	})
}

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
