package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"cursor-sync/internal/installer"
	"cursor-sync/internal/logger"
)

var (
	repoURL string
	force   bool
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and configure cursor-sync",
	Long: `Install cursor-sync and configure it to run automatically as a macOS LaunchAgent.

BEFORE INSTALLATION:
1. Copy config/sync.example.yaml to config/sync.yaml
2. Edit config/sync.yaml and replace the repository URL with your private Git repository

This command will:
- Use your config/sync.yaml settings
- Create necessary configuration files
- Set up macOS LaunchAgent for automatic startup
- Perform initial sync from remote repository

Example:
  cp config/sync.example.yaml config/sync.yaml
  # Edit config/sync.yaml with your repository URL
  cursor-sync install`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if config/sync.yaml exists
		wd, err := os.Getwd()
		if err != nil {
			logger.Fatal("Failed to get working directory: %v", err)
		}

		configPath := filepath.Join(wd, "config", "sync.yaml")
		if _, err := os.Stat(configPath); err != nil {
			logger.Fatal("❌ Configuration file not found!\n\nPlease follow these steps:\n1. cp config/sync.example.yaml config/sync.yaml\n2. Edit config/sync.yaml and replace the repository URL\n3. Run 'cursor-sync install' again")
		}

		logger.Info("Installing cursor-sync using config/sync.yaml")

		installer := installer.New("", force) // Empty repo URL, will read from config

		if err := installer.Install(); err != nil {
			logger.Fatal("Installation failed: %v", err)
		}

		fmt.Println("✅ Cursor Sync installed successfully!")
		fmt.Println("📂 Configuration loaded from: config/sync.yaml")
		fmt.Println("🚀 Daemon will start automatically on login")
		fmt.Println("📋 Use 'cursor-sync status' to check daemon status")
		fmt.Println("⏸️  Use 'cursor-sync pause' to temporarily stop syncing")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVarP(&force, "force", "f", false, "Force installation even if already configured")
}
