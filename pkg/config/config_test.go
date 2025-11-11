package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spideyz0r/fh/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Database.Path)
	assert.True(t, cfg.Deduplicate.Enabled)
	assert.Equal(t, "keep_all", cfg.Deduplicate.Strategy)
	assert.NotEmpty(t, cfg.Ignore.Patterns)
	assert.Contains(t, cfg.Ignore.Patterns, "^ls$")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  Default(),
			wantErr: false,
		},
		{
			name: "empty database path",
			config: &Config{
				Database: DatabaseConfig{Path: ""},
			},
			wantErr: true,
		},
		{
			name: "invalid dedup strategy",
			config: &Config{
				Database: DatabaseConfig{Path: "/tmp/test.db"},
				Deduplicate: DeduplicateConfig{
					Enabled:  true,
					Strategy: "invalid_strategy",
				},
			},
			wantErr: true,
		},
		{
			name: "valid keep_first strategy",
			config: &Config{
				Database: DatabaseConfig{Path: "/tmp/test.db"},
				Deduplicate: DeduplicateConfig{
					Enabled:  true,
					Strategy: "keep_first",
				},
			},
			wantErr: false,
		},
		{
			name: "valid keep_last strategy",
			config: &Config{
				Database: DatabaseConfig{Path: "/tmp/test.db"},
				Deduplicate: DeduplicateConfig{
					Enabled:  true,
					Strategy: "keep_last",
				},
			},
			wantErr: false,
		},
		{
			name: "dedup disabled with invalid strategy",
			config: &Config{
				Database: DatabaseConfig{Path: "/tmp/test.db"},
				Deduplicate: DeduplicateConfig{
					Enabled:  false,
					Strategy: "invalid",
				},
			},
			wantErr: false, // Should not validate strategy when disabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test loading non-existent file (should return defaults)
	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "keep_all", cfg.Deduplicate.Strategy)

	// Create a config file
	configYAML := `
database:
  path: /tmp/custom.db
deduplicate:
  enabled: true
  strategy: keep_first
ignore:
  patterns:
    - "^echo "
    - "^test$"
`
	err = os.WriteFile(configPath, []byte(configYAML), 0644)
	require.NoError(t, err)

	// Load the config file
	cfg, err = Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/custom.db", cfg.Database.Path)
	assert.True(t, cfg.Deduplicate.Enabled)
	assert.Equal(t, "keep_first", cfg.Deduplicate.Strategy)
	assert.Len(t, cfg.Ignore.Patterns, 2)
	assert.Contains(t, cfg.Ignore.Patterns, "^echo ")
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: :::"), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestLoad_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config with invalid strategy
	configYAML := `
database:
  path: /tmp/test.db
deduplicate:
  enabled: true
  strategy: invalid_strategy
`
	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid configuration")
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yaml")

	cfg := Default()
	cfg.Database.Path = "/custom/path.db"
	cfg.Deduplicate.Strategy = "keep_first"

	err := cfg.Save(configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Load it back and verify
	loaded, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "/custom/path.db", loaded.Database.Path)
	assert.Equal(t, "keep_first", loaded.Deduplicate.Strategy)
}

func TestGetDedupConfig(t *testing.T) {
	tests := []struct {
		name             string
		strategy         string
		enabled          bool
		expectedStrategy storage.DedupStrategy
		expectedEnabled  bool
	}{
		{
			name:             "keep_first",
			strategy:         "keep_first",
			enabled:          true,
			expectedStrategy: storage.KeepFirst,
			expectedEnabled:  true,
		},
		{
			name:             "keep_last",
			strategy:         "keep_last",
			enabled:          true,
			expectedStrategy: storage.KeepLast,
			expectedEnabled:  true,
		},
		{
			name:             "keep_all",
			strategy:         "keep_all",
			enabled:          true,
			expectedStrategy: storage.KeepAll,
			expectedEnabled:  true,
		},
		{
			name:             "disabled",
			strategy:         "keep_first",
			enabled:          false,
			expectedStrategy: storage.KeepFirst,
			expectedEnabled:  false,
		},
		{
			name:             "invalid defaults to keep_all",
			strategy:         "invalid",
			enabled:          true,
			expectedStrategy: storage.KeepAll,
			expectedEnabled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Database: DatabaseConfig{Path: "/tmp/test.db"},
				Deduplicate: DeduplicateConfig{
					Enabled:  tt.enabled,
					Strategy: tt.strategy,
				},
			}

			dedupCfg := cfg.GetDedupConfig()
			assert.Equal(t, tt.expectedEnabled, dedupCfg.Enabled)
			assert.Equal(t, tt.expectedStrategy, dedupCfg.Strategy)
		})
	}
}

func TestGetDatabasePath(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{Path: "/custom/db/path.db"},
	}

	assert.Equal(t, "/custom/db/path.db", cfg.GetDatabasePath())
}

func TestLoadDefault(t *testing.T) {
	// This test will use the actual home directory
	// It should not fail even if config doesn't exist
	cfg, err := LoadDefault()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should have default values
	assert.True(t, cfg.Deduplicate.Enabled)
	assert.NotEmpty(t, cfg.Database.Path)
}

func TestClearCache(t *testing.T) {
	// Load a config to populate cache
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	cfg := Default()
	err := cfg.Save(configPath)
	require.NoError(t, err)
	
	// Load to populate cache
	_, err = Load(configPath)
	require.NoError(t, err)
	
	// Clear cache
	ClearCache()
	
	// Should be able to load again (cache cleared)
	cfg2, err := Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg2)
}

func TestSave_InvalidPath(t *testing.T) {
	cfg := Default()
	
	// Try to save to an invalid path (root directory without permission)
	err := cfg.Save("/invalid/path/that/cannot/be/created/config.yaml")
	assert.Error(t, err)
}

func TestLoad_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	cfg := Default()
	cfg.Database.Path = "/cache/test.db"
	err := cfg.Save(configPath)
	require.NoError(t, err)
	
	// First load - cache miss
	cfg1, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "/cache/test.db", cfg1.Database.Path)
	
	// Second load - should hit cache
	cfg2, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "/cache/test.db", cfg2.Database.Path)
	
	// Should be the same instance from cache
	assert.Equal(t, cfg1, cfg2)
}

func TestValidate_EdgeCases(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cfg := &Config{}
		err := cfg.Validate()
		assert.Error(t, err)
	})
	
	t.Run("valid config with empty ignore patterns", func(t *testing.T) {
		cfg := &Config{
			Database: DatabaseConfig{Path: "/tmp/test.db"},
			Deduplicate: DeduplicateConfig{
				Enabled:  false,
				Strategy: "keep_all",
			},
			Ignore: IgnoreConfig{Patterns: []string{}},
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})
}

func TestGetDedupConfig_AllStrategies(t *testing.T) {
	strategies := []struct {
		input    string
		expected storage.DedupStrategy
	}{
		{"keep_first", storage.KeepFirst},
		{"keep_last", storage.KeepLast},
		{"keep_all", storage.KeepAll},
		{"KEEP_FIRST", storage.KeepAll}, // Invalid, defaults to KeepAll
		{"", storage.KeepAll},           // Empty, defaults to KeepAll
	}
	
	for _, s := range strategies {
		t.Run(s.input, func(t *testing.T) {
			cfg := &Config{
				Database: DatabaseConfig{Path: "/tmp/test.db"},
				Deduplicate: DeduplicateConfig{
					Enabled:  true,
					Strategy: s.input,
				},
			}
			dedupCfg := cfg.GetDedupConfig()
			assert.Equal(t, s.expected, dedupCfg.Strategy)
		})
	}
}
