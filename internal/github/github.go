package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cursor-sync/internal/auth"
	"cursor-sync/internal/logger"
)

// GitHubAPI handles GitHub API operations
type GitHubAPI struct {
	token  string
	client *http.Client
}

// RepositoryCreateRequest represents the request body for creating a repository
type RepositoryCreateRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	Private           bool   `json:"private"`
	AutoInit          bool   `json:"auto_init"`
	GitignoreTemplate string `json:"gitignore_template,omitempty"`
}

// RepositoryResponse represents the response from GitHub API
type RepositoryResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	SSHURL      string `json:"ssh_url"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// New creates a new GitHub API client
func New() (*GitHubAPI, error) {
	githubAuth, err := auth.NewGitHubAuth()
	if err != nil {
		return nil, fmt.Errorf("GitHub authentication failed: %w", err)
	}

	return &GitHubAPI{
		token: githubAuth.GetToken(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// CreateRepository creates a new private repository on GitHub
func (g *GitHubAPI) CreateRepository(owner, repoName, description string) (*RepositoryResponse, error) {
	url := fmt.Sprintf("https://api.github.com/user/repos")

	// If owner is specified and different from authenticated user, use org endpoint
	if owner != "" {
		// Check if it's an organization
		if g.isOrganization(owner) {
			url = fmt.Sprintf("https://api.github.com/orgs/%s/repos", owner)
		} else {
			// For user repositories, we'll use the user endpoint
			// GitHub will create it under the authenticated user
			url = "https://api.github.com/user/repos"
		}
	}

	requestBody := RepositoryCreateRequest{
		Name:              repoName,
		Description:       description,
		Private:           true,   // Always create as private for security
		AutoInit:          true,   // Initialize with README
		GitignoreTemplate: "Node", // Add .gitignore for Node.js projects
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		var repo RepositoryResponse
		if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &repo, nil
	}

	// Handle different error cases
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("GitHub token is invalid or expired")
	case http.StatusForbidden:
		return nil, fmt.Errorf("insufficient permissions to create repository")
	case http.StatusUnprocessableEntity:
		return nil, fmt.Errorf("repository name is invalid or already exists")
	case http.StatusNotFound:
		return nil, fmt.Errorf("organization not found or you don't have access")
	default:
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}
}

// RepositoryExists checks if a repository exists
func (g *GitHubAPI) RepositoryExists(owner, repoName string) (bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusUnauthorized:
		return false, fmt.Errorf("GitHub token is invalid or expired")
	case http.StatusForbidden:
		return false, fmt.Errorf("insufficient permissions to access repository")
	default:
		return false, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}
}

// isOrganization checks if the given name is an organization
func (g *GitHubAPI) isOrganization(name string) bool {
	url := fmt.Sprintf("https://api.github.com/orgs/%s", name)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// WaitForRepositoryReady waits for the repository to be ready after creation
// GitHub sometimes takes a few seconds to fully initialize a new repository
func (g *GitHubAPI) WaitForRepositoryReady(owner, repoName string, maxWait time.Duration) error {
	logger.Info("⏳ Waiting for repository to be ready...")

	startTime := time.Now()
	checkInterval := 2 * time.Second

	for time.Since(startTime) < maxWait {
		exists, err := g.RepositoryExists(owner, repoName)
		if err != nil {
			logger.Debug("Repository check failed: %v", err)
			time.Sleep(checkInterval)
			continue
		}

		if exists {
			logger.Info("✅ Repository is ready!")
			return nil
		}

		logger.Debug("Repository not ready yet, waiting...")
		time.Sleep(checkInterval)
	}

	return fmt.Errorf("repository not ready after %v", maxWait)
}
