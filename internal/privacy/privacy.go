package privacy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cursor-sync/internal/logger"
)

// RepoInfo represents basic repository information
type RepoInfo struct {
	Private  bool   `json:"private"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// RepositoryChecker checks repository privacy settings
type RepositoryChecker struct {
	httpClient *http.Client
}

// NewRepositoryChecker creates a new repository checker
func NewRepositoryChecker() *RepositoryChecker {
	return &RepositoryChecker{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckRepositoryPrivacy checks if a Git repository is private
func (rc *RepositoryChecker) CheckRepositoryPrivacy(repoURL string) (bool, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		// If we can't parse as GitHub URL, we can't check privacy
		// For safety, assume it might be public and warn
		logger.Warn("Cannot determine repository privacy for URL: %s", repoURL)
		return false, fmt.Errorf("cannot determine repository privacy: %w", err)
	}

	return rc.checkGitHubRepositoryPrivacy(owner, repo)
}

// checkGitHubRepositoryPrivacy checks if a GitHub repository is private
func (rc *RepositoryChecker) checkGitHubRepositoryPrivacy(owner, repo string) (bool, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	logger.Debug("Checking repository privacy: %s/%s", owner, repo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent (GitHub API requires it)
	req.Header.Set("User-Agent", "cursor-sync/1.0")

	// Add GitHub token authentication if available
	if token, err := rc.loadGitHubToken(); err == nil {
		req.Header.Set("Authorization", "token "+token)
		logger.Debug("Using GitHub token for privacy check")
	} else {
		logger.Debug("No GitHub token available for privacy check")
	}

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Repository not found or private (and we don't have access)
		// For safety, we'll assume it's private if we get 404
		logger.Debug("Repository returned 404, assuming private")
		return true, nil
	}

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("GitHub API returned status code %d", resp.StatusCode)
	}

	var repoInfo RepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return false, fmt.Errorf("failed to decode repository info: %w", err)
	}

	logger.Debug("Repository %s/%s is private: %t", owner, repo, repoInfo.Private)
	return repoInfo.Private, nil
}

// loadGitHubToken loads the GitHub token from file
func (rc *RepositoryChecker) loadGitHubToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	tokenPath := filepath.Join(home, ".cursor-sync", ".github")

	// Check if token file exists
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return "", fmt.Errorf("GitHub token not found")
	}

	// Read token from file
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read GitHub token: %w", err)
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("GitHub token is empty")
	}

	return token, nil
}

// parseGitHubURL parses a GitHub repository URL and extracts owner and repo name
func parseGitHubURL(repoURL string) (owner, repo string, err error) {
	// Handle various GitHub URL formats:
	// https://github.com/owner/repo.git
	// https://github.com/owner/repo
	// git@github.com:owner/repo.git
	// github.com/owner/repo

	// Remove common prefixes and suffixes
	url := strings.TrimSpace(repoURL)
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")
	url = strings.TrimSuffix(url, ".git")

	// Replace : with / for SSH format
	url = strings.Replace(url, ":", "/", 1)

	// Extract owner and repo using regex
	re := regexp.MustCompile(`github\.com[:/]([^/]+)/([^/\s]+)`)
	matches := re.FindStringSubmatch(url)

	if len(matches) != 3 {
		return "", "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}

	return matches[1], matches[2], nil
}

// ShowPrivacyWarning displays a prominent privacy warning
func ShowPrivacyWarning(repoURL string) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("⚠️  SECURITY WARNING: PUBLIC REPOSITORY DETECTED!")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nRepository: %s\n", repoURL)
	fmt.Println("\n❌ CURSOR SYNC BLOCKED - This repository appears to be PUBLIC!")
	fmt.Println("\nWhy this matters:")
	fmt.Println("• Cursor settings may contain sensitive information (API keys, tokens)")
	fmt.Println("• Personal configurations and extensions could be exposed")
	fmt.Println("• Workspace paths and project details might be leaked")
	fmt.Println("\n🔒 SOLUTION: Use a PRIVATE repository for syncing Cursor settings")
	fmt.Println("\nTo fix this:")
	fmt.Println("1. Create a new PRIVATE GitHub repository")
	fmt.Println("2. Update config/sync.yaml with the private repository URL")
	fmt.Println("3. Ensure the repository is set to private in GitHub settings")
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println()
}

// ShowPrivacyCheckError displays an error when privacy cannot be determined
func ShowPrivacyCheckError(repoURL string, err error) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("⚠️  WARNING: CANNOT VERIFY REPOSITORY PRIVACY")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nRepository: %s\n", repoURL)
	fmt.Printf("Error: %v\n", err)
	fmt.Println("\n❌ CURSOR SYNC BLOCKED - Cannot verify if repository is private!")
	fmt.Println("\nFor security reasons, cursor-sync only works with verified private repositories.")
	fmt.Println("\n🔒 PLEASE VERIFY:")
	fmt.Println("• Your repository URL is correct")
	fmt.Println("• The repository exists and is set to PRIVATE")
	fmt.Println("• You have network connectivity to GitHub")
	fmt.Println("\nIf you're using a private Git service (GitLab, Bitbucket, etc.),")
	fmt.Println("please ensure your repository is private on that platform.")
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println()
}
