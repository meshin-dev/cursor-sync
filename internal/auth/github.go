package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"

	"cursor-sync/internal/logger"
)

const (
	GitHubTokenFile = ".github"
)

// GitHubAuth handles GitHub authentication
type GitHubAuth struct {
	token  string
	client *github.Client
}

// NewGitHubAuth creates a new GitHub authentication handler
func NewGitHubAuth() (*GitHubAuth, error) {
	token, err := loadGitHubToken()
	if err != nil {
		return nil, err
	}

	// Create OAuth2 client with token
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	auth := &GitHubAuth{
		token:  token,
		client: client,
	}

	// Verify token works
	if err := auth.verifyToken(); err != nil {
		return nil, fmt.Errorf("GitHub token verification failed: %w", err)
	}

	return auth, nil
}

// GetClient returns the authenticated GitHub client
func (ga *GitHubAuth) GetClient() *github.Client {
	return ga.client
}

// GetToken returns the GitHub token
func (ga *GitHubAuth) GetToken() string {
	return ga.token
}

// verifyToken verifies the GitHub token is valid
func (ga *GitHubAuth) verifyToken() error {
	ctx := context.Background()

	// Get authenticated user to verify token
	user, resp, err := ga.client.Users.Get(ctx, "")
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			return fmt.Errorf("invalid GitHub token - please check your token in ~/.cursor-sync/.github")
		}
		return fmt.Errorf("failed to verify GitHub token: %w", err)
	}

	logger.Info("✅ GitHub token verified for user: %s", user.GetLogin())
	return nil
}

// loadGitHubToken loads the GitHub token from file
func loadGitHubToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	tokenPath := filepath.Join(home, ".cursor-sync", GitHubTokenFile)

	// Check if token file exists
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return "", fmt.Errorf("GitHub token not found. Please create %s with your GitHub Personal Access Token", tokenPath)
	}

	// Read token from file
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read GitHub token: %w", err)
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("GitHub token is empty. Please add your token to %s", tokenPath)
	}

	// Basic token format validation
	if !isValidGitHubTokenFormat(token) {
		return "", fmt.Errorf("invalid GitHub token format. Expected format: ghp_... or github_pat_...")
	}

	logger.Debug("GitHub token loaded from %s", tokenPath)
	return token, nil
}

// SaveGitHubToken saves a GitHub token to the token file
func SaveGitHubToken(token string) error {
	if !isValidGitHubTokenFormat(token) {
		return fmt.Errorf("invalid GitHub token format")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create .cursor-sync directory if it doesn't exist
	cursorSyncDir := filepath.Join(home, ".cursor-sync")
	if err := os.MkdirAll(cursorSyncDir, 0700); err != nil {
		return fmt.Errorf("failed to create .cursor-sync directory: %w", err)
	}

	tokenPath := filepath.Join(cursorSyncDir, GitHubTokenFile)

	// Write token to file with restricted permissions
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to write GitHub token: %w", err)
	}

	logger.Info("GitHub token saved to %s", tokenPath)
	return nil
}

// isValidGitHubTokenFormat checks if the token format looks like a GitHub token
func isValidGitHubTokenFormat(token string) bool {
	// GitHub personal access tokens start with ghp_ or github_pat_
	return strings.HasPrefix(token, "ghp_") ||
		strings.HasPrefix(token, "github_pat_") ||
		strings.HasPrefix(token, "gho_") || // GitHub App tokens
		strings.HasPrefix(token, "ghu_") || // GitHub App user tokens
		strings.HasPrefix(token, "ghs_") // GitHub App installation tokens
}

// HasValidToken checks if a valid GitHub token exists
func HasValidToken() bool {
	_, err := loadGitHubToken()
	return err == nil
}

// ShowTokenRequiredMessage displays instructions for setting up GitHub token
func ShowTokenRequiredMessage() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🔑 GITHUB TOKEN REQUIRED")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\ncursor-sync requires a GitHub Personal Access Token for secure repository access.")
	fmt.Println("\nTo create a GitHub token:")
	fmt.Println("1. Go to GitHub → Settings → Developer settings → Personal access tokens")
	fmt.Println("2. Click 'Generate new token (classic)'")
	fmt.Println("3. Select scopes: 'repo' (Full control of private repositories)")
	fmt.Println("4. Copy the generated token")
	fmt.Println("\nTo configure the token:")
	home, _ := os.UserHomeDir()
	tokenPath := filepath.Join(home, ".cursor-sync", GitHubTokenFile)
	fmt.Printf("5. Save your token to: %s\n", tokenPath)
	fmt.Printf("   echo 'your_token_here' > %s\n", tokenPath)
	fmt.Printf("   chmod 600 %s\n", tokenPath)
	fmt.Println("\nToken format should start with: ghp_ or github_pat_")
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println()
}
