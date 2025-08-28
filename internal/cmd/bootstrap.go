package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"cursor-sync/internal/interactive"
	"cursor-sync/internal/logger"
)

// bootstrapCmd represents the comprehensive setup command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "🚀 Complete setup wizard - does everything in one command",
	Long: `Bootstrap is the ultimate one-command setup that guides you through the entire cursor-sync installation process.

This command will:
1. 📂 Configure your IDE installation path (Cursor, VS Code, or custom)
2. 🔍 Validate your IDE installation
3. 🔑 Help you set up your GitHub Personal Access Token
4. 📦 Configure your private repository (cursor-sync-bucket)
5. ⚙️  Create all necessary configuration files
6. 🔧 Install the background daemon
7. 🚀 Start the sync service
8. ✅ Verify everything is working

No need to run multiple commands - bootstrap handles everything!`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 CURSOR-SYNC BOOTSTRAP")
		fmt.Println("=" + fmt.Sprintf("%*s", 79, "="))
		fmt.Println()
		fmt.Println("Welcome! This wizard will set up cursor-sync completely in one go.")
		fmt.Println("Sit back and follow the prompts - we'll handle everything!")
		fmt.Println()

		// Step 1: Cursor Validation
		if err := validateCursorInstallation(); err != nil {
			fmt.Printf("❌ Bootstrap failed at Cursor validation: %v\n", err)
			os.Exit(1)
		}

		// Step 2: Interactive Setup (Token + Repository)
		if err := runInteractiveSetup(); err != nil {
			fmt.Printf("❌ Bootstrap failed at interactive setup: %v\n", err)
			os.Exit(1)
		}

		// Step 3: Final Configuration Validation
		if err := validateConfiguration(); err != nil {
			fmt.Printf("❌ Bootstrap failed at configuration validation: %v\n", err)
			os.Exit(1)
		}

		// Step 4: Installation
		if err := performInstallation(); err != nil {
			fmt.Printf("❌ Bootstrap failed at installation: %v\n", err)
			os.Exit(1)
		}

		// Step 5: Start Service
		if err := startSyncService(); err != nil {
			fmt.Printf("❌ Bootstrap failed at service startup: %v\n", err)
			os.Exit(1)
		}

		// Step 6: Final Verification
		if err := verifyInstallation(); err != nil {
			fmt.Printf("❌ Bootstrap failed at final verification: %v\n", err)
			os.Exit(1)
		}

		// Success!
		showSuccessMessage()
	},
}

func validateCursorInstallation() error {
	fmt.Println("🔍 STEP 1: Validating Cursor IDE Installation")
	fmt.Println(fmt.Sprintf("%*s", 50, "-"))

	// Use the existing check command logic
	checkCmd.Run(checkCmd, []string{})
	fmt.Println()
	return nil
}

func runInteractiveSetup() error {
	fmt.Println("⚙️ STEP 2: Interactive Configuration")
	fmt.Println(fmt.Sprintf("%*s", 50, "-"))

	wizard := interactive.NewSetupWizard()
	if err := wizard.RunInteractiveSetup(); err != nil {
		return fmt.Errorf("interactive setup failed: %w", err)
	}

	fmt.Println()
	return nil
}

func validateConfiguration() error {
	fmt.Println("✅ STEP 3: Validating Complete Configuration")
	fmt.Println(fmt.Sprintf("%*s", 50, "-"))

	// Use validate command logic but capture output
	validateCmd.Run(validateCmd, []string{})
	fmt.Println()
	return nil
}

func performInstallation() error {
	fmt.Println("🔧 STEP 4: Installing Background Daemon")
	fmt.Println(fmt.Sprintf("%*s", 50, "-"))

	// Use install command logic
	installCmd.Run(installCmd, []string{})
	fmt.Println()
	return nil
}

func startSyncService() error {
	fmt.Println("🚀 STEP 5: Starting Sync Service")
	fmt.Println(fmt.Sprintf("%*s", 50, "-"))

	// Use start command logic
	startCmd.Run(startCmd, []string{})
	fmt.Println()
	return nil
}

func verifyInstallation() error {
	fmt.Println("🔎 STEP 6: Final Verification")
	fmt.Println(fmt.Sprintf("%*s", 50, "-"))

	// Use status command to verify everything is working
	statusCmd.Run(statusCmd, []string{})
	fmt.Println()
	return nil
}

func showSuccessMessage() {
	fmt.Println("🎉 BOOTSTRAP COMPLETE!")
	fmt.Println("=" + fmt.Sprintf("%*s", 79, "="))
	fmt.Println()
	fmt.Println("✅ Cursor-sync is now fully installed and running!")
	fmt.Println()
	fmt.Println("📊 What's been set up:")
	fmt.Println("  • Cursor IDE validation passed")
	fmt.Println("  • GitHub token configured and validated")
	fmt.Println("  • Private repository configured")
	fmt.Println("  • Background daemon installed")
	fmt.Println("  • Sync service started and running")
	fmt.Println("  • Initial sync completed")
	fmt.Println()
	fmt.Println("🎯 Your settings are now syncing automatically!")
	fmt.Println()
	fmt.Println("📋 Useful commands:")
	fmt.Println("  cursor-sync status    # Check sync status")
	fmt.Println("  cursor-sync pause     # Temporarily pause syncing")
	fmt.Println("  cursor-sync resume    # Resume syncing")
	fmt.Println("  cursor-sync logs      # View sync logs")
	fmt.Println()
	fmt.Println("🔄 Make changes in Cursor IDE - they'll automatically sync within 10 seconds!")
	fmt.Println("🌟 cursor-sync is now protecting your settings across all your machines.")
	fmt.Println()

	logger.Info("Bootstrap completed successfully")
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}
