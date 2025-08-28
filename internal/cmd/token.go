package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"cursor-sync/internal/auth"
	"cursor-sync/internal/logger"
)

// tokenCmd represents the token command
var tokenCmd = &cobra.Command{
	Use:   "token <github-token>",
	Short: "Set GitHub Personal Access Token for repository authentication",
	Long: `Set the GitHub Personal Access Token (PAT) required for secure repository access.

The token is stored securely in ~/.cursor-sync/.github and used for all Git operations.

To create a GitHub token:
1. Go to GitHub → Settings → Developer settings → Personal access tokens
2. Click 'Generate new token (classic)'
3. Select scopes: 'repo' (Full control of private repositories)
4. Copy the generated token

Token format should start with: ghp_ or github_pat_

Example:
  cursor-sync token ghp_1234567890abcdef1234567890abcdef12345678`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := strings.TrimSpace(args[0])

		if err := auth.SaveGitHubToken(token); err != nil {
			logger.Fatal("Failed to save GitHub token: %v", err)
		}

		fmt.Println("✅ GitHub token saved successfully!")
		fmt.Println("🔒 Token stored securely in ~/.cursor-sync/.github")
		fmt.Println("🚀 You can now use cursor-sync with your private repositories")

		// Verify the token works
		fmt.Println("\n🔍 Verifying token...")
		if _, err := auth.NewGitHubAuth(); err != nil {
			logger.Error("Token verification failed: %v", err)
			fmt.Println("❌ Token verification failed - please check your token")
		} else {
			fmt.Println("✅ Token verified successfully!")
		}
	},
}

// tokenShowCmd represents the token show command
var tokenShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show GitHub token status",
	Long:  "Display the current GitHub token status and user information",
	Run: func(cmd *cobra.Command, args []string) {
		if !auth.HasValidToken() {
			fmt.Println("❌ No GitHub token found")
			auth.ShowTokenRequiredMessage()
			return
		}

		githubAuth, err := auth.NewGitHubAuth()
		if err != nil {
			fmt.Printf("❌ Token verification failed: %v\n", err)
			return
		}

		// Show token info (masked)
		token := githubAuth.GetToken()
		if len(token) > 8 {
			maskedToken := token[:8] + strings.Repeat("*", len(token)-8)
			fmt.Printf("✅ GitHub token: %s\n", maskedToken)
		}

		fmt.Println("🔒 Token file: ~/.cursor-sync/.github")
		fmt.Println("✅ Authentication verified")
	},
}

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.AddCommand(tokenShowCmd)
}
