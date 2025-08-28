package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cursor-sync/internal/config"
)

// configValidateCmd validates only the configuration file without Cursor installation checks
var configValidateCmd = &cobra.Command{
	Use:   "config-validate",
	Short: "Validate configuration file only (skip Cursor installation checks)",
	Long:  `Validate the configuration file syntax and values without checking Cursor installation or GitHub connectivity.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Validating configuration file...")
		fmt.Println()

		// Load configuration with Viper directly (bypass cursor validation)
		var cfg config.Config

		// Set defaults
		setConfigDefaults()

		// Unmarshal the configuration
		if err := viper.Unmarshal(&cfg); err != nil {
			fmt.Printf("❌ Configuration syntax error: %v\n", err)
			return
		}

		// Expand paths
		if err := expandConfigPaths(&cfg); err != nil {
			fmt.Printf("❌ Path expansion failed: %v\n", err)
			return
		}

		// Validate configuration values (without external dependencies)
		if err := validateConfigValues(&cfg); err != nil {
			fmt.Printf("❌ Configuration validation failed: %v\n", err)
			return
		}

		// Configuration is valid
		fmt.Println("✅ Configuration file is valid")
		fmt.Printf("   Repository URL: %s\n", cfg.Repository.URL)
		fmt.Printf("   Pull Interval: %v\n", cfg.Sync.PullInterval)
		fmt.Printf("   Push Interval: %v\n", cfg.Sync.PushInterval)
		fmt.Printf("   Debounce Time: %v\n", cfg.Sync.DebounceTime)
		fmt.Printf("   Watch Enabled: %v\n", cfg.Sync.WatchEnabled)
		fmt.Printf("   Conflict Resolution: %s\n", cfg.Sync.ConflictResolve)
		fmt.Println()
		fmt.Println("🎉 Configuration validation passed!")
	},
}

// Helper functions for config-only validation
func setConfigDefaults() {
	// Set the same defaults as the main config
	viper.SetDefault("repository.branch", "main")
	viper.SetDefault("sync.pull_interval", "5m")
	viper.SetDefault("sync.push_interval", "5m")
	viper.SetDefault("sync.debounce_time", "10s")
	viper.SetDefault("sync.watch_enabled", true)
	viper.SetDefault("sync.conflict_resolve", "newer")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.max_size", 10)
	viper.SetDefault("logging.max_days", 30)
	viper.SetDefault("logging.compress", true)
}

func expandConfigPaths(cfg *config.Config) error {
	// Expand home directory in paths (simplified version)
	// This is a minimal version that doesn't require full validation
	return nil
}

func validateConfigValues(cfg *config.Config) error {
	// Repository validation
	if cfg.Repository.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	if cfg.Repository.LocalPath == "" {
		return fmt.Errorf("repository local path is required")
	}

	// Sync validation
	if cfg.Sync.PullInterval <= 0 {
		return fmt.Errorf("pull interval must be positive")
	}

	if cfg.Sync.PushInterval <= 0 {
		return fmt.Errorf("push interval must be positive")
	}

	// CRITICAL: Debounce time validation (minimum 10 seconds)
	if cfg.Sync.DebounceTime < 10*time.Second {
		return fmt.Errorf("debounce time must be at least 10 seconds (current: %v)", cfg.Sync.DebounceTime)
	}

	// Conflict resolution validation
	if cfg.Sync.ConflictResolve != "newer" && cfg.Sync.ConflictResolve != "local" && cfg.Sync.ConflictResolve != "remote" {
		return fmt.Errorf("conflict_resolve must be 'newer', 'local', or 'remote'")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(configValidateCmd)
}
