package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"cursor-sync/internal/config"
	"cursor-sync/internal/logger"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo <repository-url>",
	Short: "Update repository URL in all configuration files",
	Long: `Update the repository URL in all configuration files.

This command will automatically update both the user's config file (~/.cursor-sync/config.yaml)
and the project's config file (config/sync.yaml) with the provided repository URL.

Examples:
  cursor-sync repo https://github.com/username/cursor-settings.git
  cursor-sync repo git@github.com:username/cursor-settings.git`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := strings.TrimSpace(args[0])

		if repoURL == "" {
			logger.Fatal("Repository URL cannot be empty")
		}

		fmt.Printf("🔄 Updating repository URL to: %s\n", repoURL)

		if err := config.UpdateRepositoryURL(repoURL); err != nil {
			logger.Fatal("Failed to update repository URL: %v", err)
		}

		fmt.Println("✅ Repository URL updated successfully in all configuration files!")
		fmt.Println("🚀 You can now run 'cursor-sync sync' to start syncing")
	},
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
