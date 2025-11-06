package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spideyz0r/fh/pkg/storage"
	"gopkg.in/yaml.v3"
)

// Cache for config to avoid repeated file reads
var (
	cacheMutex   sync.RWMutex
	cachedConfig *Config
	cachedPath   string
	cachedModTime time.Time
)

// Config holds the application configuration
type Config struct {
	Database    DatabaseConfig    `yaml:"database"`
	Deduplicate DeduplicateConfig `yaml:"deduplicate"`
	Ignore      IgnoreConfig      `yaml:"ignore"`
	Search      SearchConfig      `yaml:"search"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Path string `yaml:"path"` // Path to SQLite database file
}

// DeduplicateConfig holds deduplication settings
type DeduplicateConfig struct {
	Enabled  bool   `yaml:"enabled"`  // Enable deduplication
	Strategy string `yaml:"strategy"` // keep_first, keep_last, keep_all
}

// IgnoreConfig holds patterns for commands to ignore
type IgnoreConfig struct {
	Patterns []string `yaml:"patterns"` // Patterns to ignore (e.g., "^ls$", "^cd ")
}

// SearchConfig holds search-related configuration
type SearchConfig struct {
	Limit int `yaml:"limit"` // Max number of entries to load for FZF (0 = unlimited)
}

// Default returns the default configuration
func Default() *Config {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".fh", "history.db")

	return &Config{
		Database: DatabaseConfig{
			Path: dbPath,
		},
		Deduplicate: DeduplicateConfig{
			Enabled:  true,
			Strategy: "keep_all", // Default to keep_all for AI context
		},
		Ignore: IgnoreConfig{
			Patterns: []string{
				// Common commands to ignore
				"^ls$",
				"^ls ",
				"^cd$",
				"^cd ",
				"^pwd$",
				"^exit$",
				"^clear$",
			},
		},
		Search: SearchConfig{
			Limit: 1000, // Default: load 1000 most recent entries (0 = unlimited)
		},
	}
}

// Load loads configuration from file, falling back to defaults
// Uses a cache to avoid repeated file reads if the file hasn't changed
func Load(path string) (*Config, error) {
	// Check cache first
	cacheMutex.RLock()
	if cachedConfig != nil && cachedPath == path {
		// Check if file has been modified
		if stat, err := os.Stat(path); err == nil {
			if stat.ModTime().Equal(cachedModTime) {
				// Cache is still valid
				defer cacheMutex.RUnlock()
				return cachedConfig, nil
			}
		}
	}
	cacheMutex.RUnlock()

	// Cache miss or file changed - load from disk
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Start with defaults
	cfg := Default()

	// If file doesn't exist, cache and return defaults
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		cachedConfig = cfg
		cachedPath = path
		cachedModTime = time.Time{} // Zero time for non-existent file
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Update cache
	cachedConfig = cfg
	cachedPath = path
	cachedModTime = stat.ModTime()

	return cfg, nil
}

// LoadDefault loads configuration from default path (~/.fh/config.yaml)
func LoadDefault() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".fh", "config.yaml")
	return Load(configPath)
}

// ClearCache clears the configuration cache, forcing a reload on next Load()
func ClearCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	cachedConfig = nil
	cachedPath = ""
	cachedModTime = time.Time{}
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate database path
	if c.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// Validate dedup strategy
	validStrategies := map[string]bool{
		"keep_first": true,
		"keep_last":  true,
		"keep_all":   true,
	}

	if c.Deduplicate.Enabled && !validStrategies[c.Deduplicate.Strategy] {
		return fmt.Errorf("invalid dedup strategy: %s (must be keep_first, keep_last, or keep_all)", c.Deduplicate.Strategy)
	}

	return nil
}

// GetDedupConfig converts config to storage.DedupConfig
func (c *Config) GetDedupConfig() storage.DedupConfig {
	var strategy storage.DedupStrategy

	switch c.Deduplicate.Strategy {
	case "keep_first":
		strategy = storage.KeepFirst
	case "keep_last":
		strategy = storage.KeepLast
	case "keep_all":
		strategy = storage.KeepAll
	default:
		strategy = storage.KeepAll // Safe default
	}

	return storage.DedupConfig{
		Enabled:  c.Deduplicate.Enabled,
		Strategy: strategy,
	}
}

// GetDatabasePath returns the configured database path
func (c *Config) GetDatabasePath() string {
	return c.Database.Path
}
