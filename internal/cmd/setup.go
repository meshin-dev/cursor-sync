package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"cursor-sync/internal/interactive"
	"cursor-sync/internal/logger"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run interactive setup wizard for cursor-sync configuration",
	Long: `Run the interactive setup wizard to configure cursor-sync with all required settings.

This wizard will guide you through:
- Configuring your IDE installation path (Cursor, VS Code, or custom)
- Setting up your GitHub Personal Access Token
- Configuring your Git repository for settings storage (cursor-sync-bucket recommended)
- Validating repository privacy and accessibility  
- Creating necessary configuration files

The setup wizard is also automatically triggered when required settings are missing.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Starting interactive setup wizard...")

		wizard := interactive.NewSetupWizard()
		if err := wizard.RunInteractiveSetup(); err != nil {
			fmt.Printf("❌ Setup failed: %v\n", err)
			logger.Error("Interactive setup failed: %v", err)
			return
		}

		fmt.Println("🎉 Interactive setup completed successfully!")
		logger.Info("Interactive setup completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
