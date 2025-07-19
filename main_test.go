package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path)
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
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path)
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
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path)
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
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path)
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
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			if _, exists := pathSet[absPath]; !exists {
				pathSet[absPath] = struct{}{}
				uniquePaths = append(uniquePaths, path)
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
