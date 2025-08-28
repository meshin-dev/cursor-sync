package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cursor-sync/internal/config"
	"cursor-sync/internal/logger"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cursor-sync",
	Short: "Sync Cursor IDE settings across machines using Git",
	Long: `Cursor Sync is a tool that automatically synchronizes your Cursor IDE settings,
extensions, keybindings, and other configuration files across multiple machines
using a Git repository with real-time file watching.

Features:
- Real-time file watching and sync
- Automatic conflict resolution based on commit timestamps
- Configurable sync intervals
- Pause/resume functionality
- Comprehensive logging
- macOS LaunchAgent integration`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logger
		logger.Init(verbose)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cursor-sync/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cursor-sync" (without extension)
		configDir := fmt.Sprintf("%s/.cursor-sync", home)
		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		logger.Debug("Using config file: %s", viper.ConfigFileUsed())
	} else {
		// Create default config if none exists
		if err := config.CreateDefaultConfig(); err != nil {
			logger.Error("Failed to create default config: %v", err)
		}
	}
}
