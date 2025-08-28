package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"cursor-sync/internal/cursor"
)

// Config represents the application configuration
type Config struct {
	Repository Repository `yaml:"repository" mapstructure:"repository"`
	Sync       Sync       `yaml:"sync" mapstructure:"sync"`
	Cursor     Cursor     `yaml:"cursor" mapstructure:"cursor"`
	Logging    Logging    `yaml:"logging" mapstructure:"logging"`
}

// Repository configuration
type Repository struct {
	URL       string `yaml:"url" mapstructure:"url"`
	LocalPath string `yaml:"local_path" mapstructure:"local_path"`
	Branch    string `yaml:"branch" mapstructure:"branch"`
}

// Sync configuration
type Sync struct {
	PullInterval       time.Duration `yaml:"pull_interval" mapstructure:"pull_interval"`
	PushInterval       time.Duration `yaml:"push_interval" mapstructure:"push_interval"`
	DebounceTime       time.Duration `yaml:"debounce_time" mapstructure:"debounce_time"`
	WatchEnabled       bool          `yaml:"watch_enabled" mapstructure:"watch_enabled"`
	ConflictResolve    string        `yaml:"conflict_resolve" mapstructure:"conflict_resolve"`
	HashThrottleDelay  time.Duration `yaml:"hash_throttle_delay" mapstructure:"hash_throttle_delay"`
	HashPollingTimeout time.Duration `yaml:"hash_polling_timeout" mapstructure:"hash_polling_timeout"`
}

// Cursor configuration
type Cursor struct {
	ConfigPath   string   `yaml:"config_path" mapstructure:"config_path"`
	ExcludePaths []string `yaml:"exclude_paths" mapstructure:"exclude_paths"`
	IncludePaths []string `yaml:"include_paths" mapstructure:"include_paths"`
}

// Logging configuration
type Logging struct {
	Level    string `yaml:"level" mapstructure:"level"`
	LogDir   string `yaml:"log_dir" mapstructure:"log_dir"`
	MaxSize  int    `yaml:"max_size" mapstructure:"max_size"`
	MaxDays  int    `yaml:"max_days" mapstructure:"max_days"`
	Compress bool   `yaml:"compress" mapstructure:"compress"`
}

// Load loads the configuration from file and environment variables
func Load() (*Config, error) {
	var cfg Config

	// Set defaults from example config first
	setDefaults()

	// Set up viper to read from user config file
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	userConfigPath := filepath.Join(home, ".cursor-sync", "config.yaml")
	viper.SetConfigFile(userConfigPath)

	// Read the user config file (this will override defaults)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the configuration
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Parse time durations manually since viper doesn't handle them well
	if err := parseTimeDurations(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse time durations: %w", err)
	}

	// Expand environment variables and home directory
	if err := expandPaths(&cfg); err != nil {
		return nil, fmt.Errorf("failed to expand paths: %w", err)
	}

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate Cursor installation
	if err := validateCursorInstallation(&cfg); err != nil {
		cursor.ShowValidationError(err)
		return nil, fmt.Errorf("cursor validation failed: %w", err)
	}

	return &cfg, nil
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".cursor-sync")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// Check if user has a config/sync.yaml file in project directory
	wd, _ := os.Getwd()
	projectConfigPath := filepath.Join(wd, "config", "sync.yaml")
	if _, err := os.Stat(projectConfigPath); err == nil {
		// Copy from project config
		data, err := os.ReadFile(projectConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read project config: %w", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to copy project config: %w", err)
		}

		return nil
	}

	config := getDefaultConfig(home)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func setDefaults() {
	// Load defaults from the example config file
	wd, err := os.Getwd()
	if err == nil {
		exampleConfigPath := filepath.Join(wd, "config", "sync.example.yaml")
		if _, err := os.Stat(exampleConfigPath); err == nil {
			viper.SetConfigFile(exampleConfigPath)
			viper.ReadInConfig()
		}
	}
}

func getDefaultConfig(home string) *Config {
	// Load the example config as the default
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	exampleConfigPath := filepath.Join(wd, "config", "sync.example.yaml")
	if _, err := os.Stat(exampleConfigPath); err == nil {
		// Load from example config
		viper.SetConfigFile(exampleConfigPath)
		if err := viper.ReadInConfig(); err == nil {
			var cfg Config
			if err := viper.Unmarshal(&cfg); err == nil {
				// Parse time durations
				parseTimeDurations(&cfg)
				// Expand paths
				expandPaths(&cfg)
				return &cfg
			}
		}
	}

	// Fallback to minimal config if example config fails
	return &Config{
		Repository: Repository{
			URL:       "",
			LocalPath: filepath.Join(home, ".cursor-sync", "settings"),
			Branch:    "main",
		},
		Sync: Sync{
			PullInterval:       5 * time.Minute,
			PushInterval:       5 * time.Minute,
			DebounceTime:       10 * time.Second,
			WatchEnabled:       true,
			ConflictResolve:    "newer",
			HashThrottleDelay:  100 * time.Millisecond,
			HashPollingTimeout: 10 * time.Second,
		},
		Cursor: Cursor{
			ConfigPath:   filepath.Join(home, "Library", "Application Support", "Cursor"),
			ExcludePaths: []string{},
			IncludePaths: []string{},
		},
		Logging: Logging{
			Level:    "info",
			LogDir:   filepath.Join(home, ".cursor-sync", "logs"),
			MaxSize:  10,
			MaxDays:  30,
			Compress: true,
		},
	}
}

func expandPaths(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Expand home directory in paths
	cfg.Repository.LocalPath = expandHome(cfg.Repository.LocalPath, home)
	cfg.Cursor.ConfigPath = expandHome(cfg.Cursor.ConfigPath, home)
	cfg.Logging.LogDir = expandHome(cfg.Logging.LogDir, home)

	return nil
}

func expandHome(path, home string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(home, path[1:])
	}
	return path
}

func validate(cfg *Config) error {
	if cfg.Repository.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	if cfg.Repository.LocalPath == "" {
		return fmt.Errorf("repository local path is required")
	}

	if cfg.Cursor.ConfigPath == "" {
		return fmt.Errorf("cursor config path is required")
	}

	if cfg.Sync.PullInterval <= 0 {
		return fmt.Errorf("pull interval must be positive")
	}

	if cfg.Sync.PushInterval <= 0 {
		return fmt.Errorf("push interval must be positive")
	}

	if cfg.Sync.DebounceTime < 10*time.Second {
		return fmt.Errorf("debounce time must be at least 10 seconds (current: %v)", cfg.Sync.DebounceTime)
	}

	if cfg.Sync.ConflictResolve != "newer" && cfg.Sync.ConflictResolve != "local" && cfg.Sync.ConflictResolve != "remote" {
		return fmt.Errorf("conflict_resolve must be 'newer', 'local', or 'remote'")
	}

	return nil
}

// validateCursorInstallation performs comprehensive Cursor installation validation
func validateCursorInstallation(cfg *Config) error {
	detector := cursor.NewDetector(cfg.Cursor.ConfigPath)
	return detector.DetectAndValidate()
}

// parseTimeDurations manually parses time duration strings from viper
func parseTimeDurations(cfg *Config) error {
	// Parse pull interval
	if pullStr := viper.GetString("sync.pull_interval"); pullStr != "" {
		if duration, err := time.ParseDuration(pullStr); err == nil {
			cfg.Sync.PullInterval = duration
		}
	}

	// Parse push interval
	if pushStr := viper.GetString("sync.push_interval"); pushStr != "" {
		if duration, err := time.ParseDuration(pushStr); err == nil {
			cfg.Sync.PushInterval = duration
		}
	}

	// Parse debounce time
	if debounceStr := viper.GetString("sync.debounce_time"); debounceStr != "" {
		if duration, err := time.ParseDuration(debounceStr); err == nil {
			cfg.Sync.DebounceTime = duration
		}
	}

	// Parse hash throttle delay
	if hashThrottleStr := viper.GetString("sync.hash_throttle_delay"); hashThrottleStr != "" {
		if duration, err := time.ParseDuration(hashThrottleStr); err == nil {
			cfg.Sync.HashThrottleDelay = duration
		}
	}

	// Parse hash polling timeout
	if hashPollingStr := viper.GetString("sync.hash_polling_timeout"); hashPollingStr != "" {
		if duration, err := time.ParseDuration(hashPollingStr); err == nil {
			cfg.Sync.HashPollingTimeout = duration
		}
	}

	return nil
}

// UpdateRepositoryURL updates the repository URL in all configuration files
func UpdateRepositoryURL(repoURL string) error {
	// Update user's config file
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	userConfigPath := filepath.Join(home, ".cursor-sync", "config.yaml")
	if err := updateConfigFileURL(userConfigPath, repoURL); err != nil {
		return fmt.Errorf("failed to update user config: %w", err)
	}

	// Update project config file
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	projectConfigPath := filepath.Join(wd, "config", "sync.yaml")
	if err := updateConfigFileURL(projectConfigPath, repoURL); err != nil {
		return fmt.Errorf("failed to update project config: %w", err)
	}

	return nil
}

// updateConfigFileURL updates the repository URL in a specific config file
func updateConfigFileURL(configPath, repoURL string) error {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // Skip if file doesn't exist
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Replace placeholder URLs with the actual repository URL
	content := string(data)

	// Replace various placeholder patterns
	replacements := []string{
		"https://github.com/your-username/cursor-sync-bucket.git",
		"https://github.com/yourusername/cursor-settings.git",
		"https://github.com/YOUR-USERNAME/cursor-sync-bucket.git",
		"https://github.com/YOUR-USERNAME/cursor-settings.git",
		"REPLACE_WITH_YOUR_USERNAME",
		"REPLACE_WITH_YOUR_REPO",
	}

	for _, placeholder := range replacements {
		if strings.Contains(content, placeholder) {
			content = strings.ReplaceAll(content, placeholder, repoURL)
		}
	}

	// Write the updated content back
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write updated config file: %w", err)
	}

	return nil
}
