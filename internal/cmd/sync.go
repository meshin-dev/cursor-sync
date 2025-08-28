package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"cursor-sync/internal/config"
	"cursor-sync/internal/logger"
	"cursor-sync/internal/sync"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Perform a manual sync operation",
	Long: `Manually trigger a sync operation between local and remote repositories.

This command performs a full sync sequence:
1. Pull changes from remote repository
2. Push any local changes to remote repository

This is useful for:
- Testing sync functionality
- Forcing a sync outside of normal intervals
- Troubleshooting sync issues`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Starting manual sync operation...")

		cfg, err := config.Load()
		if err != nil {
			logger.Fatal("Failed to load configuration: %v", err)
		}

		// Create syncer instance
		syncer, err := sync.New(cfg)
		if err != nil {
			logger.Fatal("Failed to create syncer: %v", err)
		}

		// Initialize syncer
		if err := syncer.Initialize(); err != nil {
			logger.Fatal("Failed to initialize syncer: %v", err)
		}

		fmt.Println("🔄 Performing manual sync...")

		// Perform pull sync
		fmt.Println("📥 Pulling remote changes...")
		if err := syncer.SyncFromRemote(); err != nil {
			logger.Error("Failed to pull remote changes: %v", err)
			fmt.Println("❌ Pull sync failed")
		} else {
			fmt.Println("✅ Remote changes pulled successfully")
		}

		// Perform push sync
		fmt.Println("📤 Pushing local changes...")
		if err := syncer.SyncToRemote(); err != nil {
			logger.Error("Failed to push local changes: %v", err)
			fmt.Println("❌ Push sync failed")
		} else {
			fmt.Println("✅ Local changes pushed successfully")
		}

		fmt.Println("🎉 Manual sync completed")
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
