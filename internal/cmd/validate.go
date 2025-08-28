package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"cursor-sync/internal/config"
	"cursor-sync/internal/cursor"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate cursor-sync configuration and Cursor installation",
	Long: `Validate that cursor-sync is properly configured and that Cursor IDE is installed and accessible.

This command checks:
- Configuration file validity
- Cursor IDE installation and directory structure  
- Required settings files and directories
- Repository configuration (if provided)`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Validating cursor-sync configuration and Cursor installation...")
		fmt.Println()

		// Load and validate configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("❌ Configuration validation failed: %v\n", err)
			return
		}

		// Configuration loaded successfully
		fmt.Println("✅ Configuration loaded successfully")
		fmt.Printf("   Repository: %s\n", cfg.Repository.URL)
		fmt.Printf("   Branch: %s\n", cfg.Repository.Branch)
		fmt.Printf("   Local Path: %s\n", cfg.Repository.LocalPath)
		fmt.Printf("   Cursor Path: %s\n", cfg.Cursor.ConfigPath)
		fmt.Println()

		// Cursor validation already happened during config.Load(),
		// so if we get here, everything is valid
		fmt.Println("✅ Cursor IDE installation validated")
		fmt.Printf("   Settings Directory: %s\n", cfg.Cursor.ConfigPath)
		fmt.Printf("   Pull Interval: %v\n", cfg.Sync.PullInterval)
		fmt.Printf("   Push Interval: %v\n", cfg.Sync.PushInterval)
		fmt.Printf("   Debounce Time: %v\n", cfg.Sync.DebounceTime)
		fmt.Printf("   Watch Enabled: %v\n", cfg.Sync.WatchEnabled)
		fmt.Printf("   Conflict Resolution: %s\n", cfg.Sync.ConflictResolve)
		fmt.Println()

		fmt.Println("🎉 All validations passed! cursor-sync is ready to use.")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Set your GitHub token: cursor-sync token <your-token>")
		fmt.Println("2. Install the daemon: cursor-sync install")
		fmt.Println("3. Start syncing: cursor-sync start")
	},
}

// checkCmd represents the check command (alias for validate with different output format)
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Quick health check of Cursor installation",
	Long:  `Perform a quick health check to verify Cursor IDE is installed and accessible.`,
	Run: func(cmd *cobra.Command, args []string) {
		detector := cursor.NewDetector(cursor.GetDefaultCursorPath())

		fmt.Print("🔍 Checking Cursor installation... ")

		if err := detector.DetectAndValidate(); err != nil {
			fmt.Println("❌")
			fmt.Printf("\nCursor check failed: %v\n", err)
			return
		}

		fmt.Println("✅")
		fmt.Printf("Cursor IDE found at: %s\n", cursor.GetDefaultCursorPath())
		fmt.Println("Ready for synchronization!")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(checkCmd)
}
